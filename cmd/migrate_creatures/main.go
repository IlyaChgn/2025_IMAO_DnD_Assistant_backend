package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Command line flags
var (
	dryRun     = flag.Bool("dry-run", true, "Run without writing to database")
	limit      = flag.Int("limit", 0, "Limit number of creatures to process (0 = all)")
	verbose    = flag.Bool("verbose", false, "Print detailed output for each creature")
	mongoURI   = flag.String("mongo", "", "MongoDB connection URI (or set MONGODB env var)")
	creatureID = flag.String("id", "", "Process single creature by ID")
)

func main() {
	flag.Parse()

	uri := *mongoURI
	if uri == "" {
		uri = os.Getenv("MONGODB")
	}
	if uri == "" {
		log.Fatal("MongoDB URI required: use -mongo flag or MONGODB env var")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Connect error:", err)
	}
	defer client.Disconnect(ctx)

	coll := client.Database("bestiary_db").Collection("creatures")

	// Build query
	filter := bson.M{}
	if *creatureID != "" {
		oid, err := primitive.ObjectIDFromHex(*creatureID)
		if err != nil {
			log.Fatal("Invalid creature ID:", err)
		}
		filter["_id"] = oid
	}

	// Find options
	findOpts := options.Find()
	if *limit > 0 {
		findOpts.SetLimit(int64(*limit))
	}

	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		log.Fatal("Find error:", err)
	}
	defer cursor.Close(ctx)

	stats := &MigrationStats{}

	for cursor.Next(ctx) {
		var creature bson.M
		if err := cursor.Decode(&creature); err != nil {
			log.Printf("Decode error: %v", err)
			stats.Errors++
			continue
		}

		result := migrateCreature(creature)
		stats.Total++

		if result.HasChanges() {
			stats.Modified++
		}

		// Track skipped creatures
		creatureName := ""
		if name, ok := creature["name"].(bson.M); ok {
			creatureName, _ = name["rus"].(string)
		}

		if result.MovementSkipped {
			stats.MovementSkipped++
			stats.SkippedCreatures = append(stats.SkippedCreatures, SkippedInfo{
				Name:   creatureName,
				Reason: "Movement: " + result.MovementReason,
			})
		}
		if result.ActionsSkipped {
			stats.ActionsSkipped++
			stats.SkippedCreatures = append(stats.SkippedCreatures, SkippedInfo{
				Name:   creatureName,
				Reason: "Actions: " + result.ActionsReason,
			})
		}

		if *verbose || (*creatureID != "") {
			printResult(creature, result)
		}

		// Write to database if not dry run
		if !*dryRun && result.HasChanges() {
			update := bson.M{"$set": result.Updates}
			_, err := coll.UpdateByID(ctx, creature["_id"], update)
			if err != nil {
				log.Printf("Update error for %v: %v", creature["_id"], err)
				stats.Errors++
			} else {
				stats.Written++
			}
		}
	}

	printStats(stats)

	if *dryRun {
		fmt.Println("\n⚠️  DRY RUN - no changes written. Use -dry-run=false to apply.")
	}
}

// MigrationStats tracks migration progress
type MigrationStats struct {
	Total            int
	Modified         int
	Written          int
	Errors           int
	MovementSkipped  int
	ActionsSkipped   int
	SkippedCreatures []SkippedInfo
}

type SkippedInfo struct {
	Name   string
	Reason string
}

// MigrationResult contains the result of migrating a single creature
type MigrationResult struct {
	Updates           bson.M
	Movement          *Movement
	Vision            *Vision
	StructuredActions []StructuredAction
	Multiattacks      []MultiattackGroup
	MovementSkipped   bool
	MovementReason    string
	ActionsSkipped    bool
	ActionsReason     string
}

func (r *MigrationResult) HasChanges() bool {
	return len(r.Updates) > 0
}

