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
func Evaluate(
	triggers []TriggerInput,
	ctx EventContext,
	cooldowns models.CooldownState,
	randFloat func() float32,
) ([]models.TriggerResult, error) {
	var results []models.TriggerResult

	for _, ti := range triggers {
		if ti.Trigger.Trigger != ctx.Event {
			continue
		}

		// Cooldown gate
		if cooldowns != nil && ti.CooldownKey != "" && cooldowns[ti.CooldownKey] {
			results = append(results, models.TriggerResult{
				TriggerEvent: ctx.Event,
				EffectType:   ti.Trigger.Effect.Type,
				Skipped:      true,
				SkipReason:   "cooldown",
				Description:  fmt.Sprintf("%s: on cooldown", ctx.SourceName),
			})
			continue
		}

		// Chance gate: 0 is treated as 1.0 (always fires)
		chance := ti.Trigger.Chance
		if chance > 0 && chance < 1.0 && randFloat() >= chance {
			results = append(results, models.TriggerResult{
				TriggerEvent: ctx.Event,
				EffectType:   ti.Trigger.Effect.Type,
				Skipped:      true,
				SkipReason:   "chance",
				Description:  fmt.Sprintf("%s: chance check failed", ctx.SourceName),
			})
			continue
		}

		// Execute effect
		result, err := executeEffect(ti.Trigger.Effect, ctx.SourceName)
		if err != nil {
			return nil, fmt.Errorf("trigger %q effect %q: %w", ctx.SourceName, ti.Trigger.Effect.Type, err)
		}
		result.TriggerEvent = ctx.Event
		results = append(results, result)
	}

	return results, nil
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
