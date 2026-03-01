package usecases

import (
	"fmt"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/triggers"
)

// triggerOpts carries contextual data for trigger evaluation.
type triggerOpts struct {
	SourceID    string                  // stable ID for cooldown keys (e.g. weapon.ID)
	SourceName  string                  // display name for descriptions
	TargetStats *TargetStats            // resistance/immunity/vulnerability lookup (nil = no resistance check)
	RandFloat   func() float32          // injected RNG for testability
	Owner       *models.ParticipantFull // for charge tracking (nil = no cooldowns)
}

// applyTriggerResults evaluates triggers for the given event(s) and applies
// side effects (damage, heal, temp HP) to participants. Returns results and
// state changes. Reusable by any resolver (weapon, spell, feature).
//
// When isCrit is true, both on_hit and on_critical triggers are dispatched.
// Cooldown state is built from opts.Owner.TriggerCharges when Owner is set.
func applyTriggerResults(
	triggerDefs []models.TriggerEffect,
	opts triggerOpts,
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
			CooldownKey: opts.SourceID + ":" + strconv.Itoa(i),
		}
	}

	// Build cooldown state from owner's charges
	var cooldowns models.CooldownState
	if opts.Owner != nil {
		cooldowns = triggers.BuildCooldownState(triggerDefs, opts.SourceID, opts.Owner.TriggerCharges)
	}

	ctx := triggers.EventContext{
		Event:      event,
		SourceName: opts.SourceName,
	}
	if target != nil {
		ctx.TargetName = participantName(target)
	}

	// Evaluate for the primary event (on_hit)
	hitResults := triggers.Evaluate(inputs, ctx, cooldowns, opts.RandFloat)

	// On critical hit, also dispatch on_critical triggers
	var critResults []models.TriggerResult
	if isCrit && event == models.ItemTriggerOnHit {
		critCtx := ctx
		critCtx.Event = models.ItemTriggerOnCritical
		critResults = triggers.Evaluate(inputs, critCtx, cooldowns, opts.RandFloat)
	}

	// Consume charges for non-skipped triggers (mutates Owner.TriggerCharges in-place)
	if opts.Owner != nil {
		consumeFiredCharges(opts.Owner, triggerDefs, inputs, hitResults, event)
		if len(critResults) > 0 {
			consumeFiredCharges(opts.Owner, triggerDefs, inputs, critResults, models.ItemTriggerOnCritical)
		}
	}

	results := hitResults
	if len(critResults) > 0 {
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
				finalDmg := r.DamageResult.Total
				if opts.TargetStats != nil {
					finalDmg, _ = applyResistance(finalDmg, r.DamageResult.DamageType, opts.TargetStats)
				}
				applyDamageToTarget(target, finalDmg)
				stateChanges = append(stateChanges, models.StateChange{
					TargetID:    target.InstanceID,
					HPDelta:     -finalDmg,
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
// TODO: pass max HP for PCs to cap healing properly.
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

// consumeFiredCharges consumes cooldown charges for non-skipped trigger results.
// Results must correspond to triggers matching the given event (same order as Evaluate).
func consumeFiredCharges(
	owner *models.ParticipantFull,
	defs []models.TriggerEffect,
	inputs []triggers.TriggerInput,
	results []models.TriggerResult,
	event models.TriggerEvent,
) {
	rIdx := 0
	for i := range inputs {
		if rIdx >= len(results) {
			break
		}
		if inputs[i].Trigger.Trigger != event {
			continue
		}
		if !results[rIdx].Skipped {
			owner.TriggerCharges = triggers.ConsumeCooldown(
				owner.TriggerCharges, inputs[i].CooldownKey, defs[i].Cooldown,
			)
		}
		rIdx++
	}
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