// Target structs (matching internal/models but simplified for migration)
type Movement struct {
	Walk   int  `json:"walk,omitempty" bson:"walk,omitempty"`
	Fly    int  `json:"fly,omitempty" bson:"fly,omitempty"`
	Swim   int  `json:"swim,omitempty" bson:"swim,omitempty"`
	Climb  int  `json:"climb,omitempty" bson:"climb,omitempty"`
	Burrow int  `json:"burrow,omitempty" bson:"burrow,omitempty"`
	Hover  bool `json:"hover,omitempty" bson:"hover,omitempty"`
}

type Vision struct {
	Darkvision  int `json:"darkvision,omitempty" bson:"darkvision,omitempty"`
	Blindsight  int `json:"blindsight,omitempty" bson:"blindsight,omitempty"`
	Truesight   int `json:"truesight,omitempty" bson:"truesight,omitempty"`
	Tremorsense int `json:"tremorsense,omitempty" bson:"tremorsense,omitempty"`
}

type StructuredAction struct {
	ID          string        `json:"id" bson:"id"`
	Name        string        `json:"name" bson:"name"`
	Description string        `json:"description" bson:"description"`
	Category    string        `json:"category" bson:"category"`
	Attack      *AttackData   `json:"attack,omitempty" bson:"attack,omitempty"`
	SavingThrow *SaveData     `json:"savingThrow,omitempty" bson:"savingThrow,omitempty"`
	Recharge    *RechargeData `json:"recharge,omitempty" bson:"recharge,omitempty"`
	Effects     []Effect      `json:"effects,omitempty" bson:"effects,omitempty"`
}

type AttackData struct {
	Type    string       `json:"type" bson:"type"`
	Bonus   int          `json:"bonus" bson:"bonus"`
	Reach   int          `json:"reach,omitempty" bson:"reach,omitempty"`
	Range   *RangeData   `json:"range,omitempty" bson:"range,omitempty"`
	Targets int          `json:"targets" bson:"targets"`
	Damage  []DamageRoll `json:"damage" bson:"damage"`
}

type RangeData struct {
	Normal int `json:"normal" bson:"normal"`
	Long   int `json:"long,omitempty" bson:"long,omitempty"`
}

type DamageRoll struct {
	DiceCount  int    `json:"diceCount" bson:"diceCount"`
	DiceType   string `json:"diceType" bson:"diceType"`
	Bonus      int    `json:"bonus,omitempty" bson:"bonus,omitempty"`
	DamageType string `json:"damageType" bson:"damageType"`
}

type SaveData struct {
	Ability   string `json:"ability" bson:"ability"`
	DC        int    `json:"dc" bson:"dc"`
	OnFail    string `json:"onFail" bson:"onFail"`
	OnSuccess string `json:"onSuccess" bson:"onSuccess"`
}

type RechargeData struct {
	MinRoll int `json:"minRoll" bson:"minRoll"`
}

type Effect struct {
	Condition   *ConditionEffect `json:"condition,omitempty" bson:"condition,omitempty"`
	Description string           `json:"description,omitempty" bson:"description,omitempty"`
}

type ConditionEffect struct {
	Condition string `json:"condition" bson:"condition"`
	Duration  string `json:"duration,omitempty" bson:"duration,omitempty"`
	EscapeDC  int    `json:"escapeDC,omitempty" bson:"escapeDC,omitempty"`
}

type MultiattackGroup struct {
	ID      string             `json:"id" bson:"id"`
	Name    string             `json:"name" bson:"name"`
	Actions []MultiattackEntry `json:"actions" bson:"actions"`
}

type MultiattackEntry struct {
	ActionID string `json:"actionId" bson:"actionId"`
	Count    int    `json:"count" bson:"count"`
}

