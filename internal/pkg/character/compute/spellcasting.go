package compute

import (
	"fmt"
	"math"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// ────────────────────────────────────────────────────────────
// Caster type enum
// ────────────────────────────────────────────────────────────

type casterType string

const (
	casterFull casterType = "full"
	casterHalf casterType = "half"
	casterPact casterType = "pact"
)

// ────────────────────────────────────────────────────────────
// Spell slot tables (hardcoded from PHB)
// ────────────────────────────────────────────────────────────

// Index = class level, value = [1st, 2nd, ..., 9th]
var fullCasterSlots = [][]int{
	/*  0 */ {},
	/*  1 */ {2},
	/*  2 */ {3},
	/*  3 */ {4, 2},
	/*  4 */ {4, 3},
	/*  5 */ {4, 3, 2},
	/*  6 */ {4, 3, 3},
	/*  7 */ {4, 3, 3, 1},
	/*  8 */ {4, 3, 3, 2},
	/*  9 */ {4, 3, 3, 3, 1},
	/* 10 */ {4, 3, 3, 3, 2},
	/* 11 */ {4, 3, 3, 3, 2, 1},
	/* 12 */ {4, 3, 3, 3, 2, 1},
	/* 13 */ {4, 3, 3, 3, 2, 1, 1},
	/* 14 */ {4, 3, 3, 3, 2, 1, 1},
	/* 15 */ {4, 3, 3, 3, 2, 1, 1, 1},
	/* 16 */ {4, 3, 3, 3, 2, 1, 1, 1},
	/* 17 */ {4, 3, 3, 3, 2, 1, 1, 1, 1},
	/* 18 */ {4, 3, 3, 3, 3, 1, 1, 1, 1},
	/* 19 */ {4, 3, 3, 3, 3, 2, 1, 1, 1},
	/* 20 */ {4, 3, 3, 3, 3, 2, 2, 1, 1},
}

var halfCasterSlots = [][]int{
	/*  0 */ {},
	/*  1 */ {},
	/*  2 */ {2},
	/*  3 */ {3},
	/*  4 */ {3},
	/*  5 */ {4, 2},
	/*  6 */ {4, 2},
	/*  7 */ {4, 3},
	/*  8 */ {4, 3},
	/*  9 */ {4, 3, 2},
	/* 10 */ {4, 3, 2},
	/* 11 */ {4, 3, 3},
	/* 12 */ {4, 3, 3},
	/* 13 */ {4, 3, 3, 1},
	/* 14 */ {4, 3, 3, 1},
	/* 15 */ {4, 3, 3, 2},
	/* 16 */ {4, 3, 3, 2},
	/* 17 */ {4, 3, 3, 3, 1},
	/* 18 */ {4, 3, 3, 3, 1},
	/* 19 */ {4, 3, 3, 3, 2},
	/* 20 */ {4, 3, 3, 3, 2},
}

// [slot count, slot level]
var pactMagicSlots = [][2]int{
	/*  0 */ {0, 0},
	/*  1 */ {1, 1},
	/*  2 */ {2, 1},
	/*  3 */ {2, 2},
	/*  4 */ {2, 2},
	/*  5 */ {2, 3},
	/*  6 */ {2, 3},
	/*  7 */ {2, 4},
	/*  8 */ {2, 4},
	/*  9 */ {2, 5},
	/* 10 */ {2, 5},
	/* 11 */ {3, 5},
	/* 12 */ {3, 5},
	/* 13 */ {3, 5},
	/* 14 */ {3, 5},
	/* 15 */ {3, 5},
	/* 16 */ {3, 5},
	/* 17 */ {4, 5},
	/* 18 */ {4, 5},
	/* 19 */ {4, 5},
	/* 20 */ {4, 5},
}

// ────────────────────────────────────────────────────────────
// Class configs (hardcoded from PHB)
// ────────────────────────────────────────────────────────────

type classConfig struct {
	casterType           casterType
	spellcastingAbility  string
	preparedCountFormula string // "ability_mod_plus_level" | "ability_mod_plus_half_level" | ""
	cantripsKnownTable   []int  // index = class level
}

var classConfigs = map[string]classConfig{
	"wizard": {
		casterType:           casterFull,
		spellcastingAbility:  "int",
		preparedCountFormula: "ability_mod_plus_level",
		cantripsKnownTable:   []int{0, 3, 3, 3, 4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5},
	},
	"cleric": {
		casterType:           casterFull,
		spellcastingAbility:  "wis",
		preparedCountFormula: "ability_mod_plus_level",
		cantripsKnownTable:   []int{0, 3, 3, 3, 4, 4, 4, 4, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5},
	},
	"druid": {
		casterType:           casterFull,
		spellcastingAbility:  "wis",
		preparedCountFormula: "ability_mod_plus_level",
		cantripsKnownTable:   []int{0, 2, 2, 2, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},
	},
	"bard": {
		casterType:          casterFull,
		spellcastingAbility: "cha",
		cantripsKnownTable:  []int{0, 2, 2, 2, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},
	},
	"sorcerer": {
		casterType:          casterFull,
		spellcastingAbility: "cha",
		cantripsKnownTable:  []int{0, 4, 4, 4, 5, 5, 5, 5, 5, 5, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6},
	},
	"warlock": {
		casterType:          casterPact,
		spellcastingAbility: "cha",
		cantripsKnownTable:  []int{0, 2, 2, 2, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4},
	},
	"paladin": {
		casterType:           casterHalf,
		spellcastingAbility:  "cha",
		preparedCountFormula: "ability_mod_plus_half_level",
		cantripsKnownTable:   []int{},
	},
	"ranger": {
		casterType:          casterHalf,
		spellcastingAbility: "wis",
		cantripsKnownTable:  []int{},
	},
}

// ────────────────────────────────────────────────────────────
// Slot lookup helpers
// ────────────────────────────────────────────────────────────

func clampLevel(level int) int {
	if level < 0 {
		return 0
	}
	if level > 20 {
		return 20
	}
	return level
}

func getSpellSlots(ct casterType, classLevel int) map[int]int {
	var table [][]int
	switch ct {
	case casterFull:
		table = fullCasterSlots
	case casterHalf:
		table = halfCasterSlots
	default:
		return map[int]int{}
	}

	level := clampLevel(classLevel)
	slots := table[level]
	result := make(map[int]int, len(slots))
	for i, count := range slots {
		result[i+1] = count
	}
	return result
}

func getPactMagicSlots(classLevel int) (count, slotLevel int) {
	level := clampLevel(classLevel)
	entry := pactMagicSlots[level]
	return entry[0], entry[1]
}

// multiclass caster level (PHB p.164)
type multiclassCasterEntry struct {
	ct         casterType
	classLevel int
}

func computeMulticlassCasterLevel(entries []multiclassCasterEntry) int {
	total := 0
	for _, e := range entries {
		lvl := e.classLevel
		if lvl < 0 {
			lvl = 0
		}
		switch e.ct {
		case casterFull:
			total += lvl
		case casterHalf:
			total += int(math.Floor(float64(lvl) / 2))
		// "third" casters (Eldritch Knight, Arcane Trickster) not yet supported
		case casterPact:
			// Warlock pact slots are separate
		}
	}
	if total < 0 {
		return 0
	}
	if total > 20 {
		return 20
	}
	return total
}

func getMulticlassSpellSlots(entries []multiclassCasterEntry) map[int]int {
	casterLevel := computeMulticlassCasterLevel(entries)
	if casterLevel == 0 {
		return map[int]int{}
	}
	slots := fullCasterSlots[casterLevel]
	result := make(map[int]int, len(slots))
	for i, count := range slots {
		result[i+1] = count
	}
	return result
}

// ────────────────────────────────────────────────────────────
// Main spellcasting computation
// ────────────────────────────────────────────────────────────

func computeSpellcasting(base *models.CharacterBase, profBonus int,
	abilityModifiers map[string]int) *models.SpellcastingDerived {

	if base.Spellcasting == nil {
		return nil
	}

	abilityName := strings.ToLower(string(base.Spellcasting.Ability))
	abilityMod := abilityModifiers[abilityName]

	// DC and attack bonus
	spellDC := 8 + profBonus + abilityMod
	spellAttack := profBonus + abilityMod

	dcBreakdown := fmt.Sprintf("8 + %d(prof) + %d(mod) = %d", profBonus, abilityMod, spellDC)

	sign := "+"
	if spellAttack < 0 {
		sign = ""
	}
	attackBreakdown := fmt.Sprintf("%d(prof) + %d(mod) = %s%d", profBonus, abilityMod, sign, spellAttack)

	// Collect caster classes
	type casterClassEntry struct {
		config classConfig
		level  int
	}
	var casterClasses []casterClassEntry
	for _, c := range base.Classes {
		cfg, ok := classConfigs[strings.ToLower(c.ClassName)]
		if ok {
			casterClasses = append(casterClasses, casterClassEntry{config: cfg, level: c.Level})
		}
	}

	maxSpellSlots := map[int]int{}
	var pactMagic *models.PactMagicDerived

	if len(casterClasses) == 1 {
		// Single-class: use class-specific table
		entry := casterClasses[0]
		if entry.config.casterType == casterPact {
			count, slotLevel := getPactMagicSlots(entry.level)
			pactMagic = &models.PactMagicDerived{MaxSlots: count, SlotLevel: slotLevel}
		} else {
			maxSpellSlots = getSpellSlots(entry.config.casterType, entry.level)
		}
	} else if len(casterClasses) > 1 {
		// Multiclass: shared pool + separate pact magic
		var sharedEntries []multiclassCasterEntry
		for _, entry := range casterClasses {
			if entry.config.casterType == casterPact {
				count, slotLevel := getPactMagicSlots(entry.level)
				pactMagic = &models.PactMagicDerived{MaxSlots: count, SlotLevel: slotLevel}
			} else {
				sharedEntries = append(sharedEntries, multiclassCasterEntry{
					ct:         entry.config.casterType,
					classLevel: entry.level,
				})
			}
		}
		if len(sharedEntries) > 0 {
			maxSpellSlots = getMulticlassSpellSlots(sharedEntries)
		}
	}

	// Max cantrips known
	maxCantripsKnown := 0
	for _, entry := range casterClasses {
		table := entry.config.cantripsKnownTable
		if len(table) > 0 {
			idx := entry.level
			if idx >= len(table) {
				idx = len(table) - 1
			}
			maxCantripsKnown += table[idx]
		}
	}

	// Max prepared spells (for prepared casters, single-class only)
	var maxPreparedSpells *int
	if len(casterClasses) == 1 {
		entry := casterClasses[0]
		switch entry.config.preparedCountFormula {
		case "ability_mod_plus_level":
			v := abilityMod + entry.level
			if v < 1 {
				v = 1
			}
			maxPreparedSpells = &v
		case "ability_mod_plus_half_level":
			v := abilityMod + int(math.Floor(float64(entry.level)/2))
			if v < 1 {
				v = 1
			}
			maxPreparedSpells = &v
		}
	}

	return &models.SpellcastingDerived{
		SpellSaveDC:          spellDC,
		SpellAttackBonus:     spellAttack,
		SpellSaveDCBreakdown: dcBreakdown,
		SpellAttackBreakdown: attackBreakdown,
		MaxSpellSlots:        maxSpellSlots,
		PactMagic:            pactMagic,
		MaxPreparedSpells:    maxPreparedSpells,
		MaxCantripsKnown:     maxCantripsKnown,
	}
}
