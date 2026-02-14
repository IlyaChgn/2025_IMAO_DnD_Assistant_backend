// Package compute provides pure functions for deriving character stats
// from CharacterBase data. Go port of the frontend computeDerived.ts.
package compute

import (
	"math"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// ────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────

var abilityNames = []string{"str", "dex", "con", "int", "wis", "cha"}

func abilityModifier(score int) int {
	return int(math.Floor(float64(score-10) / 2))
}

func proficiencyBonus(totalLevel int) int {
	lvl := totalLevel
	if lvl < 1 {
		lvl = 1
	}
	return int(math.Ceil(float64(lvl)/4)) + 1
}

func totalLevel(base *models.CharacterBase) int {
	sum := 0
	for _, c := range base.Classes {
		sum += c.Level
	}
	return sum
}

// abilityScore returns the raw score for the given ability name.
func abilityScore(scores models.AbilityScores, name string) int {
	switch name {
	case "str":
		return scores.Str
	case "dex":
		return scores.Dex
	case "con":
		return scores.Con
	case "int":
		return scores.Int
	case "wis":
		return scores.Wis
	case "cha":
		return scores.Cha
	default:
		return 10
	}
}

// normalizeSkillName converts a skill string to the canonical form:
// lowercase, spaces replaced with underscores.
func normalizeSkillName(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), " ", "_")
}

// ────────────────────────────────────────────────────────────
// Main function
// ────────────────────────────────────────────────────────────

// ComputeDerived is a pure function: CharacterBase → DerivedStats.
// It computes all derived values from a character's base data.
func ComputeDerived(base *models.CharacterBase) *models.DerivedStats {
	tl := totalLevel(base)
	profBonus := proficiencyBonus(tl)

	// Ability modifiers
	abilityMods := make(map[string]int, 6)
	for _, name := range abilityNames {
		abilityMods[name] = abilityModifier(abilityScore(base.AbilityScores, name))
	}

	// Skills
	proficientSkills := make(map[string]bool, len(base.Proficiencies.Skills))
	for _, s := range base.Proficiencies.Skills {
		proficientSkills[normalizeSkillName(s)] = true
	}

	expertiseSkills := make(map[string]bool, len(base.Expertise))
	for _, s := range base.Expertise {
		expertiseSkills[normalizeSkillName(s)] = true
	}

	skillBonuses := make(map[string]models.BonusBreakdown, len(skillAbilities))
	for skill, ability := range skillAbilities {
		mod := abilityMods[ability]
		isProficient := proficientSkills[skill]
		isExpert := expertiseSkills[skill]

		prof := 0
		if isProficient {
			prof = profBonus
		}

		expertise := 0
		if isProficient && isExpert {
			expertise = profBonus
		}

		skillBonuses[skill] = models.BonusBreakdown{
			Total:       mod + prof + expertise,
			AbilityMod:  mod,
			Proficiency: prof,
			Expertise:   expertise,
			Other:       0,
		}
	}

	// Saving throws
	proficientSaves := make(map[string]bool, len(base.Proficiencies.SavingThrows))
	for _, s := range base.Proficiencies.SavingThrows {
		proficientSaves[strings.ToLower(string(s))] = true
	}

	saveBonuses := make(map[string]models.BonusBreakdown, 6)
	for _, name := range abilityNames {
		mod := abilityMods[name]
		isProficient := proficientSaves[name]

		prof := 0
		if isProficient {
			prof = profBonus
		}

		saveBonuses[name] = models.BonusBreakdown{
			Total:       mod + prof,
			AbilityMod:  mod,
			Proficiency: prof,
			Expertise:   0,
			Other:       0,
		}
	}

	// AC
	armorClass := 10 + abilityMods["dex"]
	if base.ArmorClassOverride != nil {
		armorClass = *base.ArmorClassOverride
	}

	// HP
	conMod := abilityMods["con"]
	var maxHp int
	if base.HitPoints.MaxOverride != nil {
		maxHp = *base.HitPoints.MaxOverride
	} else {
		diceSum := 0
		for _, v := range base.HitPoints.HitDiceRolls {
			diceSum += v
		}
		maxHp = diceSum + conMod*tl
	}

	// Speed
	speed := models.SpeedDerived{Walk: base.BaseSpeed}

	// Initiative
	initiative := models.InitiativeDerived{Bonus: abilityMods["dex"]}

	// Passive checks
	passivePerception := 10 + skillBonuses["perception"].Total
	passiveInvestigation := 10 + skillBonuses["investigation"].Total
	passiveInsight := 10 + skillBonuses["insight"].Total

	// Spellcasting
	spellcasting := computeSpellcasting(base, profBonus, abilityMods)

	return &models.DerivedStats{
		AbilityModifiers:     abilityMods,
		ProficiencyBonus:     profBonus,
		SkillBonuses:         skillBonuses,
		SaveBonuses:          saveBonuses,
		ArmorClass:           armorClass,
		MaxHp:                maxHp,
		Speed:                speed,
		Initiative:           initiative,
		PassivePerception:    passivePerception,
		PassiveInvestigation: passiveInvestigation,
		PassiveInsight:       passiveInsight,
		Resistances:          []string{},
		Immunities:           []string{},
		Vulnerabilities:      []string{},
		Spellcasting:         spellcasting,
	}
}