// migrateCreature processes a single creature document
func migrateCreature(creature bson.M) *MigrationResult {
	result := &MigrationResult{
		Updates: bson.M{},
	}

	// 1. Migrate Speed -> Movement
	if speeds, ok := creature["speed"].(bson.A); ok {
		movement, skipped, reason := convertSpeed(speeds)
		if skipped {
			result.MovementSkipped = true
			result.MovementReason = reason
		} else if movement != nil && movement.Walk > 0 {
			result.Movement = movement
			result.Updates["movement"] = movement
		}
	}

	// 2. Migrate Senses -> Vision
	if senses, ok := creature["senses"].(bson.M); ok {
		vision := convertSenses(senses)
		if vision != nil {
			result.Vision = vision
			result.Updates["vision"] = vision
		}
	}

	// 3. Migrate llm_parsed_attack -> StructuredActions
	if attacks, ok := creature["llm_parsed_attack"].(bson.A); ok && len(attacks) > 0 {
		// Get original actions for descriptions
		var originalActions bson.A
		if actions, ok := creature["actions"].(bson.A); ok {
			originalActions = actions
		}

		structuredActions, multiattacks := convertAttacks(attacks, originalActions)
		if len(structuredActions) > 0 {
			result.StructuredActions = structuredActions
			result.Updates["structuredActions"] = structuredActions
		}
		if len(multiattacks) > 0 {
			result.Multiattacks = multiattacks
			result.Updates["multiattacks"] = multiattacks
		}
	} else {
		// Check if has actions but no llm_parsed_attack
		if actions, ok := creature["actions"].(bson.A); ok && len(actions) > 0 {
			result.ActionsSkipped = true
			result.ActionsReason = "no llm_parsed_attack"
		}
	}

	return result
}

// convertSpeed converts legacy Speed array to Movement struct
func convertSpeed(speeds bson.A) (*Movement, bool, string) {
	movement := &Movement{}

	for _, s := range speeds {
		speed, ok := s.(bson.M)
		if !ok {
			continue
		}

		// Get value
		var value int
		switch v := speed["value"].(type) {
		case int32:
			value = int(v)
		case int64:
			value = int(v)
		case int:
			value = v
		case float64:
			value = int(v)
		default:
			continue
		}

		// Get name
		name := ""
		if n, ok := speed["name"].(string); ok {
			name = strings.TrimSpace(n)
		}

		// Check for complex strings that we skip
		if isComplexSpeedName(name) {
			return nil, true, fmt.Sprintf("complex speed name: %q", name)
		}

		// Map name to field
		switch {
		case name == "" || name == "ходьба":
			movement.Walk = value
		case name == "летая":
			movement.Fly = value
		case name == "плавая":
			movement.Swim = value
		case name == "лазая":
			movement.Climb = value
		case name == "копая":
			movement.Burrow = value
		default:
			// Unknown speed type - skip this creature
			return nil, true, fmt.Sprintf("unknown speed name: %q", name)
		}

		// Check for hover
		if additional, ok := speed["additional"].(string); ok {
			if strings.Contains(strings.ToLower(additional), "парит") {
				movement.Hover = true
			}
		}
	}

	return movement, false, ""
}

// isComplexSpeedName checks if speed name contains complex form-dependent text
func isComplexSpeedName(name string) bool {
	if name == "" {
		return false
	}

	complexPatterns := []string{
		"фт.", "футов", "форм", "облик", "гибрид",
		"только", "когда", "вертикальн",
	}

	nameLower := strings.ToLower(name)
	for _, pattern := range complexPatterns {
		if strings.Contains(nameLower, pattern) {
			return true
		}
	}

	return false
}

