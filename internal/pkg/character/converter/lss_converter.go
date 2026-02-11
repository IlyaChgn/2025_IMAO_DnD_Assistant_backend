package converter

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConvertLSS converts an LSS JSON byte slice into a CharacterBase and a ConversionReport.
// This is the main entry point for the LSS import pipeline.
func ConvertLSS(rawJSON []byte, userID string) (*models.CharacterBase, *models.ConversionReport, error) {
	// Step 1: Parse outer envelope
	var envelope map[string]interface{}
	if err := json.Unmarshal(rawJSON, &envelope); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Step 2: Double-parse data field (JSON string → map)
	dataStr, ok := envelope["data"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("missing or invalid 'data' field in LSS envelope")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON in 'data' field: %w", err)
	}

	report := &models.ConversionReport{Success: true}
	char := &models.CharacterBase{
		ID:      primitive.NewObjectID(),
		UserID:  userID,
		Version: 1,
	}

	// Initialize slices to empty (avoid null in JSON output)
	char.Classes = []models.ClassEntry{}
	char.Proficiencies.Skills = []string{}
	char.Proficiencies.SavingThrows = []models.AbilityType{}
	char.Weapons = []models.WeaponDef{}
	char.Expertise = []string{}

	// Step 3: Extract identity fields
	extractIdentity(data, envelope, char, report)

	// Step 4: Extract ability scores
	extractAbilityScores(data, char, report)

	// Step 5: Extract proficiencies (saves, skills)
	extractProficiencies(data, char, report)

	// Step 6: Extract vitality (HP, AC, speed, hit die)
	extractVitality(data, char, report)

	// Step 7: Extract weapons
	extractWeapons(data, char, report)

	// Step 8: Extract spellcasting info
	extractSpellcasting(data, char, report)

	// Step 9: Extract text fields (Tiptap → plain text)
	extractTextFields(data, char, report)

	// Step 10: Extract coins
	extractCoins(data, char, report)

	// Extract appearance
	extractAppearance(data, char, report)

	// Extract avatar
	extractAvatar(data, char, report)

	// Extract edition from envelope
	if edition, ok := getStringField(envelope, "edition"); ok {
		char.Edition = edition
		report.FieldsCopied++
	}

	// Set timestamps and import source
	now := time.Now().UTC().Format(time.RFC3339)
	char.CreatedAt = now
	char.UpdatedAt = now

	warnings := make([]string, 0, len(report.Warnings))
	for _, w := range report.Warnings {
		warnings = append(warnings, w.Message)
	}
	char.ImportSource = &models.ImportSource{
		Format:     "lss_v2",
		ImportedAt: now,
		Warnings:   warnings,
	}

	report.CharacterName = char.Name

	return char, report, nil
}

// extractIdentity extracts name, class, level, race, background, alignment, experience.
func extractIdentity(data, envelope map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	// Name
	if name := getNestedValue(data, "name", "value"); name != "" {
		char.Name = name
		report.FieldsCopied++
	}

	// Info block
	info, _ := data["info"].(map[string]interface{})
	if info == nil {
		report.Warnings = append(report.Warnings, models.ConversionWarning{
			Field: "info", Message: "Missing info block", Level: "warning",
		})
		return
	}

	// Class
	if classRaw := getNestedValue2(info, "charClass", "value"); classRaw != "" {
		className, found := MapClassName(classRaw)
		char.Classes = []models.ClassEntry{{ClassName: className}}
		if found {
			report.FieldsParsed++
		} else {
			report.FieldsParsed++
			report.Warnings = append(report.Warnings, models.ConversionWarning{
				Field:   "info.charClass",
				Message: fmt.Sprintf("Unknown class '%s' — stored as custom", classRaw),
				Level:   "warning",
			})
		}
	}

	// Level
	if level := getNestedNumber(info, "level", "value"); level > 0 {
		if len(char.Classes) > 0 {
			char.Classes[0].Level = level
		}
		report.FieldsCopied++
	}

	// Race
	if race := getNestedValue2(info, "race", "value"); race != "" {
		char.Race = race
		report.FieldsCopied++
	}

	// Background
	if bg := getNestedValue2(info, "background", "value"); bg != "" {
		char.Background = bg
		report.FieldsCopied++
	}

	// Alignment
	if al := getNestedValue2(info, "alignment", "value"); al != "" {
		char.Alignment = al
		report.FieldsCopied++
	}

	// Experience
	if exp := getNestedNumber(info, "experience", "value"); exp > 0 {
		char.Experience = exp
		report.FieldsCopied++
	}
}

