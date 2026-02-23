package combatai

import (
	"fmt"
	"math/rand"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// SelectBonusAction chooses the best bonus action for the active NPC.
// Filters StructuredActions by ActionCategoryBonus, applies same EV logic
// as single actions (recharge priority, AoE multiplier, resource check).
// Returns nil if no valid bonus action exists or BonusActionUsed is true.
func SelectBonusAction(input *TurnInput, role CreatureRole, rng *rand.Rand) *ActionDecision {
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	if input.ActiveNPC.RuntimeState.Resources.BonusActionUsed {
		return nil
	}

	resources := &input.ActiveNPC.RuntimeState.Resources
	creature := &input.CreatureTemplate
	actions := actionsByCategory(creature.StructuredActions, models.ActionCategoryBonus)
	if len(actions) == 0 {
		return nil
	}

	var candidates []*ActionDecision

	// 1. Recharge-ready bonus action — use immediately.
	if recharge := findRechargeReady(input, actions, resources); recharge != nil {
		return recharge
	}

	// 2. Single bonus actions — evaluate by EV.
	baseline := bestBonusBaselineEV(input, actions)

	for i := range actions {
		a := &actions[i]
		if !isActionAvailable(a, resources) {
			continue
		}
		if a.Recharge != nil {
			continue // already handled above
		}
		targetIDs := SelectTarget(input, a)
		if len(targetIDs) == 0 {
			continue
		}
		targetStats := input.CombatantStats[targetIDs[0]]
		ev := ComputeExpectedDamage(*a, targetStats)

		ev, targetIDs = applyAoEMultiplier(input, a, ev, targetIDs)

		if a.Uses != nil && !ShouldUseAbility(ev, baseline, input.Intelligence) {
			continue
		}

		candidates = append(candidates, &ActionDecision{
			ActionType:     actionTypeFromAction(a),
			ActionID:       a.ID,
			TargetIDs:      targetIDs,
			ActionName:     a.Name,
			ExpectedDamage: ev,
			Reasoning:      fmt.Sprintf("bonus action %q: EV=%.1f", a.Name, ev),
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	if rng.Float64() < input.Intelligence {
		return bestByEV(candidates)
	}
	return candidates[rng.Intn(len(candidates))]
}

// bestBonusBaselineEV computes the best EV from non-resource bonus actions.
func bestBonusBaselineEV(input *TurnInput, bonusActions []models.StructuredAction) float64 {
	best := 0.0
	for i := range bonusActions {
		a := &bonusActions[i]
		if a.Uses != nil || a.Recharge != nil {
			continue
		}
		targetIDs := SelectTarget(input, a)
		if len(targetIDs) == 0 {
			continue
		}
		ev := ComputeExpectedDamage(*a, input.CombatantStats[targetIDs[0]])
		if ev > best {
			best = ev
		}
	}
	return best
}

// SelectLegendaryAction chooses the best legendary action to spend.
// Filters by ActionCategoryLegendary, checks LegendaryActions >= LegendaryCost.
// Returns nil if no legendary actions remain or no valid action exists.
func SelectLegendaryAction(input *TurnInput, rng *rand.Rand) *ActionDecision {
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	remaining := input.ActiveNPC.RuntimeState.Resources.LegendaryActions
	if remaining <= 0 {
		return nil
	}

	creature := &input.CreatureTemplate
	actions := actionsByCategory(creature.StructuredActions, models.ActionCategoryLegendary)
	if len(actions) == 0 {
		return nil
	}

	resources := &input.ActiveNPC.RuntimeState.Resources
	var candidates []*ActionDecision

	for i := range actions {
		a := &actions[i]

		cost := a.LegendaryCost
		if cost <= 0 {
			cost = 1 // default legendary action cost
		}
		if cost > remaining {
			continue
		}

		if !isActionAvailable(a, resources) {
			continue
		}

		targetIDs := SelectTarget(input, a)
		if len(targetIDs) == 0 {
			continue
		}
		targetStats := input.CombatantStats[targetIDs[0]]
		ev := ComputeExpectedDamage(*a, targetStats)

		ev, targetIDs = applyAoEMultiplier(input, a, ev, targetIDs)

		if ev <= 0 {
			continue
		}

		candidates = append(candidates, &ActionDecision{
			ActionType:     actionTypeFromAction(a),
			ActionID:       a.ID,
			TargetIDs:      targetIDs,
			LegendaryCost:  cost,
			ActionName:     a.Name,
			ExpectedDamage: ev,
			Reasoning:      fmt.Sprintf("legendary action %q (cost %d): EV=%.1f", a.Name, cost, ev),
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	if rng.Float64() < input.Intelligence {
		return bestByEV(candidates)
	}
	return candidates[rng.Intn(len(candidates))]
}