// convertSenses converts legacy Senses to Vision struct
func convertSenses(senses bson.M) *Vision {
	vision := &Vision{}

	sensesArray, ok := senses["senses"].(bson.A)
	if !ok {
		return vision // Return empty vision
	}

	for _, s := range sensesArray {
		sense, ok := s.(bson.M)
		if !ok {
			continue
		}

		name, _ := sense["name"].(string)
		var value int
		switch v := sense["value"].(type) {
		case int32:
			value = int(v)
		case int64:
			value = int(v)
		case int:
			value = v
		case float64:
			value = int(v)
		}

		nameLower := strings.ToLower(name)
		switch {
		case strings.Contains(nameLower, "тёмное зрение") || strings.Contains(nameLower, "темное зрение"):
			vision.Darkvision = value
		case strings.Contains(nameLower, "слепое зрение"):
			vision.Blindsight = value
		case strings.Contains(nameLower, "истинное зрение"):
			vision.Truesight = value
		case strings.Contains(nameLower, "чувство вибрации"):
			vision.Tremorsense = value
		}
	}

	return vision
}

// convertAttacks converts llm_parsed_attack to StructuredActions and MultiattackGroups.
func convertAttacks(attacks bson.A, originalActions bson.A) ([]StructuredAction, []MultiattackGroup) {
	var result []StructuredAction

	// Build map of original action descriptions
	descMap := make(map[string]string)
	for _, a := range originalActions {
		if action, ok := a.(bson.M); ok {
			name, _ := action["name"].(string)
			value, _ := action["value"].(string)
			descMap[name] = value
		}
	}

	// Track used IDs to prevent collisions
	usedIDs := make(map[string]int)

	for _, a := range attacks {
		attack, ok := a.(bson.M)
		if !ok {
			continue
		}

		name, _ := attack["name"].(string)
		if name == "" {
			continue
		}

		// Skip multiattack containers — they are processed in extractMultiattacks
		if subAttacks, ok := attack["attacks"].(bson.A); ok && len(subAttacks) > 0 {
			continue
		}

		// Parse recharge from name
		cleanName, recharge := parseRecharge(name)

		id := generateID(cleanName)
		if count, exists := usedIDs[id]; exists {
			id = fmt.Sprintf("%s-%d", id, count+1)
		}
		usedIDs[id]++

		sa := StructuredAction{
			ID:          id,
			Name:        cleanName,
			Description: descMap[name], // Original HTML
			Category:    "action",
		}

		if recharge != nil {
			sa.Recharge = recharge
		}

		// Determine if it's an attack roll or saving throw
		attackType, _ := attack["type"].(string)

		if attackType == "area" || hasOnlySaveDC(attack) {
			// Saving throw based action
			sa.SavingThrow = convertToSaveData(attack)
		} else {
			// Attack roll based action
			sa.Attack = convertToAttackData(attack)
		}

		// Convert effects
		if effects, ok := attack["additional_effects"].(bson.A); ok {
			sa.Effects = convertEffects(effects)
		}

		result = append(result, sa)
	}

	// Second pass: extract multiattack groups from attacks that have "attacks" sub-field
	multiattacks := extractMultiattacks(attacks, result)

	return result, multiattacks
}

// extractMultiattacks scans llm_parsed_attack entries for multiattack containers
// (entries with non-empty "attacks" sub-field) and resolves references to StructuredActions.
func extractMultiattacks(attacks bson.A, actions []StructuredAction) []MultiattackGroup {
	var groups []MultiattackGroup

	for _, a := range attacks {
		attack, ok := a.(bson.M)
		if !ok {
			continue
		}

		subAttacks, ok := attack["attacks"].(bson.A)
		if !ok || len(subAttacks) == 0 {
			continue
		}

		name, _ := attack["name"].(string)
		if name == "" {
			name = "Мультиатака"
		}
		cleanName, _ := parseRecharge(name)

		var entries []MultiattackEntry
		for _, sa := range subAttacks {
			sub, ok := sa.(bson.M)
			if !ok {
				continue
			}

			typeName, _ := sub["type"].(string)
			if typeName == "" {
				continue
			}

			count := 1
			if c, ok := sub["count"].(int32); ok {
				count = int(c)
			} else if c, ok := sub["count"].(int64); ok {
				count = int(c)
			} else if c, ok := sub["count"].(float64); ok {
				count = int(c)
			}

			// Match type name against StructuredAction names
			actionID := matchActionByName(typeName, actions)
			if actionID == "" {
				log.Printf("Warning: multiattack sub-entry %q not matched to any StructuredAction", typeName)
				continue
			}

			entries = append(entries, MultiattackEntry{
				ActionID: actionID,
				Count:    count,
			})
		}

		if len(entries) > 0 {
			groups = append(groups, MultiattackGroup{
				ID:      generateID(cleanName),
				Name:    cleanName,
				Actions: entries,
			})
		}
	}

	return groups
}

