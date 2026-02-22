package combatai

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// RuleBasedAI implements CombatAI using deterministic rule-based logic.
// DecideTurn is a pure function — it does not mutate TurnInput or access external state.
type RuleBasedAI struct {
	rng *rand.Rand
}

// NewRuleBasedAI creates a CombatAI with a random seed.
func NewRuleBasedAI() CombatAI {
	return &RuleBasedAI{rng: rand.New(rand.NewSource(rand.Int63()))}
}

// NewRuleBasedAIWithSeed creates a RuleBasedAI with a fixed seed for deterministic testing.
func NewRuleBasedAIWithSeed(seed int64) *RuleBasedAI {
	return &RuleBasedAI{rng: rand.New(rand.NewSource(seed))}
}

// DecideTurn orchestrates a single NPC turn decision.
//
// Returns (nil, nil) if the NPC is dead.
// Returns (TurnDecision with Action=nil) if incapacitated.
func (ai *RuleBasedAI) DecideTurn(input *TurnInput) (*TurnDecision, error) {
	// 1. Dead → no decision.
	if !IsAlive(&input.ActiveNPC) {
		return nil, nil
	}

	// 2. Incapacitated → skip turn.
	if isIncapacitated(&input.ActiveNPC) {
		return &TurnDecision{Reasoning: "Incapacitated — skip"}, nil
	}

	// 3. Classify combat role.
	role := ClassifyRole(input.CreatureTemplate)

	// 4. Select best action (internally calls SelectTarget, EvaluateMultiattack, etc.).
	action := SelectAction(input, role, ai.rng)

	// 5. Dodge fallback: no good action AND low HP.
	if action == nil {
		npcStats := input.CombatantStats[input.ActiveNPC.InstanceID]
		if hpPercent(&input.ActiveNPC, npcStats) < 0.25 {
			action = dodgeDecision()
		}
	}

	// 6. Build turn decision.
	return &TurnDecision{
		Action:    action,
		Reasoning: buildReasoning(role, action),
	}, nil
}

// isIncapacitated checks if the NPC has a condition that prevents taking actions.
// In D&D 5e, stunned/paralyzed/petrified/unconscious all imply incapacitated.
func isIncapacitated(p *models.ParticipantFull) bool {
	for _, c := range p.RuntimeState.Conditions {
		switch c.Condition {
		case models.ConditionIncapacitated, models.ConditionStunned,
			models.ConditionParalyzed, models.ConditionPetrified,
			models.ConditionUnconscious:
			return true
		}
	}
	return false
}

// buildReasoning creates a human-readable explanation for the DM log.
func buildReasoning(role CreatureRole, action *ActionDecision) string {
	if action == nil {
		return fmt.Sprintf("%s role: no action available", capitalize(string(role)))
	}

	parts := []string{fmt.Sprintf("%s role:", capitalize(string(role)))}

	if action.MultiattackSteps != nil {
		parts = append(parts, fmt.Sprintf("multiattack %s", action.ActionName))
	} else {
		parts = append(parts, action.ActionName)
	}

	if len(action.TargetIDs) > 0 {
		parts = append(parts, fmt.Sprintf("on %s", action.TargetIDs[0]))
	}

	if action.ExpectedDamage > 0 {
		parts = append(parts, fmt.Sprintf("(EV=%.1f)", action.ExpectedDamage))
	}

	return strings.Join(parts, " ")
}

// capitalize returns the string with its first letter uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
