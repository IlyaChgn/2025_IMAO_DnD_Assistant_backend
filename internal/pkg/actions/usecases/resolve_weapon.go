package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

// resolveWeaponAttack handles the "weapon_attack" action type.
// Rolls attack + damage, and if a target is provided mutates the target's HP.
func resolveWeaponAttack(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	encounterID string,
	charBase *models.CharacterBase,
	derived *models.DerivedStats,
	_ *models.ParticipantFull,
	ed *EncounterData,
	userID int,
) (*models.ActionResponse, error) {
	l := logger.FromContext(ctx)

	if cmd.WeaponID == "" {
		return nil, apperrors.MissingWeaponIDErr
	}

	// Find weapon on character
	weapon := findWeapon(charBase, cmd.WeaponID)
	if weapon == nil {
		return nil, apperrors.WeaponNotFoundErr
	}

	// Determine attack ability
	attackAbility := resolveAttackAbility(weapon, derived)
	abilityMod := derived.AbilityModifiers[attackAbility]
	profBonus := derived.ProficiencyBonus

	attackBonus := abilityMod + profBonus + weapon.MagicBonus

	// Roll attack
	natural, attackTotal, attackRolls := dice.RollD20(attackBonus, cmd.Advantage, cmd.Disadvantage)
	isCrit := natural == 20

	attackResult := &models.ActionRollResult{
		Expression: fmt.Sprintf("1d20%+d", attackBonus),
		Rolls:      attackRolls,
		Modifier:   attackBonus,
		Total:      attackTotal,
		Natural:    natural,
	}

	// Roll damage
	damageRoll, err := rollWeaponDamage(weapon, abilityMod, isCrit)
	if err != nil {
		return nil, fmt.Errorf("roll weapon damage: %w", err)
	}

	summary := fmt.Sprintf("%s attacks with %s: %d to hit",
		charBase.Name, weapon.Name, attackTotal)
	if isCrit {
		summary += " (CRITICAL HIT!)"
	}
	summary += fmt.Sprintf(", %d %s damage", damageRoll.Total, weapon.DamageType)

	resp := &models.ActionResponse{
		RollResult:  attackResult,
		DamageRolls: []models.ActionRollResult{*damageRoll},
		Summary:     summary,
	}

	// Apply damage to target if provided
	if cmd.TargetID != "" {
		target, _, err := ed.FindParticipantByInstanceID(cmd.TargetID)
		if err != nil {
			// Target not found — still return the rolls, just don't mutate
			return resp, nil
		}

		applyDamageToTarget(target, damageRoll.Total)

		resp.StateChanges = []models.StateChange{{
			TargetID:    cmd.TargetID,
			HPDelta:     -damageRoll.Total,
			Description: fmt.Sprintf("%s takes %d %s damage from %s", target.DisplayName, damageRoll.Total, weapon.DamageType, weapon.Name),
		}}

		// Persist encounter data
		if err := persistEncounterData(ctx, uc, ed, encounterID); err != nil {
			l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
			return nil, fmt.Errorf("persist encounter: %w", err)
		}
	}

	return resp, nil
}

func findWeapon(charBase *models.CharacterBase, weaponID string) *models.WeaponDef {
	for i := range charBase.Weapons {
		if charBase.Weapons[i].ID == weaponID {
			return &charBase.Weapons[i]
		}
	}

	return nil
}

// resolveAttackAbility determines which ability modifier to use for a weapon attack.
func resolveAttackAbility(weapon *models.WeaponDef, derived *models.DerivedStats) string {
	if weapon.AbilityOverride != "" {
		return strings.ToLower(weapon.AbilityOverride)
	}

	hasFinesse := false
	for _, p := range weapon.Properties {
		if strings.ToLower(p) == "finesse" {
			hasFinesse = true
			break
		}
	}

	if hasFinesse {
		strMod := derived.AbilityModifiers["str"]
		dexMod := derived.AbilityModifiers["dex"]
		if dexMod > strMod {
			return "dex"
		}

		return "str"
	}

	switch weapon.AttackType {
	case "ranged":
		return "dex"
	default:
		return "str"
	}
}

func rollWeaponDamage(weapon *models.WeaponDef, abilityMod int, isCrit bool) (*models.ActionRollResult, error) {
	result, err := dice.Roll(weapon.DamageDice)
	if err != nil {
		return nil, err
	}

	totalDamage := result.Total + abilityMod + weapon.MagicBonus

	if isCrit {
		// Roll extra dice for crit (double the dice, not the modifier)
		count, sides, _, err := dice.Parse(weapon.DamageDice)
		if err == nil {
			critRolls, critTotal := dice.RollDice(count, sides)
			result.Rolls = append(result.Rolls, critRolls...)
			totalDamage += critTotal
		}
	}

	modifier := abilityMod + weapon.MagicBonus + result.Modifier

	expr := weapon.DamageDice
	if modifier > 0 {
		expr = fmt.Sprintf("%s+%d", weapon.DamageDice, modifier)
	} else if modifier < 0 {
		expr = fmt.Sprintf("%s%d", weapon.DamageDice, modifier)
	}

	return &models.ActionRollResult{
		Expression: expr,
		Rolls:      result.Rolls,
		Modifier:   modifier,
		Total:      totalDamage,
	}, nil
}

// applyDamageToTarget subtracts damage from a participant's HP (temp HP first).
func applyDamageToTarget(target *models.ParticipantFull, damage int) {
	if target.CharacterRuntime != nil {
		remaining := damage
		if target.CharacterRuntime.TemporaryHP > 0 {
			if target.CharacterRuntime.TemporaryHP >= remaining {
				target.CharacterRuntime.TemporaryHP -= remaining
				return
			}
			remaining -= target.CharacterRuntime.TemporaryHP
			target.CharacterRuntime.TemporaryHP = 0
		}
		target.CharacterRuntime.CurrentHP -= remaining
		if target.CharacterRuntime.CurrentHP < 0 {
			target.CharacterRuntime.CurrentHP = 0
		}

		return
	}

	// Creature runtime state path
	remaining := damage
	if target.RuntimeState.TempHP > 0 {
		if target.RuntimeState.TempHP >= remaining {
			target.RuntimeState.TempHP -= remaining
			return
		}
		remaining -= target.RuntimeState.TempHP
		target.RuntimeState.TempHP = 0
	}
	target.RuntimeState.CurrentHP -= remaining
	if target.RuntimeState.CurrentHP < 0 {
		target.RuntimeState.CurrentHP = 0
	}
}

// FindParticipantByInstanceID searches by the InstanceID (encounter slot ID),
// used when targeting a specific creature/character in the encounter.
func (ed *EncounterData) FindParticipantByInstanceID(instanceID string) (*models.ParticipantFull, int, error) {
	for i := range ed.Participants {
		p := &ed.Participants[i]
		if p.InstanceID == instanceID {
			return p, i, nil
		}
	}

	return nil, -1, apperrors.ParticipantNotFoundErr
}

// persistEncounterData marshals and saves the mutated encounter data.
func persistEncounterData(ctx context.Context, uc *actionsUsecases, ed *EncounterData, encounterID string) error {
	data, err := ed.Marshal()
	if err != nil {
		return err
	}

	return uc.encounterRepo.UpdateEncounter(ctx, data, encounterID)
}