// extractAbilityScores extracts the six ability scores.
// Modifiers from LSS are ignored — they're recalculated from scores.
func extractAbilityScores(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	stats, _ := data["stats"].(map[string]interface{})
	if stats == nil {
		return
	}

	abilities := map[string]*int{
		"str": &char.AbilityScores.Str,
		"dex": &char.AbilityScores.Dex,
		"con": &char.AbilityScores.Con,
		"int": &char.AbilityScores.Int,
		"wis": &char.AbilityScores.Wis,
		"cha": &char.AbilityScores.Cha,
	}

	for code, target := range abilities {
		stat, ok := stats[code].(map[string]interface{})
		if !ok {
			continue
		}
		score := toInt(stat["score"])
		if score > 0 {
			*target = score
			report.FieldsCopied++
		}

		// Verify modifier matches score (warn if LSS has wrong modifier)
		if mod, ok := stat["modifier"]; ok {
			lssMod := toInt(mod)
			expectedMod := int(math.Floor(float64(score-10) / 2))
			if lssMod != expectedMod && score > 0 {
				report.Warnings = append(report.Warnings, models.ConversionWarning{
					Field:   fmt.Sprintf("stats.%s.modifier", code),
					Message: fmt.Sprintf("LSS modifier %d doesn't match calculated %d (from score %d) — using calculated", lssMod, expectedMod, score),
					Level:   "info",
				})
			}
		}
	}
}

// extractProficiencies extracts saving throw and skill proficiencies.
func extractProficiencies(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	// Saving throws
	saves, _ := data["saves"].(map[string]interface{})
	if saves != nil {
		var savingThrows []models.AbilityType
		for code, entry := range saves {
			save, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			if toBool(save["isProf"]) {
				if at, ok := MapAbilityCode(code); ok {
					savingThrows = append(savingThrows, at)
				}
			}
		}
		char.Proficiencies.SavingThrows = savingThrows
		report.FieldsCopied++
	}

	// Skills
	skills, _ := data["skills"].(map[string]interface{})
	if skills != nil {
		var proficientSkills []string
		var expertiseSkills []string
		for skillName, entry := range skills {
			skill, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			isProf := toInt(skill["isProf"])
			// Normalize skill name: "sleight of hand" → "sleight_of_hand", "animal handling" → "animal_handling"
			normalized := strings.ReplaceAll(skillName, " ", "_")
			if isProf == 1 {
				proficientSkills = append(proficientSkills, normalized)
			} else if isProf == 2 {
				expertiseSkills = append(expertiseSkills, normalized)
				proficientSkills = append(proficientSkills, normalized) // expertise implies proficiency
			}
		}
		char.Proficiencies.Skills = proficientSkills
		char.Expertise = expertiseSkills
		report.FieldsCopied++
	}
}

// extractVitality extracts HP, AC, speed, and hit die.
func extractVitality(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	vitality, _ := data["vitality"].(map[string]interface{})
	if vitality == nil {
		return
	}

	// Max HP
	if hpMax := getNestedNumber(vitality, "hp-max", "value"); hpMax > 0 {
		hp := hpMax
		char.HitPoints.MaxOverride = &hp
		report.FieldsCopied++
	}

	// AC
	if acVal := getNestedNumber(vitality, "ac", "value"); acVal > 0 {
		ac := acVal
		char.ArmorClassOverride = &ac
		report.FieldsCopied++
	}

	// Speed
	if speed := getNestedNumber(vitality, "speed", "value"); speed > 0 {
		char.BaseSpeed = speed
		report.FieldsCopied++
	}

	// Hit Die
	if hdRaw := getNestedValue(vitality, "hit-die", "value"); hdRaw != "" {
		char.HitPoints.HitDie = ParseHitDie(hdRaw)
		report.FieldsParsed++
	}
}

