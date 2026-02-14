package usecases

import (
	"fmt"
	"math/rand/v2"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/triggers"
)

// applyTriggerResults evaluates triggers for the given event(s) and applies
// side effects (damage, heal, temp HP) to participants. Returns results and
// state changes. Reusable by any resolver (weapon, spell, feature).
//
// When isCrit is true, both on_hit and on_critical triggers are dispatched.
// Cooldown state is nil (T42 will add persistence).
func applyTriggerResults(
	triggerDefs []models.TriggerEffect,
	sourceName string,
	event models.TriggerEvent,
	isCrit bool,
	target *models.ParticipantFull,
	attacker *models.ParticipantFull,
) ([]models.TriggerResult, []models.StateChange) {
	if len(triggerDefs) == 0 {
		return nil, nil
	}

	// Build TriggerInputs from definitions
	inputs := make([]triggers.TriggerInput, len(triggerDefs))
	for i, td := range triggerDefs {
		inputs[i] = triggers.TriggerInput{
			Trigger:     td,
			CooldownKey: sourceName + ":" + strconv.Itoa(i),
		}
	}

	ctx := triggers.EventContext{
		Event:      event,
		SourceName: sourceName,
	}
	if target != nil {
		ctx.TargetName = participantName(target)
	}

	randFloat := func() float32 { return rand.Float32() }

	// Evaluate for the primary event (on_hit)
	results := triggers.Evaluate(inputs, ctx, nil, randFloat)

	// On critical hit, also dispatch on_critical triggers
	if isCrit && event == models.ItemTriggerOnHit {
		critCtx := ctx
		critCtx.Event = models.ItemTriggerOnCritical
		critResults := triggers.Evaluate(inputs, critCtx, nil, randFloat)
		results = append(results, critResults...)
	}

	// Apply side effects
	var stateChanges []models.StateChange

	for _, r := range results {
		if r.Skipped {
			continue
		}

		switch r.EffectType {
		case models.EffectDealDamage:
			if r.DamageResult != nil && target != nil {
				applyDamageToTarget(target, r.DamageResult.Total)
				stateChanges = append(stateChanges, models.StateChange{
					TargetID:    target.InstanceID,
					HPDelta:     -r.DamageResult.Total,
					Description: r.Description,
				})
			}

		case models.EffectHeal:
			if r.HealResult != nil && attacker != nil {
				applyHealToParticipant(attacker, r.HealResult.Total)
				stateChanges = append(stateChanges, models.StateChange{
					TargetID:    attacker.InstanceID,
					HPDelta:     r.HealResult.Total,
					Description: r.Description,
				})
			}

		case models.EffectGrantTempHP:
			if r.TempHPResult != nil && attacker != nil {
				applied := applyTempHP(attacker, r.TempHPResult.Amount)
				if applied {
					stateChanges = append(stateChanges, models.StateChange{
						TargetID:    attacker.InstanceID,
						Description: r.Description,
					})
				}
			}

		case models.EffectApplyCondition, models.EffectRemoveCondition:
			// No HP mutation — description only (future: runtime condition tracking)
		}
	}

	return results, stateChanges
}

// applyHealToParticipant adds HP to a participant, capped at max for creatures.
// For PCs, max HP is derived externally — no cap applied here (DM adjusts if needed).
func applyHealToParticipant(p *models.ParticipantFull, amount int) {
	if p.CharacterRuntime != nil {
		p.CharacterRuntime.CurrentHP += amount
		return
	}

	p.RuntimeState.CurrentHP += amount
	if p.RuntimeState.CurrentHP > p.RuntimeState.MaxHP {
		p.RuntimeState.CurrentHP = p.RuntimeState.MaxHP
	}
}

// applyTempHP sets temp HP on a participant if the new amount exceeds the current.
// D&D 5e: temp HP doesn't stack — take the higher value.
func applyTempHP(p *models.ParticipantFull, amount int) bool {
	if p.CharacterRuntime != nil {
		if amount > p.CharacterRuntime.TemporaryHP {
			p.CharacterRuntime.TemporaryHP = amount
			return true
		}
		return false
	}

	if amount > p.RuntimeState.TempHP {
		p.RuntimeState.TempHP = amount
		return true
	}
	return false
}

// participantName returns a display name for a participant.
func participantName(p *models.ParticipantFull) string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	if p.CharacterRuntime != nil {
		return fmt.Sprintf("character:%s", p.CharacterRuntime.CharacterID)
	}
	return p.InstanceID
}