// matchActionByName finds a StructuredAction by name, using case-insensitive exact match
// first, then substring fallback.
func matchActionByName(typeName string, actions []StructuredAction) string {
	lower := strings.ToLower(typeName)

	// Exact case-insensitive match
	for _, sa := range actions {
		if strings.EqualFold(sa.Name, typeName) {
			return sa.ID
		}
	}

	// Substring fallback: action name contains the type name or vice versa
	for _, sa := range actions {
		saLower := strings.ToLower(sa.Name)
		if strings.Contains(saLower, lower) || strings.Contains(lower, saLower) {
			return sa.ID
		}
	}

	return ""
}

// parseRecharge extracts recharge info from action name
func parseRecharge(name string) (string, *RechargeData) {
	re := regexp.MustCompile(`\s*\(перезарядка\s*(\d+)(?:[–\-]6)?\)\s*`)
	matches := re.FindStringSubmatch(name)

	if len(matches) >= 2 {
		cleanName := re.ReplaceAllString(name, "")
		cleanName = strings.TrimSpace(cleanName)

		minRoll, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Printf("Warning: failed to parse recharge value %q: %v", matches[1], err)
			minRoll = 6
		}

		return cleanName, &RechargeData{MinRoll: minRoll}
	}

	return name, nil
}

// generateID creates a stable ID from action name using transliteration
func generateID(name string) string {
	// Transliteration map
	translit := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
		'ж': "zh", 'з': "z", 'и': "i", 'й': "j", 'к': "k", 'л': "l", 'м': "m",
		'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
		'ф': "f", 'х': "h", 'ц': "c", 'ч': "ch", 'ш': "sh", 'щ': "sch",
		'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
		'А': "a", 'Б': "b", 'В': "v", 'Г': "g", 'Д': "d", 'Е': "e", 'Ё': "yo",
		'Ж': "zh", 'З': "z", 'И': "i", 'Й': "j", 'К': "k", 'Л': "l", 'М': "m",
		'Н': "n", 'О': "o", 'П': "p", 'Р': "r", 'С': "s", 'Т': "t", 'У': "u",
		'Ф': "f", 'Х': "h", 'Ц': "c", 'Ч': "ch", 'Ш': "sh", 'Щ': "sch",
		'Ъ': "", 'Ы': "y", 'Ь': "", 'Э': "e", 'Ю': "yu", 'Я': "ya",
	}

	var result strings.Builder
	prevDash := false

	for _, r := range strings.ToLower(name) {
		if tr, ok := translit[r]; ok {
			result.WriteString(tr)
			prevDash = false
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
			prevDash = false
		} else if !prevDash && result.Len() > 0 {
			result.WriteRune('-')
			prevDash = true
		}
	}

	id := strings.Trim(result.String(), "-")
	if id == "" {
		id = "action"
	}
	return id
}

// hasOnlySaveDC checks if attack has save_dc but no attack_bonus
func hasOnlySaveDC(attack bson.M) bool {
	_, hasSaveDC := attack["save_dc"]
	bonus, _ := attack["attack_bonus"].(string)
	return hasSaveDC && bonus == ""
}