// extractWeapons extracts weapons from the weaponsList array.
func extractWeapons(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	weaponsList, ok := data["weaponsList"].([]interface{})
	if !ok {
		return
	}

	var weapons []models.WeaponDef
	for i, entry := range weaponsList {
		weapon, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		name := getNestedValue(weapon, "name", "value")
		if name == "" {
			continue
		}

		dmgRaw := getNestedValue(weapon, "dmg", "value")
		dice, damageType := ParseDamageString(dmgRaw)

		// Skip entries with no damage (like "Колчан стрел")
		if dice == "" && dmgRaw == "" {
			report.Warnings = append(report.Warnings, models.ConversionWarning{
				Field:   fmt.Sprintf("weaponsList[%d]", i),
				Message: fmt.Sprintf("Weapon '%s' has no damage — skipped", name),
				Level:   "info",
			})
			continue
		}

		w := models.WeaponDef{
			ID:         fmt.Sprintf("weapon_%d", i),
			Name:       name,
			AttackType: "melee", // default, no reliable way to determine from LSS
			DamageDice: dice,
			DamageType: damageType,
		}

		// Guess attack type from weapon name
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, "лук") || strings.Contains(lowerName, "арбалет") ||
			strings.Contains(lowerName, "праща") {
			w.AttackType = "ranged"
		}

		weapons = append(weapons, w)
		report.FieldsParsed++
	}

	char.Weapons = weapons
}

// extractSpellcasting extracts spellcasting ability and spell text.
func extractSpellcasting(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	spellsInfo, _ := data["spellsInfo"].(map[string]interface{})
	if spellsInfo == nil {
		return
	}

	// Determine spellcasting ability
	var ability models.AbilityType
	base, _ := spellsInfo["base"].(map[string]interface{})
	if base != nil {
		// Try "code" field first (some LSS exports use it)
		if code, ok := base["code"].(string); ok {
			if at, found := MapAbilityCode(code); found {
				ability = at
			}
		}
		// Fall back to "value" field (Russian ability name like "Мудрость")
		if ability == "" {
			if val, ok := base["value"].(string); ok {
				if at, found := MapAbilityNameRu(val); found {
					ability = at
				}
			}
		}
	}

	if ability == "" {
		report.Warnings = append(report.Warnings, models.ConversionWarning{
			Field:   "spellsInfo.base",
			Message: "Could not determine spellcasting ability",
			Level:   "warning",
		})
		return
	}

	sc := &models.CharacterSpellcasting{
		Ability: ability,
	}

	// Extract spell text per level from text fields
	textBlock, _ := data["text"].(map[string]interface{})
	if textBlock != nil {
		spellTexts := make(map[int]string)
		for level := 0; level <= 9; level++ {
			key := fmt.Sprintf("spells-level-%d", level)
			if entry, ok := textBlock[key]; ok {
				text := extractTiptapFromTextField(entry)
				if text != "" && !isEmptySpellText(text) {
					spellTexts[level] = text
					report.FieldsParsed++
				}
			}
		}
		if len(spellTexts) > 0 {
			sc.SpellTexts = spellTexts
		}
	}

	char.Spellcasting = sc
	report.FieldsParsed++
}

// extractTextFields converts Tiptap rich text to plain text for narrative fields.
func extractTextFields(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	textBlock, _ := data["text"].(map[string]interface{})
	if textBlock == nil {
		return
	}

	fieldMap := map[string]*string{
		"personality": &char.PersonalityTraits,
		"ideals":      &char.Ideals,
		"bonds":       &char.Bonds,
		"flaws":       &char.Flaws,
		"background":  &char.Backstory,
	}

	for key, target := range fieldMap {
		if entry, ok := textBlock[key]; ok {
			text := extractTiptapFromTextField(entry)
			if text != "" {
				*target = text
				report.FieldsCopied++
			}
		}
	}

	// Collect traits, features, equipment, attacks into notes
	var notesParts []string
	for _, key := range []string{"traits", "features", "allies", "equipment", "attacks"} {
		if entry, ok := textBlock[key]; ok {
			text := extractTiptapFromTextField(entry)
			if text != "" {
				notesParts = append(notesParts, fmt.Sprintf("=== %s ===\n%s", key, text))
				report.FieldsParsed++
			}
		}
	}
	if len(notesParts) > 0 {
		char.Notes = strings.Join(notesParts, "\n\n")
	}
}

