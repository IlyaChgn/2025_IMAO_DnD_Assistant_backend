package usecases

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
)

// findStructuredAction searches creature.StructuredActions by ID.
func findStructuredAction(creature *models.Creature, actionID string) *models.StructuredAction {
	for i := range creature.StructuredActions {
		if creature.StructuredActions[i].ID == actionID {
			return &creature.StructuredActions[i]
		}
	}
	return nil
}

// npcActorName resolves the display name for an NPC participant.
func npcActorName(participant *models.ParticipantFull, creature *models.Creature) string {
	if participant.DisplayName != "" {
		return participant.DisplayName
	}
	if creature.Name.Eng != "" {
		return creature.Name.Eng
	}
	return participant.InstanceID
}

// checkAndDeductNpcResource validates and deducts Uses/Recharge/LegendaryCost
// for a StructuredAction. Returns state changes describing what was consumed.
func checkAndDeductNpcResource(
	action *models.StructuredAction,
	resources *models.ResourceState,
) ([]models.StateChange, error) {
	var stateChanges []models.StateChange

	// Recharge check
	if action.Recharge != nil {
		if resources.RechargeReady == nil {
			resources.RechargeReady = make(map[string]bool)
		}
		ready, exists := resources.RechargeReady[action.ID]
		// First use is always available; after use it must be recharged
		if exists && !ready {
			return nil, apperrors.RechargeNotReadyErr
		}
		resources.RechargeReady[action.ID] = false
		stateChanges = append(stateChanges, models.StateChange{
			Description: fmt.Sprintf("Used %s (recharge consumed)", action.Name),
		})
	}

	// Limited uses check
	if action.Uses != nil && action.Uses.Max > 0 {
		if resources.AbilityUses == nil {
			resources.AbilityUses = make(map[string]int)
		}
		remaining, exists := resources.AbilityUses[action.ID]
		if !exists {
			// First use: initialize from template max
			remaining = action.Uses.Max
			resources.AbilityUses[action.ID] = remaining
		}
		if remaining <= 0 {
			return nil, apperrors.FeatureUsesExhaustedErr
		}
		resources.AbilityUses[action.ID] = remaining - 1
		stateChanges = append(stateChanges, models.StateChange{
			FeatureUsed: action.ID,
			Description: fmt.Sprintf("Used %s (%d/%d uses remaining)",
				action.Name, remaining-1, action.Uses.Max),
		})
	}

	// Legendary action cost
	if action.LegendaryCost > 0 {
		if resources.LegendaryActions < action.LegendaryCost {
			return nil, apperrors.LegendaryActionsExhaustedErr
		}
		resources.LegendaryActions -= action.LegendaryCost
		stateChanges = append(stateChanges, models.StateChange{
			Description: fmt.Sprintf("Used %s (%d legendary action(s), %d remaining)",
				action.Name, action.LegendaryCost, resources.LegendaryActions),
		})
	}

	return stateChanges, nil
}

// rollNpcDamage rolls damage from a DamageRoll slice, handles crit doubling,
// and applies resistance/vulnerability/immunity per damage type.
// Returns the individual roll results and the total final damage.
func rollNpcDamage(
	damageRolls []models.DamageRoll,
	isCrit bool,
	ts *TargetStats,
) ([]models.ActionRollResult, int) {
	var results []models.ActionRollResult
	totalDamage := 0

	for _, dmg := range damageRolls {
		// Skip conditional damage for now (e.g., "extra 2d6 fire on crit only")
		if dmg.Condition != "" && !isCrit {
			continue
		}

		diceType := strings.TrimPrefix(dmg.DiceType, "d")
		if dmg.DiceCount <= 0 || diceType == "" {
			// Flat bonus only
			if dmg.Bonus > 0 {
				finalDamage := dmg.Bonus
				rollResult := models.ActionRollResult{
					Expression: fmt.Sprintf("%d", dmg.Bonus),
					Total:      dmg.Bonus,
					DamageType: dmg.DamageType,
				}
				if ts != nil && dmg.DamageType != "" {
					adjusted, appliedMod := applyResistance(finalDamage, dmg.DamageType, ts)
					rollResult.AppliedModifier = appliedMod
					rollResult.FinalDamage = intPtr(adjusted)
					finalDamage = adjusted
				} else {
					rollResult.FinalDamage = intPtr(finalDamage)
				}
				results = append(results, rollResult)
				totalDamage += finalDamage
			}
			continue
		}

		// Build dice expression
		expr := fmt.Sprintf("%dd%s", dmg.DiceCount, diceType)
		if dmg.Bonus != 0 {
			expr = fmt.Sprintf("%s%+d", expr, dmg.Bonus)
		}
		result, err := dice.Roll(expr)
		if err != nil {
			continue
		}

		rollResult := models.ActionRollResult{
			Expression: expr,
			Rolls:      result.Rolls,
			Modifier:   result.Modifier,
			Total:      result.Total,
			DamageType: dmg.DamageType,
		}

		finalDamage := result.Total

		// Crit: double the dice (not the modifier)
		if isCrit {
			sides, pErr := strconv.Atoi(diceType)
			if pErr == nil {
				critRolls, critTotal := dice.RollDice(dmg.DiceCount, sides)
				rollResult.Rolls = append(rollResult.Rolls, critRolls...)
				rollResult.Total += critTotal
				finalDamage += critTotal
			}
		}

		// Apply resistance/vulnerability/immunity
		if ts != nil && dmg.DamageType != "" {
			adjusted, appliedMod := applyResistance(finalDamage, dmg.DamageType, ts)
			rollResult.AppliedModifier = appliedMod
			rollResult.FinalDamage = intPtr(adjusted)
			finalDamage = adjusted
		} else {
			rollResult.FinalDamage = intPtr(finalDamage)
		}

		results = append(results, rollResult)
		totalDamage += finalDamage
	}

	return results, totalDamage
}

