package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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

// SpellDefInfo holds minimal spell info loaded from spell_definitions.
type SpellDefInfo struct {
	EngName string
	Name    string // English display name
	Level   int
	School  string
}

// InnateSpellcasting mirrors models.InnateSpellcasting for the migration.
type InnateSpellcasting struct {
	Ability          string               `json:"ability,omitempty" bson:"ability,omitempty"`
	SpellSaveDC      int                  `json:"spellSaveDC" bson:"spellSaveDC"`
	SpellAttackBonus int                  `json:"spellAttackBonus,omitempty" bson:"spellAttackBonus,omitempty"`
	AtWill           []SpellKnown         `json:"atWill,omitempty" bson:"atWill,omitempty"`
	PerDay           map[int][]SpellKnown `json:"perDay,omitempty" bson:"perDay,omitempty"`
	Note             string               `json:"note,omitempty" bson:"note,omitempty"`
}

// SpellKnown mirrors models.SpellKnown for the migration.
type SpellKnown struct {
	SpellID string `json:"spellID,omitempty" bson:"spellID,omitempty"`
	Name    string `json:"name" bson:"name"`
	Level   int    `json:"level" bson:"level"`
	School  string `json:"school,omitempty" bson:"school,omitempty"`
}

// MigrationStats tracks migration progress.
type MigrationStats struct {
	Total       int
	Modified    int
	Written     int
	Skipped     int
	Errors      int
	ParseErrors []ParseErrorInfo
}

// ParseErrorInfo records a per-creature parse error.
type ParseErrorInfo struct {
	Name   string
	Reason string
}

// mapAbility maps Russian ability name (any grammatical case) to AbilityType.
func mapAbility(rus string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(rus))
	switch {
	case strings.HasPrefix(s, "харизм"):
		return "CHA", true
	case strings.HasPrefix(s, "интеллект"):
		return "INT", true
	case strings.HasPrefix(s, "мудрост"):
		return "WIS", true
	case strings.HasPrefix(s, "сил"):
		return "STR", true
	case strings.HasPrefix(s, "ловкост"):
		return "DEX", true
	case strings.HasPrefix(s, "телосложени"):
		return "CON", true
	default:
		return "", false
	}
}

