package combatai

import (
	"fmt"
	"math"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// EvaluateMultiattack picks the best multiattack group and returns an
// ActionDecision with MultiattackSteps filled. Returns nil if no groups
// are provided or no valid group can be assembled.
func EvaluateMultiattack(input *TurnInput, groups []models.MultiattackGroup) *ActionDecision {
	if len(groups) == 0 {
		return nil
	}

	actionLookup := buildActionLookup(input.CreatureTemplate.StructuredActions)

	var bestDecision *ActionDecision
	bestEV := math.Inf(-1)

	for i := range groups {
		decision := evaluateGroup(input, &groups[i], actionLookup)
		if decision == nil {
			continue
		}
		if decision.ExpectedDamage > bestEV {
			bestEV = decision.ExpectedDamage
			bestDecision = decision
		}
	}

	return bestDecision
}

// evaluateGroup builds an ActionDecision for a single multiattack group.
// Returns nil if the group has no resolvable actions.
func evaluateGroup(input *TurnInput, group *models.MultiattackGroup, lookup map[string]models.StructuredAction) *ActionDecision {
	var steps []MultiattackStep
	var totalEV float64

	for _, entry := range group.Actions {
		action, ok := lookup[entry.ActionID]
		if !ok {
			continue // unknown action ID — skip entry
		}

		targetIDs := SelectTarget(input, &action)
		if len(targetIDs) == 0 {
			continue
		}

		targetStats := input.CombatantStats[targetIDs[0]]
		ev := ComputeExpectedDamage(action, targetStats)

		for range entry.Count {
			steps = append(steps, MultiattackStep{
				ActionType: actionTypeFromAction(&action),
				ActionID:   entry.ActionID,
				TargetIDs:  targetIDs,
			})
			totalEV += ev
		}
	}

	if len(steps) == 0 {
		return nil
	}

	return &ActionDecision{
		MultiattackGroupID: group.ID,
		MultiattackSteps:   steps,
		ActionName:         group.Name,
		ExpectedDamage:     totalEV,
		Reasoning:          fmt.Sprintf("multiattack %q: %d steps, EV=%.1f", group.Name, len(steps), totalEV),
	}
}

// buildActionLookup indexes StructuredActions by their ID for O(1) lookups.
func buildActionLookup(actions []models.StructuredAction) map[string]models.StructuredAction {
	lookup := make(map[string]models.StructuredAction, len(actions))
	for _, a := range actions {
		lookup[a.ID] = a
	}
	return lookup
}

// actionTypeFromAction returns the ActionType for a StructuredAction.
func actionTypeFromAction(action *models.StructuredAction) models.ActionType {
	if action.Attack != nil {
		return models.ActionWeaponAttack
	}
	if action.SavingThrow != nil {
		return models.ActionUseFeature
	}
	return models.ActionUseFeature
}
