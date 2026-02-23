package usecases

import (
	"context"
	"fmt"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

// resolveNpcWeaponAttack handles the "weapon_attack" action type for NPC participants.
// Finds a StructuredAction by cmd.WeaponID, rolls attack + damage, applies to target.
func resolveNpcWeaponAttack(
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

	if cmd.WeaponID == "" {
		return nil, apperrors.MissingWeaponIDErr
	}

	// Find the structured action on the creature
	action := findStructuredAction(creature, cmd.WeaponID)
	if action == nil || action.Attack == nil {
		return nil, apperrors.ActionNotFoundErr
	}

	// Check and deduct resources (recharge, limited uses, legendary cost)
	resourceChanges, err := checkAndDeductNpcResource(action, &participant.RuntimeState.Resources)
	if err != nil {
		return nil, err
	}

	// Attack roll
	attackBonus := action.Attack.Bonus
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
		RollResult:   attackResult,
		StateChanges: resourceChanges,
	}

	mutated := len(resourceChanges) > 0

	// If no target, just roll attack + damage (no hit check)
	if cmd.TargetID == "" {
		damageRolls, _ := rollNpcDamage(action.Attack.Damage, isCrit, nil)
		resp.DamageRolls = damageRolls
		resp.Summary = fmt.Sprintf("%s attacks with %s: %d to hit",
			actorName, action.Name, attackTotal)
		if isCrit {
			resp.Summary += " (CRITICAL HIT!)"
		}
		if len(damageRolls) > 0 {
			resp.Summary += ", " + npcDamageSummary(damageRolls)
		}

		if mutated {
			if pErr := persistEncounterData(ctx, uc, ed, encounterID); pErr != nil {
				l.UsecasesError(pErr, userID, map[string]any{"encounterID": encounterID})
				return nil, fmt.Errorf("persist encounter: %w", pErr)
			}
		}

		return resp, nil
	}

	// Target provided — resolve hit/miss
	target, _, err := ed.FindParticipantByInstanceID(cmd.TargetID)
	if err != nil {
		// Target not found — still return the rolls, just don't mutate
		damageRolls, _ := rollNpcDamage(action.Attack.Damage, isCrit, nil)
		resp.DamageRolls = damageRolls
		resp.Summary = fmt.Sprintf("%s attacks with %s: %d to hit",
			actorName, action.Name, attackTotal)
		if len(damageRolls) > 0 {
			resp.Summary += ", " + npcDamageSummary(damageRolls)
		}

		if mutated {
			if pErr := persistEncounterData(ctx, uc, ed, encounterID); pErr != nil {
				l.UsecasesError(pErr, userID, map[string]any{"encounterID": encounterID})
				return nil, fmt.Errorf("persist encounter: %w", pErr)
			}
		}

		return resp, nil
	}

	ts, err := loadTargetStats(ctx, uc, target)
	if err != nil {
		// Can't load target stats — fall back to rolling without hit check
		l.UsecasesWarn(err, userID, map[string]any{"targetID": cmd.TargetID})
		damageRolls, _ := rollNpcDamage(action.Attack.Damage, isCrit, nil)
		resp.DamageRolls = damageRolls
		resp.Summary = fmt.Sprintf("%s attacks with %s: %d to hit",
			actorName, action.Name, attackTotal)
		if len(damageRolls) > 0 {
			resp.Summary += ", " + npcDamageSummary(damageRolls)
		}

		if mutated {
			if pErr := persistEncounterData(ctx, uc, ed, encounterID); pErr != nil {
				l.UsecasesError(pErr, userID, map[string]any{"encounterID": encounterID})
				return nil, fmt.Errorf("persist encounter: %w", pErr)
			}
		}

		return resp, nil
	}

	// Evaluate Shield reaction for NPC targets.
	// Skip on natural 20 (auto-hit) and natural 1 (auto-miss) — Shield has no effect.
	if uc.reactionEval != nil && !target.IsPlayerCharacter && !isCrit && natural != 1 {
		shieldResult, sErr := uc.reactionEval.EvaluateShield(ctx, ed, cmd.TargetID, attackTotal)
		if sErr != nil {
			l.UsecasesWarn(sErr, userID, map[string]any{"targetID": cmd.TargetID, "reaction": "shield"})
		}
		if sErr == nil && shieldResult != nil {
			applyShieldReaction(shieldResult, ed)
			ts.AC = shieldResult.NewEffectiveAC
			resp.ReactionSummary = append(resp.ReactionSummary, buildShieldSummary(shieldResult))
			mutated = true
		}
	}

	// D&D 5e hit rules: nat 1 always misses, nat 20 always hits, otherwise compare vs AC
	hit := natural != 1 && (isCrit || attackTotal >= ts.AC)
	resp.Hit = &hit

	if !hit {
		resp.Summary = fmt.Sprintf("%s attacks %s with %s: %d to hit vs AC %d — MISS",
			actorName, ts.Name, action.Name, attackTotal, ts.AC)

		if mutated {
			if pErr := persistEncounterData(ctx, uc, ed, encounterID); pErr != nil {
				l.UsecasesError(pErr, userID, map[string]any{"encounterID": encounterID})
				return nil, fmt.Errorf("persist encounter: %w", pErr)
			}
		}

		return resp, nil
	}

	// Hit — roll damage with resistance
	damageRolls, totalDamage := rollNpcDamage(action.Attack.Damage, isCrit, ts)
	resp.DamageRolls = damageRolls

	// Evaluate Parry reaction for NPC targets (melee attacks only per D&D 5e).
	// D&D 5e Parry: only melee weapon attacks can be parried (not spell attacks).
	isMeleeAttack := action.Attack != nil && (action.Attack.Type == models.AttackRollMeleeWeapon || action.Attack.Type == models.AttackRollMeleeOrRangedWeapon)
	if uc.reactionEval != nil && !target.IsPlayerCharacter && totalDamage > 0 && isMeleeAttack {
		parryResult, pErr := uc.reactionEval.EvaluateParry(ctx, ed, cmd.TargetID, totalDamage)
		if pErr != nil {
			l.UsecasesWarn(pErr, userID, map[string]any{"targetID": cmd.TargetID, "reaction": "parry"})
		}
		if pErr == nil && parryResult != nil {
			totalDamage -= parryResult.DamageReduction
			if totalDamage < 0 {
				totalDamage = 0
			}
			// Update DamageRolls to reflect post-Parry damage.
			adjustDamageRollsForReduction(resp.DamageRolls, parryResult.DamageReduction)
			applyParryReaction(parryResult, ed)
			resp.ReactionSummary = append(resp.ReactionSummary, buildParrySummary(parryResult))
			mutated = true
		}
	}

	resp.Summary = fmt.Sprintf("%s attacks %s with %s: %d to hit vs AC %d — HIT",
		actorName, ts.Name, action.Name, attackTotal, ts.AC)
	if isCrit {
		resp.Summary = fmt.Sprintf("%s attacks %s with %s: %d to hit vs AC %d — CRITICAL HIT!",
			actorName, ts.Name, action.Name, attackTotal, ts.AC)
	}
	if len(damageRolls) > 0 {
		resp.Summary += fmt.Sprintf(", %d damage", totalDamage)
	}

	// Apply damage to target
	if totalDamage > 0 {
		applyDamageToTarget(target, totalDamage)
		resp.StateChanges = append(resp.StateChanges, models.StateChange{
			TargetID:    cmd.TargetID,
			HPDelta:     -totalDamage,
			Description: fmt.Sprintf("%s takes %d damage from %s", ts.Name, totalDamage, action.Name),
		})
		mutated = true
	}

	// Apply action effects (conditions on hit)
	for _, eff := range action.Effects {
		if eff.Condition != nil {
			applyNpcConditionEffect(eff.Condition, action, participant, target, cmd.TargetID, resp)
			mutated = true
		}
	}

	// Persist encounter data
	if mutated {
		if pErr := persistEncounterData(ctx, uc, ed, encounterID); pErr != nil {
			l.UsecasesError(pErr, userID, map[string]any{"encounterID": encounterID})
			return nil, fmt.Errorf("persist encounter: %w", pErr)
		}
	}

	return resp, nil
}

// applyNpcConditionEffect applies a condition from an NPC action to a target.
func applyNpcConditionEffect(
	cond *models.ConditionEffect,
	action *models.StructuredAction,
	source *models.ParticipantFull,
	target *models.ParticipantFull,
	targetID string,
	resp *models.ActionResponse,
) {
	dc := 0
	if action.SavingThrow != nil {
		dc = action.SavingThrow.DC
	}
	if cond.EscapeDC > 0 {
		dc = cond.EscapeDC
	}

	ac := buildActiveCondition(cond, source.InstanceID, action.Name, dc, targetID)
	appendConditionToTarget(target, ac)

	durationStr := cond.Duration
	if durationStr == "" {
		durationStr = "until removed"
	}

	resp.ConditionApplied = append(resp.ConditionApplied, models.ConditionApplied{
		TargetID:  targetID,
		Condition: string(cond.Condition),
		Duration:  durationStr,
		SaveEnds:  cond.SaveEnds,
	})

	resp.Summary += fmt.Sprintf(", %s is %s", target.DisplayName, cond.Condition)
}