// Compiled regexes.
var (
	abilityRe      = regexp.MustCompile(`<strong>(.+?)</strong>`)
	dcRe           = regexp.MustCompile(`(?i)(?:Сл спасброска от заклинани[яй]|спасброс\S* от заклинани\S* Сл)\s+(\d+)`)
	attackBonusRe  = regexp.MustCompile(`(?i)(?:модификатор атаки заклинанием|попаданию\S* атак\S* заклинани\S*)\s*\+?(\d+)`)
	attackBonusFB  = regexp.MustCompile(`\+(\d+)\s*(?:</[^>]+>\s*)?(?:&nbsp;|\s)*к\s*(?:&nbsp;|\s)*попаданию`)
	noteRe         = regexp.MustCompile(`(?i)не нуждаясь в ([^.:<]+)`)
	spellListDivRe = regexp.MustCompile(`(?si)<div class="spell-list">(.*?)</div>`)
	spellSectionRe = regexp.MustCompile(`(?si)<p>(.*?)</p>`)
	usageAtWillRe  = regexp.MustCompile(`(?i)неограниченн?о|по желанию`)
	usagePerDayRe  = regexp.MustCompile(`(\d+)/день`)
	spellLinkRe    = regexp.MustCompile(`href="/spells/(.+?)"[^>]*>.*?\[(.+?)\]`)
	spellLinkAnyRe = regexp.MustCompile(`<a\s+href="/spells/([^"]+)"[^>]*>(.*?)</a>`)
	htmlTagRe      = regexp.MustCompile(`<[^>]*>`)
	parenthRe      = regexp.MustCompile(`\s*\([^)]*\)\s*`)
	bareEmRe       = regexp.MustCompile(`<em>([^<]+)</em>`)
	innateFeatRe   = regexp.MustCompile(`(?i)врождённое колдовство|врожденное колдовство`)
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

	db := client.Database("bestiary_db")

	// Load spell definitions into lookup map
	spellMap := loadSpellDefinitions(ctx, db)
	fmt.Printf("Loaded %d spell definitions\n", len(spellMap))

	coll := db.Collection("creatures")

	// Build query: find creatures with innate spellcasting feat
	filter := bson.M{
		"feats.name": bson.M{
			"$regex":   "врождённое колдовство|врожденное колдовство",
			"$options": "i",
		},
	}
	if *creatureID != "" {
		oid, err := primitive.ObjectIDFromHex(*creatureID)
		if err != nil {
			log.Fatal("Invalid creature ID:", err)
		}
		filter["_id"] = oid
	}

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

		stats.Total++

		creatureName := getCreatureName(creature)

		// Skip if innateSpellcasting already set
		if existing, ok := creature["innateSpellcasting"]; ok && existing != nil {
			if *verbose {
				fmt.Printf("  SKIP %s: innateSpellcasting already set\n", creatureName)
			}
			stats.Skipped++
			continue
		}

		// Find the innate spellcasting feat HTML
		featHTML := findInnateFeatHTML(creature)
		if featHTML == "" {
			if *verbose {
				fmt.Printf("  SKIP %s: no feat HTML found\n", creatureName)
			}
			stats.Skipped++
			continue
		}

		// Parse HTML into InnateSpellcasting
		innate, err := parseInnateSpellcasting(featHTML, spellMap)
		if err != nil {
			if *verbose {
				fmt.Printf("  ERROR %s: %v\n", creatureName, err)
			}
			stats.ParseErrors = append(stats.ParseErrors, ParseErrorInfo{
				Name:   creatureName,
				Reason: err.Error(),
			})
			stats.Errors++
			continue
		}

		stats.Modified++

		if *verbose || *creatureID != "" {
			printInnate(creatureName, innate)
		}

		// Write to database
		if !*dryRun {
			update := bson.M{"$set": bson.M{"innateSpellcasting": innate}}
			_, err := coll.UpdateByID(ctx, creature["_id"], update)
			if err != nil {
				log.Printf("Update error for %s: %v", creatureName, err)
				stats.Errors++
			} else {
				stats.Written++
			}
		}
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Cursor iteration error: %v", err)
		stats.Errors++
	}

	printStats(stats)

	if *dryRun {
		fmt.Println("\n⚠️  DRY RUN - no changes written. Use -dry-run=false to apply.")
	}
}

// loadSpellDefinitions loads all spell definitions from the database into a lookup map
// keyed by engName.
func loadSpellDefinitions(ctx context.Context, db *mongo.Database) map[string]SpellDefInfo {
	coll := db.Collection("spell_definitions")
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Warning: could not load spell definitions: %v", err)
		return make(map[string]SpellDefInfo)
	}
	defer cursor.Close(ctx)

	result := make(map[string]SpellDefInfo)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		engName, _ := doc["engName"].(string)
		if engName == "" {
			continue
		}

		level := toInt(doc["level"])
		school, _ := doc["school"].(string)

		displayName := engName
		if nameObj, ok := doc["name"].(bson.M); ok {
			if eng, ok := nameObj["eng"].(string); ok && eng != "" {
				displayName = eng
			}
		}

		result[engName] = SpellDefInfo{
			EngName: engName,
			Name:    displayName,
			Level:   level,
			School:  school,
		}
	}

	return result
}

// toInt extracts an int from a BSON value that may be int32, int64, or float64.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func getCreatureName(creature bson.M) string {
	if name, ok := creature["name"].(bson.M); ok {
		if rus, ok := name["rus"].(string); ok {
			return rus
		}
		if eng, ok := name["eng"].(string); ok {
			return eng
		}
	}
	return "unknown"
}

// findInnateFeatHTML finds the innate spellcasting feat HTML from creature feats.
func findInnateFeatHTML(creature bson.M) string {
	feats, ok := creature["feats"].(bson.A)
	if !ok {
		return ""
	}

	for _, f := range feats {
		feat, ok := f.(bson.M)
		if !ok {
			continue
		}

		name, _ := feat["name"].(string)
		if innateFeatRe.MatchString(name) {
			value, _ := feat["value"].(string)
			return value
		}
	}

	return ""
}

