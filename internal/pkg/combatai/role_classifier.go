package combatai

import (
	"strconv"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// ClassifyRole determines a creature's combat behavior archetype using a
// 7-step priority algorithm. First matching rule wins.
func ClassifyRole(creature models.Creature) CreatureRole {
	actions := actionsByCategory(creature.StructuredActions, models.ActionCategoryAction)

	hasMelee, hasRanged := attackTypes(actions)

	// 1. Caster: has Spellcasting with attack/damage spells.
	if creature.Spellcasting != nil && hasSpells(creature.Spellcasting) {
		return RoleCaster
	}

	// 2. Ranged: all actions are ranged, no melee.
	if hasRanged && !hasMelee {
		return RoleRanged
	}

	cr := parseCR(creature.ChallengeRating)

	// 3. Brute: high STR and high HP for its CR.
	if creature.Ability.Str >= 16 && float64(creature.Hits.Average) >= cr*15 {
		return RoleBrute
	}

	// 4. Skirmisher: DEX much higher than STR and fast.
	if creature.Ability.Dex >= creature.Ability.Str+4 && creature.Movement.Walk >= 40 {
		return RoleSkirmisher
	}

	// 5. Tank: very high AC or tanky HP with melee.
	if creature.ArmorClass >= 18 || (float64(creature.Hits.Average) >= cr*20 && hasMelee) {
		return RoleTank
	}

	// 6. Controller: >=2 actions with SavingThrow or ConditionEffect.
	if countControlActions(actions) >= 2 {
		return RoleController
	}

	// 7. Fallback.
	if hasRanged {
		return RoleRanged
	}
	return RoleBrute
}

// actionsByCategory filters StructuredActions by the given category.
func actionsByCategory(actions []models.StructuredAction, cat models.ActionCategory) []models.StructuredAction {
	var result []models.StructuredAction
	for i := range actions {
		if actions[i].Category == cat {
			result = append(result, actions[i])
		}
	}
	return result
}

// attackTypes returns whether the action list contains melee and/or ranged attacks.
func attackTypes(actions []models.StructuredAction) (hasMelee, hasRanged bool) {
	for i := range actions {
		if actions[i].Attack == nil {
			continue
		}
		switch actions[i].Attack.Type {
		case models.AttackRollMeleeWeapon, models.AttackRollMeleeSpell:
			hasMelee = true
		case models.AttackRollRangedWeapon, models.AttackRollRangedSpell:
			hasRanged = true
		}
	}
	return
}

// hasSpells returns true if the spellcasting has at least one leveled spell.
func hasSpells(sc *models.Spellcasting) bool {
	for level, spells := range sc.SpellsByLevel {
		if level >= 1 && len(spells) > 0 {
			return true
		}
	}
	return len(sc.Spells) > 0
}

// countControlActions counts actions that have a SavingThrow or a ConditionEffect.
func countControlActions(actions []models.StructuredAction) int {
	count := 0
	for i := range actions {
		if actions[i].SavingThrow != nil {
			count++
			continue
		}
		for j := range actions[i].Effects {
			if actions[i].Effects[j].Condition != nil {
				count++
				break
			}
		}
	}
	return count
}

// parseCR converts a ChallengeRating string to float64.
// Examples: "1/4" → 0.25, "1/2" → 0.5, "3" → 3.0, "0" → 0.0.
func parseCR(cr string) float64 {
	if parts := strings.SplitN(cr, "/", 2); len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den != 0 {
			return num / den
		}
	}
	v, err := strconv.ParseFloat(cr, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseProfBonus converts a ProficiencyBonus string like "+2" to int.
func parseProfBonus(pb string) int {
	s := strings.TrimPrefix(pb, "+")
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}