// convertToAttackData converts llm_parsed_attack to AttackData
func convertToAttackData(attack bson.M) *AttackData {
	data := &AttackData{
		Targets: 1,
	}

	// Type
	attackType, _ := attack["type"].(string)
	data.Type = mapAttackType(attackType)

	// Bonus
	if bonus, ok := attack["attack_bonus"].(string); ok {
		bonus = strings.TrimPrefix(bonus, "+")
		if v, err := strconv.Atoi(strings.TrimSpace(bonus)); err == nil {
			data.Bonus = v
		} else {
			log.Printf("Warning: failed to parse attack_bonus %q: %v", bonus, err)
		}
	}

	// Reach
	if reach, ok := attack["reach"].(string); ok {
		data.Reach = parseDistance(reach)
	}

	// Range
	if rangeStr, ok := attack["range"].(string); ok {
		data.Range = parseRange(rangeStr)
	}

	// Damage
	if dmg, ok := attack["damage"].(bson.M); ok {
		data.Damage = []DamageRoll{convertDamage(dmg)}
	}

	return data
}

// convertToSaveData converts llm_parsed_attack to SaveData
func convertToSaveData(attack bson.M) *SaveData {
	data := &SaveData{
		OnFail:    "effect applies",
		OnSuccess: "no effect",
	}

	// DC
	switch v := attack["save_dc"].(type) {
	case int32:
		data.DC = int(v)
	case int64:
		data.DC = int(v)
	case int:
		data.DC = v
	case float64:
		data.DC = int(v)
	}

	// Ability
	if saveType, ok := attack["save_type"].(string); ok {
		data.Ability = mapSaveType(saveType)
	}

	return data
}

// convertDamage converts damage object to DamageRoll
func convertDamage(dmg bson.M) DamageRoll {
	roll := DamageRoll{}

	// Count
	switch v := dmg["count"].(type) {
	case int32:
		roll.DiceCount = int(v)
	case int64:
		roll.DiceCount = int(v)
	case int:
		roll.DiceCount = v
	case float64:
		roll.DiceCount = int(v)
	}

	// Dice type
	if dice, ok := dmg["dice"].(string); ok {
		roll.DiceType = dice
	}

	// Bonus
	switch v := dmg["bonus"].(type) {
	case int32:
		roll.Bonus = int(v)
	case int64:
		roll.Bonus = int(v)
	case int:
		roll.Bonus = v
	case float64:
		roll.Bonus = int(v)
	}

	// Damage type
	if dt, ok := dmg["type"].(string); ok {
		roll.DamageType = mapDamageType(dt)
	}

	return roll
}

// convertEffects converts additional_effects to Effect slice
func convertEffects(effects bson.A) []Effect {
	var result []Effect

	for _, e := range effects {
		eff, ok := e.(bson.M)
		if !ok {
			continue
		}

		condition, _ := eff["condition"].(string)
		duration, _ := eff["duration"].(string)

		mappedCondition := mapCondition(condition)

		if mappedCondition != "" {
			// Standard condition
			ce := &ConditionEffect{
				Condition: mappedCondition,
				Duration:  duration,
			}

			// Escape DC
			switch v := eff["escape_dc"].(type) {
			case int32:
				ce.EscapeDC = int(v)
			case int64:
				ce.EscapeDC = int(v)
			case int:
				ce.EscapeDC = v
			case float64:
				ce.EscapeDC = int(v)
			}

			result = append(result, Effect{Condition: ce})
		} else if condition != "" {
			// Non-standard effect - store as description
			result = append(result, Effect{Description: condition})
		}
	}

	return result
}

// mapAttackType maps Russian/mixed attack types to English
func mapAttackType(t string) string {
	switch strings.ToLower(t) {
	case "melee":
		return "melee_weapon"
	case "ranged":
		return "ranged_weapon"
	case "melee/ranged", "ranged/melee", "ranged_or_melee", "both", "either", "versatile":
		return "melee_or_ranged_weapon"
	case "touch":
		return "melee_spell"
	default:
		return "melee_weapon"
	}
}

