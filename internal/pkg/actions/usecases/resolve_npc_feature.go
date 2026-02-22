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

// resolveNpcUseFeature handles the "use_feature" action type for NPC participants.
// Handles both attack-based and save-based StructuredActions (breath weapons, eye rays, etc.).
func resolveNpcUseFeature(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	encounterID string,
	creature *models.Creature,
	actorName string,
	participant *models.ParticipantFull,
	ed *EncounterData,
	userID int,
) (*models.ActionResponse, error) {
	l := logger.FromContext(ctx)

	if cmd.FeatureID == "" {
		return nil, apperrors.MissingFeatureIDErr
	}

	// Find the structured action on the creature
	action := findStructuredAction(creature, cmd.FeatureID)
	if action == nil {
		return nil, apperrors.ActionNotFoundErr
	}

	// Check and deduct resources
	resourceChanges, err := checkAndDeductNpcResource(action, &participant.RuntimeState.Resources)
	if err != nil {
		return nil, err
	}

	resp := &models.ActionResponse{
		StateChanges: resourceChanges,
		Summary:      fmt.Sprintf("%s uses %s", actorName, action.Name),
	}

	mutated := len(resourceChanges) > 0

	// Branch: attack-based or save-based action
	if action.Attack != nil {
		m := resolveNpcFeatureAttack(ctx, uc, cmd, action, actorName, participant, ed, resp, userID)
		mutated = mutated || m
	} else if action.SavingThrow != nil {
		m := resolveNpcFeatureSave(ctx, uc, cmd, action, actorName, participant, ed, resp, userID)
		mutated = mutated || m
	}

	// Apply non-damage effects (healing, movement descriptions)
	for _, eff := range action.Effects {
		if eff.Healing != nil {
			resolveNpcHealingEffect(eff.Healing, action.Name, cmd, participant, ed, resp)
			mutated = true
		}
		if eff.Movement != nil {
			resp.Summary += fmt.Sprintf(", %s %d ft", eff.Movement.Type, eff.Movement.Distance)
		}
		if eff.Description != "" {
			resp.Summary += fmt.Sprintf(", %s", eff.Description)
		}
	}

	// Persist if state was mutated
	if mutated {
		if pErr := persistEncounterData(ctx, uc, ed, encounterID); pErr != nil {
			l.UsecasesError(pErr, userID, map[string]any{"encounterID": encounterID})
			return nil, fmt.Errorf("persist encounter: %w", pErr)
		}
	}

	return resp, nil
}

// resolveNpcFeatureAttack handles attack-roll resolution for NPC features.
// Returns true if encounter state was mutated.
func resolveNpcFeatureAttack(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	action *models.StructuredAction,
	actorName string,
	participant *models.ParticipantFull,
	ed *EncounterData,
	resp *models.ActionResponse,
	userID int,
) bool {
	l := logger.FromContext(ctx)
	mutated := false

	attackBonus := action.Attack.Bonus
	natural, total, rolls := dice.RollD20(attackBonus, cmd.Advantage, cmd.Disadvantage)
	isCrit := natural == 20

	resp.RollResult = &models.ActionRollResult{
		Expression: fmt.Sprintf("1d20%+d", attackBonus),
		Rolls:      rolls,
		Modifier:   attackBonus,
		Total:      total,
		Natural:    natural,
	}

	// Load target stats if target is provided
	var ts *TargetStats
	if cmd.TargetID != "" {
		target, _, tErr := ed.FindParticipantByInstanceID(cmd.TargetID)
		if tErr == nil {
			loaded, lErr := loadTargetStats(ctx, uc, target)
			if lErr != nil {
				l.UsecasesWarn(lErr, userID, map[string]any{"targetID": cmd.TargetID})
			} else {
				ts = loaded
			}
		}
	}

	if ts != nil {
		hit := natural != 1 && (isCrit || total >= ts.AC)
		resp.Hit = &hit

		if hit {
			resp.Summary += fmt.Sprintf(", %d to hit vs AC %d — HIT", total, ts.AC)
			if isCrit {
				resp.Summary += " (CRITICAL!)"
			}
		} else {
			resp.Summary += fmt.Sprintf(", %d to hit vs AC %d — MISS", total, ts.AC)
			return mutated
		}
	} else {
		resp.Summary += fmt.Sprintf(", %d to hit", total)
	}

	// Roll damage
	damageRolls, totalDamage := rollNpcDamage(action.Attack.Damage, isCrit, ts)
	resp.DamageRolls = damageRolls
	if len(damageRolls) > 0 {
		resp.Summary += ", " + npcDamageSummary(damageRolls)
	}

	// Apply damage to target
	if cmd.TargetID != "" && totalDamage > 0 {
		target, _, tErr := ed.FindParticipantByInstanceID(cmd.TargetID)
		if tErr == nil {
			applyDamageToTarget(target, totalDamage)
			targetName := participantName(target)
			resp.StateChanges = append(resp.StateChanges, models.StateChange{
				TargetID:    cmd.TargetID,
				HPDelta:     -totalDamage,
				Description: fmt.Sprintf("%s takes %d damage from %s", targetName, totalDamage, action.Name),
			})
			mutated = true

			// Apply conditions on hit
			for _, eff := range action.Effects {
				if eff.Condition != nil {
					applyNpcConditionEffect(eff.Condition, action, participant, target, cmd.TargetID, resp)
				}
			}
		}
	}

	return mutated
}