// parseInnateSpellcasting parses HTML feat text into InnateSpellcasting struct.
func parseInnateSpellcasting(featHTML string, spellMap map[string]SpellDefInfo) (*InnateSpellcasting, error) {
	innate := &InnateSpellcasting{}

	// 1. Extract ability (optional — some stat blocks omit it)
	if matches := abilityRe.FindStringSubmatch(featHTML); len(matches) >= 2 {
		if ability, ok := mapAbility(matches[1]); ok {
			innate.Ability = ability
		}
		// If ability word is in <strong> but doesn't map, leave empty
	}
	// If no ability found at all, leave empty — spell list is still useful

	// 2. Extract DC (optional — some innate casters only have utility spells)
	if matches := dcRe.FindStringSubmatch(featHTML); len(matches) >= 2 {
		dc, _ := strconv.Atoi(matches[1])
		innate.SpellSaveDC = dc
	}

	// 3. Extract attack bonus (optional)
	if matches := attackBonusRe.FindStringSubmatch(featHTML); len(matches) >= 2 {
		bonus, _ := strconv.Atoi(matches[1])
		innate.SpellAttackBonus = bonus
	} else if matches := attackBonusFB.FindStringSubmatch(featHTML); len(matches) >= 2 {
		bonus, _ := strconv.Atoi(matches[1])
		innate.SpellAttackBonus = bonus
	}

	// 4. Extract note (optional)
	if matches := noteRe.FindStringSubmatch(featHTML); len(matches) >= 2 {
		innate.Note = strings.TrimSpace(html.UnescapeString(matches[1]))
	}

	// 5. Parse spell sections from <div class="spell-list">
	spellListMatch := spellListDivRe.FindStringSubmatch(featHTML)
	if len(spellListMatch) < 2 {
		return nil, fmt.Errorf("no spell-list div found")
	}

	spellListHTML := spellListMatch[1]
	sections := spellSectionRe.FindAllStringSubmatch(spellListHTML, -1)

	if len(sections) == 0 {
		return nil, fmt.Errorf("no spell sections found inside spell-list div")
	}

	perDay := make(map[int][]SpellKnown)

	for _, section := range sections {
		if len(section) < 2 {
			continue
		}
		sectionHTML := section[1]

		// Determine usage type
		var usageType string
		var usesPerDay int

		if usageAtWillRe.MatchString(sectionHTML) {
			usageType = "at_will"
		} else if matches := usagePerDayRe.FindStringSubmatch(sectionHTML); len(matches) >= 2 {
			usageType = "per_day"
			usesPerDay, _ = strconv.Atoi(matches[1])
		} else {
			continue
		}

		// Extract spells from this section
		spells := extractSpells(sectionHTML, spellMap)

		switch usageType {
		case "at_will":
			innate.AtWill = append(innate.AtWill, spells...)
		case "per_day":
			if usesPerDay > 0 && len(spells) > 0 {
				perDay[usesPerDay] = append(perDay[usesPerDay], spells...)
			}
		}
	}

	if len(perDay) > 0 {
		innate.PerDay = perDay
	}

	// Validate we got at least some spells
	totalSpells := len(innate.AtWill)
	for _, spells := range innate.PerDay {
		totalSpells += len(spells)
	}
	if totalSpells == 0 {
		return nil, fmt.Errorf("no spells extracted")
	}

	return innate, nil
}