// mapSaveType maps Russian save types to English abbreviations
func mapSaveType(t string) string {
	tLower := strings.ToLower(t)
	switch {
	case strings.Contains(tLower, "сил"):
		return "STR"
	case strings.Contains(tLower, "ловк"):
		return "DEX"
	case strings.Contains(tLower, "тел"):
		return "CON"
	case strings.Contains(tLower, "инт"):
		return "INT"
	case strings.Contains(tLower, "мудр"):
		return "WIS"
	case strings.Contains(tLower, "хар"):
		return "CHA"
	default:
		return "CON"
	}
}

// mapDamageType maps Russian damage types to English
func mapDamageType(t string) string {
	tLower := strings.ToLower(t)

	// Check for combined types first
	if strings.Contains(tLower, ",") || strings.Contains(tLower, " или ") {
		return "varies"
	}

	switch {
	case strings.Contains(tLower, "дроб"):
		return "bludgeoning"
	case strings.Contains(tLower, "кол") || strings.Contains(tLower, "прокал") || strings.Contains(tLower, "проник"):
		return "piercing"
	case strings.Contains(tLower, "руб"):
		return "slashing"
	case strings.Contains(tLower, "огн") || strings.Contains(tLower, "огон"):
		return "fire"
	case strings.Contains(tLower, "холод"):
		return "cold"
	case strings.Contains(tLower, "электр") || strings.Contains(tLower, "молн"):
		return "lightning"
	case strings.Contains(tLower, "кисл"):
		return "acid"
	case strings.Contains(tLower, "яд"):
		return "poison"
	case strings.Contains(tLower, "некрот"):
		return "necrotic"
	case strings.Contains(tLower, "излуч") || strings.Contains(tLower, "свет"):
		return "radiant"
	case strings.Contains(tLower, "псих"):
		return "psychic"
	case strings.Contains(tLower, "звук") || strings.Contains(tLower, "гром"):
		return "thunder"
	case strings.Contains(tLower, "сил"):
		return "force"
	default:
		return "varies"
	}
}

// mapCondition maps Russian conditions to English
func mapCondition(c string) string {
	cLower := strings.ToLower(c)

	switch {
	case strings.Contains(cLower, "слеп") || strings.Contains(cLower, "ослепл"):
		return "blinded"
	case strings.Contains(cLower, "очаров"):
		return "charmed"
	case strings.Contains(cLower, "оглох") || strings.Contains(cLower, "глух"):
		return "deafened"
	case strings.Contains(cLower, "испуг") || strings.Contains(cLower, "напуг") || strings.Contains(cLower, "страх"):
		return "frightened"
	case strings.Contains(cLower, "схвач") || strings.Contains(cLower, "захвач"):
		return "grappled"
	case strings.Contains(cLower, "недееспособ"):
		return "incapacitated"
	case strings.Contains(cLower, "невидим"):
		return "invisible"
	case strings.Contains(cLower, "парализ"):
		return "paralyzed"
	case strings.Contains(cLower, "окамен"):
		return "petrified"
	case strings.Contains(cLower, "отравл"):
		return "poisoned"
	case strings.Contains(cLower, "ничком") || strings.Contains(cLower, "сбит с ног") || strings.Contains(cLower, "сбита с ног"):
		return "prone"
	case strings.Contains(cLower, "опутан") || strings.Contains(cLower, "удержив"):
		return "restrained"
	case strings.Contains(cLower, "ошеломл") || strings.Contains(cLower, "оглуш"):
		return "stunned"
	case strings.Contains(cLower, "без сознан") || strings.Contains(cLower, "бессозн") || strings.Contains(cLower, "теряет сознан"):
		return "unconscious"
	case strings.Contains(cLower, "истощ"):
		return "exhaustion"
	default:
		return "" // Non-standard condition
	}
}