// resolveNpcFeatureSave handles save-based resolution for NPC features (breath weapons, etc.).
// Supports multi-target saves. Returns true if encounter state was mutated.
func resolveNpcFeatureSave(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	action *models.StructuredAction,
	actorName string,
	participant *models.ParticipantFull,
	ed *EncounterData,
	resp *models.ActionResponse,
	userID int,
) bool {
	l := logger.FromContext(ctx)
	mutated := false
	st := action.SavingThrow
	ability := strings.ToLower(string(st.Ability))

	targetIDs := resolveTargetIDs(cmd)

	resp.Summary += fmt.Sprintf(", DC %d %s save", st.DC, strings.ToUpper(ability))

	if len(targetIDs) == 0 {
		// No targets — just report the DC
		return mutated
	}

	// Roll damage once (shared across all targets for area effects)
	var baseDamageRolls []models.ActionRollResult
	var baseTotalDamage int
	if len(st.Damage) > 0 {
		baseDamageRolls, baseTotalDamage = rollNpcDamage(st.Damage, false, nil)
		resp.DamageRolls = baseDamageRolls
	}

	// Process each target individually
	for _, targetID := range targetIDs {
		target, _, tErr := ed.FindParticipantByInstanceID(targetID)
		if tErr != nil {
			l.UsecasesWarn(tErr, userID, map[string]any{"targetID": targetID, "phase": "npc_feature_save"})
			continue
		}

		ts, tsErr := loadTargetStats(ctx, uc, target)
		if tsErr != nil {
			l.UsecasesWarn(tsErr, userID, map[string]any{"targetID": targetID})
			continue
		}

		// Roll target's saving throw
		saveBonus := ts.SaveBonuses[ability]
		saveNatural, saveTotal, saveRolls := dice.RollD20(saveBonus, false, false)
		saved := saveTotal >= st.DC

		resp.StateChanges = append(resp.StateChanges, models.StateChange{
			Description: fmt.Sprintf("%s %s save: %d (1d20%+d = [%s])",
				ts.Name, strings.ToUpper(ability), saveTotal, saveBonus, formatRolls(saveRolls)),
		})

		if saved {
			resp.Summary += fmt.Sprintf("; %s saves (%d)", ts.Name, saveNatural)
		} else {
			resp.Summary += fmt.Sprintf("; %s fails (%d)", ts.Name, saveNatural)
		}

		// Calculate damage for this target
		targetDamage := baseTotalDamage

		if saved {
			onSuccess := strings.ToLower(st.OnSuccess)
			if strings.Contains(onSuccess, "half") {
				targetDamage = targetDamage / 2
			} else {
				targetDamage = 0
			}
		}

		// Apply per-damage-type resistance for this target
		if targetDamage > 0 && len(baseDamageRolls) > 0 {
			adjustedTotal := 0
			for _, dr := range baseDamageRolls {
				rollDmg := dr.Total
				if dr.FinalDamage != nil {
					rollDmg = *dr.FinalDamage
				}
				if saved {
					onSuccess := strings.ToLower(st.OnSuccess)
					if strings.Contains(onSuccess, "half") {
						rollDmg = rollDmg / 2
					} else {
						rollDmg = 0
					}
				}
				if dr.DamageType != "" {
					adjusted, _ := applyResistance(rollDmg, dr.DamageType, ts)
					rollDmg = adjusted
				}
				adjustedTotal += rollDmg
			}
			targetDamage = adjustedTotal
		}

		// Apply damage to this target
		if targetDamage > 0 {
			applyDamageToTarget(target, targetDamage)
			resp.StateChanges = append(resp.StateChanges, models.StateChange{
				TargetID:    targetID,
				HPDelta:     -targetDamage,
				Description: fmt.Sprintf("%s takes %d damage from %s", ts.Name, targetDamage, action.Name),
			})
			mutated = true
		}

		// Apply conditions on failed save
		if !saved {
			for _, eff := range action.Effects {
				if eff.Condition != nil {
					applyNpcConditionEffect(eff.Condition, action, participant, target, targetID, resp)
					mutated = true
				}
			}
		}
	}

	return mutated
}

// resolveNpcHealingEffect rolls healing from an ActionEffect's HealingEffect.
func resolveNpcHealingEffect(
	healing *models.HealingEffect,
	actionName string,
	cmd *models.ActionCommand,
	participant *models.ParticipantFull,
	ed *EncounterData,
	resp *models.ActionResponse,
) {
	diceType := strings.TrimPrefix(healing.DiceType, "d")
	if healing.DiceCount <= 0 || diceType == "" {
		return
	}

	expr := fmt.Sprintf("%dd%s", healing.DiceCount, diceType)
	if healing.Bonus != 0 {
		expr = fmt.Sprintf("%s%+d", expr, healing.Bonus)
	}
	result, err := dice.Roll(expr)
	if err != nil {
		return
	}

	resp.HealingRolls = append(resp.HealingRolls, models.ActionRollResult{
		Expression: expr,
		Rolls:      result.Rolls,
		Modifier:   result.Modifier,
		Total:      result.Total,
	})
	resp.Summary += fmt.Sprintf(", %d healing", result.Total)

	// Apply healing: if target is specified, heal target; otherwise heal self
	targetID := cmd.TargetID
	if targetID == "" {
		targetID = participant.InstanceID
	}
	target, _, tErr := ed.FindParticipantByInstanceID(targetID)
	if tErr == nil && result.Total > 0 {
		applyHealToParticipant(target, result.Total)
		targetName := participantName(target)
		resp.StateChanges = append(resp.StateChanges, models.StateChange{
			TargetID:    targetID,
			HPDelta:     result.Total,
			Description: fmt.Sprintf("%s heals %d HP from %s", targetName, result.Total, actionName),
		})
	}
}
