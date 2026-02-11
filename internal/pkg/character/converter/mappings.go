package converter

import (
	"regexp"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// classNameMap maps Russian D&D 5e class names to English slugs.
// Keys are normalized (TrimSpace + lower-cased first word match).
var classNameMap = map[string]string{
	// Standard D&D 5e classes
	"бард":         "bard",
	"варвар":       "barbarian",
	"воин":         "fighter",
	"волшебник":    "wizard",
	"волшебница":   "wizard",   // feminine form
	"друид":        "druid",
	"жрец":         "cleric",
	"изобретатель": "artificer",
	"колдун":       "warlock",
	"монах":        "monk",
	"паладин":      "paladin",
	"плут":         "rogue",
	"следопыт":     "ranger",
	"чародей":      "sorcerer",
}

// MapClassName converts a Russian class name to an English slug.
// If the full name isn't found, tries matching just the first word (handles "Волшебница ШВ" etc.).
// Returns the mapped name and true if found, or the original name and false if not.
func MapClassName(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	lower := strings.ToLower(trimmed)
	if slug, ok := classNameMap[lower]; ok {
		return slug, true
	}

	// Try first word only (handles "Волшебница ШВ", "Плут Убийца", etc.)
	firstWord := strings.Fields(lower)[0]
	if slug, ok := classNameMap[firstWord]; ok {
		return slug, true
	}

	return trimmed, false
}

// abilityCodeMap maps LSS lowercase ability codes to AbilityType uppercase constants.
var abilityCodeMap = map[string]models.AbilityType{
	"str": models.AbilitySTR,
	"dex": models.AbilityDEX,
	"con": models.AbilityCON,
	"int": models.AbilityINT,
	"wis": models.AbilityWIS,
	"cha": models.AbilityCHA,
}

// MapAbilityCode converts an LSS lowercase ability code ("int") to AbilityType ("INT").
// Returns the mapped type and true if found, or empty string and false if not.
func MapAbilityCode(code string) (models.AbilityType, bool) {
	lower := strings.ToLower(strings.TrimSpace(code))
	at, ok := abilityCodeMap[lower]
	return at, ok
}

// abilityNameRuMap maps Russian ability names (as used in spellsInfo.base.value) to AbilityType.
var abilityNameRuMap = map[string]models.AbilityType{
	"сила":          models.AbilitySTR,
	"ловкость":      models.AbilityDEX,
	"телосложение":  models.AbilityCON,
	"интеллект":     models.AbilityINT,
	"мудрость":      models.AbilityWIS,
	"харизма":       models.AbilityCHA,
}

// MapAbilityNameRu converts a Russian ability name ("Мудрость") to AbilityType ("WIS").
func MapAbilityNameRu(name string) (models.AbilityType, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	at, ok := abilityNameRuMap[lower]
	return at, ok
}

// damageTypeMapRu maps Russian damage type words to English damage types.
var damageTypeMapRu = map[string]string{
	"рубящий":       "slashing",
	"колющий":       "piercing",
	"дробящий":      "bludgeoning",
	"огнём":         "fire",
	"огненный":      "fire",
	"холодом":       "cold",
	"молнией":       "lightning",
	"кислотой":      "acid",
	"ядом":          "poison",
	"некротический": "necrotic",
	"излучением":    "radiant",
	"психический":   "psychic",
	"звуком":        "thunder",
	"силовой":       "force",
}

// diceRegex matches Russian dice notation: 1к10, 2к6, 3к10+5, 4к4-2
var diceRegex = regexp.MustCompile(`(\d+)к(\d+)`)

// ParseDamageString parses a Russian damage notation string.
// Examples:
//
//	"1к10 рубящий"    → ("1d10", "slashing")
//	"3к10"            → ("3d10", "")
//	"4к4+10"          → ("4d4", "")
//	"2к6 + 1к8 огнём" → ("2d6", "fire") — takes first dice group
func ParseDamageString(s string) (dice, damageType string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}

	// Extract dice notation (first match)
	match := diceRegex.FindStringSubmatch(s)
	if len(match) >= 3 {
		dice = match[1] + "d" + match[2]
	}

	// Extract damage type — check each word against the map
	words := strings.Fields(strings.ToLower(s))
	for _, word := range words {
		// Clean punctuation
		word = strings.Trim(word, ".,;:!?()[]")
		if dt, ok := damageTypeMapRu[word]; ok {
			damageType = dt
			break
		}
	}

	return dice, damageType
}

// hitDieMap maps Russian hit die notation to English format.
var hitDieMap = map[string]string{
	"к6":  "d6",
	"к8":  "d8",
	"к10": "d10",
	"к12": "d12",
}

// ParseHitDie parses a Russian hit die string like "1к10" or "к8" into "d10" or "d8".
func ParseHitDie(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))

	// Try direct match for "к6", "к8", "к10", "к12"
	for ru, en := range hitDieMap {
		if strings.Contains(s, ru) {
			return en
		}
	}

	// Try regex for "1к10" format — extract just the die
	match := diceRegex.FindStringSubmatch(s)
	if len(match) >= 3 {
		return "d" + match[2]
	}

	return s
}

// skillNameMapRu maps Russian skill names to English slugs.
// Used for parsing proficiency text from LSS.
var skillNameMapRu = map[string]string{
	"акробатика":        "acrobatics",
	"анализ":            "investigation",
	"атлетика":          "athletics",
	"внимательность":    "perception",
	"выживание":         "survival",
	"выступление":       "performance",
	"запугивание":       "intimidation",
	"история":           "history",
	"ловкость рук":      "sleight_of_hand",
	"магия":             "arcana",
	"медицина":          "medicine",
	"обман":             "deception",
	"природа":           "nature",
	"проницательность":  "insight",
	"религия":           "religion",
	"скрытность":        "stealth",
	"убеждение":         "persuasion",
	"уход за животными": "animal_handling",
}

// MapSkillName converts a Russian skill name to its English slug.
func MapSkillName(ru string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(ru))
	en, ok := skillNameMapRu[lower]
	return en, ok
}