// parseDistance extracts numeric distance from string like "5 фт." or "10 футов"
func parseDistance(s string) int {
	re := regexp.MustCompile(`(\d+)\s*(?:фт|фут)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) >= 2 {
		d, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Printf("Warning: failed to parse distance %q: %v", matches[1], err)
			return 0
		}
		return d
	}
	return 0
}

// parseRange extracts normal/long range from string like "30/120 фт."
func parseRange(s string) *RangeData {
	re := regexp.MustCompile(`(\d+)/(\d+)`)
	matches := re.FindStringSubmatch(s)
	if len(matches) >= 3 {
		normal, err1 := strconv.Atoi(matches[1])
		long, err2 := strconv.Atoi(matches[2])
		if err1 != nil || err2 != nil {
			log.Printf("Warning: failed to parse range %q", s)
			return nil
		}
		return &RangeData{Normal: normal, Long: long}
	}

	// Try single distance
	re2 := regexp.MustCompile(`(\d+)\s*(?:фт|фут)`)
	matches2 := re2.FindStringSubmatch(s)
	if len(matches2) >= 2 {
		normal, err := strconv.Atoi(matches2[1])
		if err != nil {
			log.Printf("Warning: failed to parse range distance %q: %v", matches2[1], err)
			return nil
		}
		return &RangeData{Normal: normal}
	}

	return nil
}

// printResult prints migration result for a creature
func printResult(creature bson.M, result *MigrationResult) {
	name := creature["name"].(bson.M)
	fmt.Printf("\n=== %s ===\n", name["rus"])

	if result.Movement != nil {
		j, _ := json.MarshalIndent(result.Movement, "", "  ")
		fmt.Printf("Movement: %s\n", j)
	} else if result.MovementSkipped {
		fmt.Printf("Movement: SKIPPED (%s)\n", result.MovementReason)
	}

	if result.Vision != nil {
		j, _ := json.MarshalIndent(result.Vision, "", "  ")
		fmt.Printf("Vision: %s\n", j)
	}

	if len(result.StructuredActions) > 0 {
		fmt.Printf("StructuredActions: %d actions\n", len(result.StructuredActions))
		for i, sa := range result.StructuredActions {
			fmt.Printf("  [%d] %s (id=%s)\n", i, sa.Name, sa.ID)
			if sa.Attack != nil {
				fmt.Printf("      Attack: %s +%d\n", sa.Attack.Type, sa.Attack.Bonus)
			}
			if sa.SavingThrow != nil {
				fmt.Printf("      Save: DC %d %s\n", sa.SavingThrow.DC, sa.SavingThrow.Ability)
			}
			if sa.Recharge != nil {
				fmt.Printf("      Recharge: %d-6\n", sa.Recharge.MinRoll)
			}
		}
	} else if result.ActionsSkipped {
		fmt.Printf("StructuredActions: SKIPPED (%s)\n", result.ActionsReason)
	}

	if len(result.Multiattacks) > 0 {
		fmt.Printf("Multiattacks: %d group(s)\n", len(result.Multiattacks))
		for i, mg := range result.Multiattacks {
			fmt.Printf("  [%d] %s (id=%s)\n", i, mg.Name, mg.ID)
			for _, entry := range mg.Actions {
				fmt.Printf("      %s ×%d\n", entry.ActionID, entry.Count)
			}
		}
	}
}

// printStats prints final migration statistics
func printStats(stats *MigrationStats) {
	fmt.Println("\n========== MIGRATION STATS ==========")
	fmt.Printf("Total processed:     %d\n", stats.Total)
	fmt.Printf("Modified:            %d\n", stats.Modified)
	fmt.Printf("Written to DB:       %d\n", stats.Written)
	fmt.Printf("Errors:              %d\n", stats.Errors)
	fmt.Printf("Movement skipped:    %d\n", stats.MovementSkipped)
	fmt.Printf("Actions skipped:     %d\n", stats.ActionsSkipped)

	if len(stats.SkippedCreatures) > 0 {
		fmt.Println("\n--- Skipped creatures ---")
		for _, s := range stats.SkippedCreatures {
			fmt.Printf("  %s: %s\n", s.Name, s.Reason)
		}
	}
}
