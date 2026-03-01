package triggers

import (
	"fmt"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// EventContext carries information about the game event being processed.
type EventContext struct {
	Event      models.TriggerEvent
	SourceName string // item name, e.g. "Flame Tongue"
	TargetName string // target name if applicable
}

// TriggerInput pairs a TriggerEffect with a cooldown key for state lookup.
type TriggerInput struct {
	Trigger     models.TriggerEffect
	CooldownKey string // "{itemInstanceID}:{triggerIndex}"
}

// Evaluate processes triggers against an event, returning results for each match.
// Pure logic — no DB access. randFloat is injected for test determinism.
//
// A malformed trigger (bad params, unknown effect type) produces a result with
// Skipped=true and SkipReason="error" instead of aborting the entire evaluation.
// This ensures valid triggers still fire even if a sibling has a data error.
//
// Note: on_critical does NOT imply on_hit. The caller must dispatch both events
// separately when a critical hit occurs (e.g., Evaluate with on_hit, then on_critical).
func Evaluate(
	triggers []TriggerInput,
	ctx EventContext,
	cooldowns models.CooldownState,
	randFloat func() float32,
) []models.TriggerResult {
	var results []models.TriggerResult

	for _, ti := range triggers {
		if ti.Trigger.Trigger != ctx.Event {
			continue
		}

		// Cooldown gate
		if cooldowns != nil && ti.CooldownKey != "" && cooldowns[ti.CooldownKey] {
			results = append(results, models.TriggerResult{
				Event:       ctx.Event,
				EffectType:  ti.Trigger.Effect.Type,
				Skipped:     true,
				SkipReason:  "cooldown",
				Description: fmt.Sprintf("%s: on cooldown", ctx.SourceName),
			})
			continue
		}

		// Chance gate: 0 is treated as 1.0 (always fires), values outside
		// [0, 1] are clamped to prevent silent misconfiguration.
		chance := clampChance(ti.Trigger.Chance)
		if chance > 0 && chance < 1.0 && randFloat() >= chance {
			results = append(results, models.TriggerResult{
				Event:       ctx.Event,
				EffectType:  ti.Trigger.Effect.Type,
				Skipped:     true,
				SkipReason:  "chance",
				Description: fmt.Sprintf("%s: chance check failed", ctx.SourceName),
			})
			continue
		}

		// Execute effect — errors are captured, not propagated
		result, err := executeEffect(ti.Trigger.Effect, ctx.SourceName)
		if err != nil {
			results = append(results, models.TriggerResult{
				Event:       ctx.Event,
				EffectType:  ti.Trigger.Effect.Type,
				Skipped:     true,
				SkipReason:  "error",
				Description: fmt.Sprintf("%s: %v", ctx.SourceName, err),
			})
			continue
		}
		result.Event = ctx.Event
		results = append(results, result)
	}

	return results
}

// clampChance normalizes a chance value to [0, 1].
func clampChance(c float32) float32 {
	if c < 0 {
		return 0
	}
	if c > 1 {
		return 1
	}
	return c
}

func executeEffect(effect models.Effect, source string) (models.TriggerResult, error) {
	switch effect.Type {
	case models.EffectDealDamage:
		return executeDealDamage(effect.Params, source)
	case models.EffectHeal:
		return executeHeal(effect.Params, source)
	case models.EffectApplyCondition:
		return executeApplyCondition(effect.Params, source)
	case models.EffectRemoveCondition:
		return executeRemoveCondition(effect.Params, source)
	case models.EffectGrantTempHP:
		return executeGrantTempHP(effect.Params, source)
	default:
		return models.TriggerResult{}, fmt.Errorf("unknown effect type: %q", effect.Type)
	}
}
