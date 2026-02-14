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
// Rolls attack + damage, compares vs target AC, applies resistance, and mutates HP.
func resolveWeaponAttack(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	encounterID string,
	charBase *models.CharacterBase,
	derived *models.DerivedStats,
	participant *models.ParticipantFull,
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

	resp := &models.ActionResponse{
		RollResult: attackResult,
	}

	// If no target, just roll attack + damage (no hit check)
	if cmd.TargetID == "" {
		damageRoll, err := rollWeaponDamage(weapon, abilityMod, isCrit)
		if err != nil {
			return nil, fmt.Errorf("roll weapon damage: %w", err)
		}
		damageRoll.DamageType = weapon.DamageType

		resp.DamageRolls = []models.ActionRollResult{*damageRoll}
		resp.Summary = fmt.Sprintf("%s attacks with %s: %d to hit",
			charBase.Name, weapon.Name, attackTotal)
		if isCrit {
			resp.Summary += " (CRITICAL HIT!)"
		}
		resp.Summary += fmt.Sprintf(", %d %s damage", damageRoll.Total, weapon.DamageType)

		return resp, nil
	}

	// Target provided — resolve hit/miss
	target, _, err := ed.FindParticipantByInstanceID(cmd.TargetID)
	if err != nil {
		// Target not found — still return the rolls, just don't mutate
		damageRoll, dErr := rollWeaponDamage(weapon, abilityMod, isCrit)
		if dErr != nil {
			return nil, fmt.Errorf("roll weapon damage: %w", dErr)
		}
		damageRoll.DamageType = weapon.DamageType
		resp.DamageRolls = []models.ActionRollResult{*damageRoll}
		resp.Summary = fmt.Sprintf("%s attacks with %s: %d to hit, %d %s damage",
			charBase.Name, weapon.Name, attackTotal, damageRoll.Total, weapon.DamageType)
		return resp, nil
	}

	ts, err := loadTargetStats(ctx, uc, target)
	if err != nil {
		// Can't load target stats — fall back to rolling without hit check
		l.UsecasesWarn(err, userID, map[string]any{"targetID": cmd.TargetID})
		damageRoll, dErr := rollWeaponDamage(weapon, abilityMod, isCrit)
		if dErr != nil {
			return nil, fmt.Errorf("roll weapon damage: %w", dErr)
		}
		damageRoll.DamageType = weapon.DamageType
		resp.DamageRolls = []models.ActionRollResult{*damageRoll}
		resp.Summary = fmt.Sprintf("%s attacks with %s: %d to hit, %d %s damage",
			charBase.Name, weapon.Name, attackTotal, damageRoll.Total, weapon.DamageType)
		return resp, nil
	}

	// D&D 5e hit rules: nat 1 always misses, nat 20 always hits, otherwise compare vs AC
	hit := natural != 1 && (isCrit || attackTotal >= ts.AC)
	resp.Hit = &hit

	if !hit {
		resp.Summary = fmt.Sprintf("%s attacks %s with %s: %d to hit vs AC %d — MISS",
			charBase.Name, ts.Name, weapon.Name, attackTotal, ts.AC)
		return resp, nil
	}

	// Hit — roll damage
	damageRoll, err := rollWeaponDamage(weapon, abilityMod, isCrit)
	if err != nil {
		return nil, fmt.Errorf("roll weapon damage: %w", err)
	}

	// Apply resistance/vulnerability/immunity
	finalDamage, appliedMod := applyResistance(damageRoll.Total, weapon.DamageType, ts)
	damageRoll.DamageType = weapon.DamageType
	damageRoll.AppliedModifier = appliedMod
	damageRoll.FinalDamage = intPtr(finalDamage)

	resp.DamageRolls = []models.ActionRollResult{*damageRoll}

	resp.Summary = fmt.Sprintf("%s attacks %s with %s: %d to hit vs AC %d — HIT",
		charBase.Name, ts.Name, weapon.Name, attackTotal, ts.AC)
	if isCrit {
		resp.Summary = fmt.Sprintf("%s attacks %s with %s: %d to hit vs AC %d — CRITICAL HIT!",
			charBase.Name, ts.Name, weapon.Name, attackTotal, ts.AC)
	}
	resp.Summary += fmt.Sprintf(", %d %s damage", finalDamage, weapon.DamageType)
	if appliedMod != "normal" {
		resp.Summary += fmt.Sprintf(" (%s)", appliedMod)
	}

	// Apply final damage to target
	applyDamageToTarget(target, finalDamage)

	resp.StateChanges = []models.StateChange{{
		TargetID:    cmd.TargetID,
		HPDelta:     -finalDamage,
		Description: fmt.Sprintf("%s takes %d %s damage from %s", ts.Name, finalDamage, weapon.DamageType, weapon.Name),
	}}

	// Evaluate weapon triggers on hit
	if len(weapon.Triggers) > 0 {
		triggerResults, triggerChanges := applyTriggerResults(
			weapon.Triggers, weapon.Name,
			models.ItemTriggerOnHit, isCrit,
			target, participant,
		)
		resp.TriggerResults = triggerResults
		resp.StateChanges = append(resp.StateChanges, triggerChanges...)
		for _, tr := range triggerResults {
			if !tr.Skipped && tr.Description != "" {
				resp.Summary += fmt.Sprintf(", %s", tr.Description)
			}
		}
	}

	// Persist encounter data
	if err := persistEncounterData(ctx, uc, ed, encounterID); err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, fmt.Errorf("persist encounter: %w", err)
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