// extractSpells extracts spell entries from a section HTML.
func extractSpells(sectionHTML string, spellMap map[string]SpellDefInfo) []SpellKnown {
	var spells []SpellKnown

	// Strategy 1: href="/spells/Xxx" ... [english name]
	linkWithBrackets := spellLinkRe.FindAllStringSubmatch(sectionHTML, -1)
	if len(linkWithBrackets) > 0 {
		for _, m := range linkWithBrackets {
			urlSlug := m[1] // e.g. "Detect_magic"
			engName := html.UnescapeString(m[2]) // e.g. "detect magic"

			spellID := urlSlugToEngName(urlSlug)

			spell := SpellKnown{
				SpellID: spellID,
				Name:    titleCase(engName),
			}

			if info, ok := spellMap[spellID]; ok {
				spell.Name = info.Name
				spell.Level = info.Level
				spell.School = info.School
			}

			spells = append(spells, spell)
		}
		return spells
	}

	// Strategy 2: <a href="/spells/Xxx">Russian name</a> (no brackets)
	linkMatches := spellLinkAnyRe.FindAllStringSubmatch(sectionHTML, -1)
	if len(linkMatches) > 0 {
		for _, m := range linkMatches {
			urlSlug := m[1]
			innerHTML := m[2]

			spellID := urlSlugToEngName(urlSlug)

			// Strip inner HTML tags, entities, and parenthetical notes
			name := htmlTagRe.ReplaceAllString(innerHTML, "")
			name = html.UnescapeString(name)
			name = parenthRe.ReplaceAllString(name, "")
			name = strings.TrimSpace(name)

			spell := SpellKnown{
				SpellID: spellID,
				Name:    name,
			}

			if info, ok := spellMap[spellID]; ok {
				spell.Name = info.Name
				spell.Level = info.Level
				spell.School = info.School
			}

			spells = append(spells, spell)
		}
		return spells
	}

	// Strategy 3: bare <em>text</em> with no links (rare)
	emMatches := bareEmRe.FindAllStringSubmatch(sectionHTML, -1)
	for _, m := range emMatches {
		name := strings.TrimSpace(m[1])
		name = parenthRe.ReplaceAllString(name, "")
		name = strings.TrimSpace(name)
		if name != "" && !usageAtWillRe.MatchString(name) && !usagePerDayRe.MatchString(name) {
			spells = append(spells, SpellKnown{Name: name})
		}
	}

	return spells
}

// urlSlugToEngName converts URL slug to spell engName.
// e.g. "Detect_magic" → "detect-magic", "Wall_of_fire" → "wall-of-fire"
func urlSlugToEngName(slug string) string {
	s := strings.ToLower(slug)
	s = strings.TrimRight(s, "/")
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, "%20", "-")
	return s
}

// titleCase converts "detect magic" to "Detect Magic".
func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func printInnate(name string, innate *InnateSpellcasting) {
	fmt.Printf("\n=== %s ===\n", name)
	fmt.Printf("  Ability: %s, DC: %d", innate.Ability, innate.SpellSaveDC)
	if innate.SpellAttackBonus > 0 {
		fmt.Printf(", Attack: +%d", innate.SpellAttackBonus)
	}
	fmt.Println()

	if innate.Note != "" {
		fmt.Printf("  Note: %s\n", innate.Note)
	}

	if len(innate.AtWill) > 0 {
		fmt.Printf("  At will:\n")
		for _, s := range innate.AtWill {
			fmt.Printf("    - %s (id=%s, lvl=%d)\n", s.Name, s.SpellID, s.Level)
		}
	}

	// Sort PerDay keys for deterministic output
	if len(innate.PerDay) > 0 {
		keys := make([]int, 0, len(innate.PerDay))
		for k := range innate.PerDay {
			keys = append(keys, k)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(keys)))
		for _, uses := range keys {
			spells := innate.PerDay[uses]
			fmt.Printf("  %d/day:\n", uses)
			for _, s := range spells {
				fmt.Printf("    - %s (id=%s, lvl=%d)\n", s.Name, s.SpellID, s.Level)
			}
		}
	}

	j, _ := json.MarshalIndent(innate, "  ", "  ")
	fmt.Printf("  JSON: %s\n", j)
}

func printStats(stats *MigrationStats) {
	fmt.Println("\n========== MIGRATION STATS ==========")
	fmt.Printf("Total processed:     %d\n", stats.Total)
	fmt.Printf("Modified:            %d\n", stats.Modified)
	fmt.Printf("Written to DB:       %d\n", stats.Written)
	fmt.Printf("Skipped (already):   %d\n", stats.Skipped)
	fmt.Printf("Errors:              %d\n", stats.Errors)

	if len(stats.ParseErrors) > 0 {
		fmt.Println("\n--- Parse errors ---")
		for _, e := range stats.ParseErrors {
			fmt.Printf("  %s: %s\n", e.Name, e.Reason)
		}
	}
}