// extractCoins extracts currency values.
func extractCoins(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	coins, _ := data["coins"].(map[string]interface{})
	if coins == nil {
		return
	}

	coinMap := map[string]*int{
		"gp": &char.Coins.Gp,
		"sp": &char.Coins.Sp,
		"cp": &char.Coins.Cp,
		"pp": &char.Coins.Pp,
		"ep": &char.Coins.Ep,
	}

	for key, target := range coinMap {
		if entry, ok := coins[key].(map[string]interface{}); ok {
			*target = toInt(entry["value"])
		}
	}
	report.FieldsCopied++
}

// extractAppearance extracts physical description from subInfo.
func extractAppearance(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	subInfo, _ := data["subInfo"].(map[string]interface{})
	if subInfo == nil {
		return
	}

	fieldMap := map[string]*string{
		"age":    &char.Appearance.Age,
		"height": &char.Appearance.Height,
		"weight": &char.Appearance.Weight,
		"eyes":   &char.Appearance.Eyes,
		"skin":   &char.Appearance.Skin,
		"hair":   &char.Appearance.Hair,
	}

	for key, target := range fieldMap {
		if val := getNestedValue2(subInfo, key, "value"); val != "" {
			*target = val
			report.FieldsCopied++
		}
	}
}

// extractAvatar extracts avatar image URLs.
func extractAvatar(data map[string]interface{}, char *models.CharacterBase, report *models.ConversionReport) {
	avatar, _ := data["avatar"].(map[string]interface{})
	if avatar == nil {
		return
	}

	jpeg, _ := avatar["jpeg"].(string)
	webp, _ := avatar["webp"].(string)
	if jpeg != "" || webp != "" {
		char.Avatar = &models.CharacterAvatar{Jpeg: jpeg, Webp: webp}
		report.FieldsCopied++
	}
}

// --- Helper functions ---

// getNestedValue extracts a string from data[key1][key2].
func getNestedValue(data map[string]interface{}, key1, key2 string) string {
	entry, ok := data[key1].(map[string]interface{})
	if !ok {
		return ""
	}
	val, _ := entry[key2].(string)
	return val
}

// getNestedValue2 extracts a string from a map[key1][key2] (same as getNestedValue but for a pre-extracted sub-map).
func getNestedValue2(parent map[string]interface{}, key1, key2 string) string {
	entry, ok := parent[key1].(map[string]interface{})
	if !ok {
		return ""
	}
	return toString(entry[key2])
}

// getNestedNumber extracts an int from parent[key1][key2].
func getNestedNumber(parent map[string]interface{}, key1, key2 string) int {
	entry, ok := parent[key1].(map[string]interface{})
	if !ok {
		return 0
	}
	return toInt(entry[key2])
}

// getStringField gets a string field from a map.
func getStringField(m map[string]interface{}, key string) (string, bool) {
	val, ok := m[key].(string)
	return val, ok
}

// toInt converts interface{} to int, handling float64 (JSON default) and string.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		n2, _ := strconv.Atoi(strings.TrimSpace(n))
		return n2
	default:
		return 0
	}
}

// toString converts interface{} to string.
func toString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case float64:
		if s == float64(int(s)) {
			return strconv.Itoa(int(s))
		}
		return strconv.FormatFloat(s, 'f', -1, 64)
	case int:
		return strconv.Itoa(s)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", s)
	}
}

// toBool converts interface{} to bool.
func toBool(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case string:
		return b == "true" || b == "1"
	default:
		return false
	}
}

// extractTiptapFromTextField extracts plain text from an LSS text field.
// LSS text fields have structure: {"value": {"data": <tiptap_or_string>}} or {"value": <string>}
func extractTiptapFromTextField(entry interface{}) string {
	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		return ""
	}

	value, ok := entryMap["value"]
	if !ok {
		return ""
	}

	// value can be a string directly or a map with "data" field
	switch v := value.(type) {
	case string:
		return TiptapToPlainText(v)
	case map[string]interface{}:
		if data, ok := v["data"]; ok {
			return TiptapToPlainText(data)
		}
	}

	return ""
}

// isEmptySpellText checks if spell text is effectively empty.
func isEmptySpellText(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return lower == "" ||
		lower == "не владеет заговорами." ||
		lower == "не владеет заклинаниями." ||
		lower == "нет"
}
