package combatai

import (
	"strconv"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// ComputeExpectedDamage calculates expected damage for a StructuredAction
// against a target described by targetStats. Returns 0 if the action has
// neither an attack roll nor a saving throw.
func ComputeExpectedDamage(action models.StructuredAction, targetStats CombatantStats) float64 {
	if action.Attack != nil {
		return expectedAttackDamage(action.Attack, targetStats)
	}
	if action.SavingThrow != nil {
		return expectedSaveDamage(action.SavingThrow, targetStats)
	}
	return 0
}

// expectedAttackDamage computes EV for an attack roll action.
func expectedAttackDamage(attack *models.AttackRollData, targetStats CombatantStats) float64 {
	hitChance := float64(21-(targetStats.AC-attack.Bonus)) / 20.0
	hitChance = clamp(hitChance, 0.05, 0.95)

	avgDamage := avgDamageRolls(attack.Damage)

	// hit_chance * avg + 0.05 * avg (crit bonus: nat 20 adds extra dice ≈ avg again)
	return hitChance*avgDamage + 0.05*avgDamage
}

// expectedSaveDamage computes EV for a saving throw action.
func expectedSaveDamage(st *models.SavingThrowData, targetStats CombatantStats) float64 {
	saveAbility := string(st.Ability)
	targetSave := targetStats.SaveBonuses[saveAbility]

	failChance := float64(st.DC-targetSave-1) / 20.0
	failChance = clamp(failChance, 0.05, 0.95)

	avgDamage := avgDamageRolls(st.Damage)

	halfOnSuccess := strings.Contains(strings.ToLower(st.OnSuccess), "half")
	if halfOnSuccess {
		return failChance*avgDamage + (1-failChance)*avgDamage/2
	}
	return failChance * avgDamage
}

// EstimateSpellDamage provides a heuristic damage estimate for a spell
// when no full SpellDefinition is available. Uses level-based approximation.
func EstimateSpellDamage(spell models.SpellKnown, spellcasting models.Spellcasting, targetStats CombatantStats) float64 {
	if spell.QuickRef == nil {
		return 0
	}
	if spell.QuickRef.Range == "Self" {
		return 0
	}

	// Cantrip scaling.
	if spell.Level == 0 {
		base := 5.5 // ~1d10
		multiplier := 1
		switch {
		case spellcasting.CasterLevel >= 17:
			multiplier = 4
		case spellcasting.CasterLevel >= 11:
			multiplier = 3
		case spellcasting.CasterLevel >= 5:
			multiplier = 2
		}
		return base * float64(multiplier)
	}

	// Leveled spell — spell attack or save-based.
	if spellcasting.SpellAttackBonus > 0 {
		hitChance := float64(21-(targetStats.AC-spellcasting.SpellAttackBonus)) / 20.0
		hitChance = clamp(hitChance, 0.05, 0.95)
		estimatedDamage := float64(spell.Level)*3.5 + 3.0
		return hitChance * estimatedDamage
	}

	// Save-based spell: use DEX save as default approximation.
	saveBonus := targetStats.SaveBonuses["DEX"]
	failChance := float64(spellcasting.SpellSaveDC-saveBonus-1) / 20.0
	failChance = clamp(failChance, 0.05, 0.95)
	estimatedDamage := float64(spell.Level) * 4.5
	return failChance*estimatedDamage + (1-failChance)*estimatedDamage/2
}

// avgDamageRolls computes the average damage across a slice of DamageRoll.
// Entries with a non-empty Condition are skipped (conditional damage like
// "extra 2d6 fire on crit").
func avgDamageRolls(rolls []models.DamageRoll) float64 {
	var total float64
	for _, dr := range rolls {
		if dr.Condition != "" {
			continue
		}
		diceMax := parseDiceMax(dr.DiceType)
		total += float64(dr.DiceCount)*float64(diceMax+1)/2.0 + float64(dr.Bonus)
	}
	return total
}

// parseDiceMax extracts the maximum face value from a dice type string.
// "d6" → 6, "d8" → 8, "d10" → 10, "d12" → 12, "d20" → 20.
// Returns 0 on invalid input.
func parseDiceMax(diceType string) int {
	s := strings.TrimPrefix(strings.ToLower(diceType), "d")
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}