// npcSpellSource describes where a spell was found on a creature.
type npcSpellSource struct {
	Spell            *models.SpellKnown
	IsInnate         bool
	IsAtWill         bool
	PerDayLimit      int // 0 if at-will or regular slot-based
	SpellSaveDC      int
	SpellAttackBonus int
	CasterLevel      int
}

// findNpcSpell searches a creature's InnateSpellcasting (at-will first, then per-day)
// and then Spellcasting (SpellsByLevel, flat Spells list) for a matching spell.
func findNpcSpell(creature *models.Creature, spellID string) (*npcSpellSource, error) {
	// Search innate spellcasting first (prefer at-will / per-day over slot-based)
	if inn := creature.InnateSpellcasting; inn != nil {
		// At-will spells
		for i := range inn.AtWill {
			if matchesNpcSpell(&inn.AtWill[i], spellID) {
				return &npcSpellSource{
					Spell:            &inn.AtWill[i],
					IsInnate:         true,
					IsAtWill:         true,
					SpellSaveDC:      inn.SpellSaveDC,
					SpellAttackBonus: inn.SpellAttackBonus,
				}, nil
			}
		}

		// Per-day spells (3/day, 2/day, 1/day)
		for limit, spells := range inn.PerDay {
			for i := range spells {
				if matchesNpcSpell(&spells[i], spellID) {
					return &npcSpellSource{
						Spell:            &spells[i],
						IsInnate:         true,
						PerDayLimit:      limit,
						SpellSaveDC:      inn.SpellSaveDC,
						SpellAttackBonus: inn.SpellAttackBonus,
					}, nil
				}
			}
		}
	}

	// Search regular spellcasting
	if sc := creature.Spellcasting; sc != nil {
		// SpellsByLevel map
		for _, spells := range sc.SpellsByLevel {
			for i := range spells {
				if matchesNpcSpell(&spells[i], spellID) {
					return &npcSpellSource{
						Spell:            &spells[i],
						SpellSaveDC:      sc.SpellSaveDC,
						SpellAttackBonus: sc.SpellAttackBonus,
						CasterLevel:      sc.CasterLevel,
					}, nil
				}
			}
		}

		// Flat spell list
		for i := range sc.Spells {
			if matchesNpcSpell(&sc.Spells[i], spellID) {
				return &npcSpellSource{
					Spell:            &sc.Spells[i],
					SpellSaveDC:      sc.SpellSaveDC,
					SpellAttackBonus: sc.SpellAttackBonus,
					CasterLevel:      sc.CasterLevel,
				}, nil
			}
		}
	}

	return nil, apperrors.NpcSpellNotKnownErr
}

// matchesNpcSpell checks if a SpellKnown matches the given spellID by ID or name.
func matchesNpcSpell(spell *models.SpellKnown, spellID string) bool {
	if spell.SpellID != "" && spell.SpellID == spellID {
		return true
	}
	if strings.EqualFold(spell.Name, spellID) {
		return true
	}
	return false
}

// creatureSkillBonus looks up a skill bonus from creature.Skills.
// Falls back to the ability modifier if the creature is not proficient.
func creatureSkillBonus(creature *models.Creature, skill string, ability string) int {
	for _, s := range creature.Skills {
		if strings.EqualFold(s.Name, skill) {
			return s.Value
		}
	}
	return creatureAbilityModifier(creature, ability)
}

// npcDamageSummary appends a damage summary string for NPC damage rolls.
func npcDamageSummary(damageRolls []models.ActionRollResult) string {
	var parts []string
	for _, dr := range damageRolls {
		finalDmg := dr.Total
		if dr.FinalDamage != nil {
			finalDmg = *dr.FinalDamage
		}
		part := fmt.Sprintf("%d %s damage", finalDmg, dr.DamageType)
		if dr.AppliedModifier != "" && dr.AppliedModifier != "normal" {
			part += fmt.Sprintf(" (%s)", dr.AppliedModifier)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}
