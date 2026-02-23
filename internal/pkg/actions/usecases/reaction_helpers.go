package usecases

import (
	"fmt"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
)

// applyShieldReaction mutates encounter data: adds +5 AC StatModifier,
// marks reaction used, and deducts the spell slot on the reactor.
func applyShieldReaction(
	result *actionsinterfaces.ShieldReactionResult,
	ed *EncounterData,
) {
	reactor, _, err := ed.FindParticipantByInstanceID(result.ReactorID)
	if err != nil {
		return
	}

	// Add Shield StatModifier (+5 AC until start of reactor's next turn).
	shieldMod := models.StatModifier{
		ID:       fmt.Sprintf("shield-%s", result.ReactorID),
		Name:     "Shield",
		SourceID: result.ReactorID,
		Modifiers: []models.ModifierEffect{
			{
				Target:    models.ModTargetAC,
				Operation: models.ModOpAdd,
				Value:     5,
			},
		},
		Duration: models.DurationUntilTurn,
	}
	reactor.RuntimeState.StatModifiers = append(reactor.RuntimeState.StatModifiers, shieldMod)

	// Mark reaction used.
	reactor.RuntimeState.Resources.ReactionUsed = true

	// Deduct spell slot (if slotLevel > 0).
	if result.SlotLevel > 0 {
		if reactor.RuntimeState.Resources.SpellSlots == nil {
			reactor.RuntimeState.Resources.SpellSlots = make(map[int]int)
		}
		if reactor.RuntimeState.Resources.SpellSlots[result.SlotLevel] > 0 {
			reactor.RuntimeState.Resources.SpellSlots[result.SlotLevel]--
		}
	}

	// Deduct innate per-day use (if InnateKey is set).
	if result.InnateKey != "" {
		deductInnateUse(&reactor.RuntimeState.Resources, result.InnateKey)
	}
}

// applyCounterspellReaction marks reaction used and deducts the spell slot
// on the reactor NPC. Called regardless of whether the counterspell succeeded
// (the slot is always consumed in D&D 5e).
func applyCounterspellReaction(
	result *actionsinterfaces.CounterspellReactionResult,
	ed *EncounterData,
) {
	reactor, _, err := ed.FindParticipantByInstanceID(result.ReactorID)
	if err != nil {
		return
	}

	// Mark reaction used.
	reactor.RuntimeState.Resources.ReactionUsed = true

	// Deduct spell slot.
	if result.SlotLevel > 0 {
		if reactor.RuntimeState.Resources.SpellSlots == nil {
			reactor.RuntimeState.Resources.SpellSlots = make(map[int]int)
		}
		if reactor.RuntimeState.Resources.SpellSlots[result.SlotLevel] > 0 {
			reactor.RuntimeState.Resources.SpellSlots[result.SlotLevel]--
		}
	}

	// Deduct innate per-day use (if InnateKey is set).
	if result.InnateKey != "" {
		deductInnateUse(&reactor.RuntimeState.Resources, result.InnateKey)
	}
}

// applyParryReaction marks reaction used on the target NPC.
func applyParryReaction(
	result *actionsinterfaces.ParryReactionResult,
	ed *EncounterData,
) {
	reactor, _, err := ed.FindParticipantByInstanceID(result.ReactorID)
	if err != nil {
		return
	}

	reactor.RuntimeState.Resources.ReactionUsed = true
}

// buildShieldSummary creates a ReactionSummary for a Shield reaction.
func buildShieldSummary(result *actionsinterfaces.ShieldReactionResult) models.ReactionSummary {
	return models.ReactionSummary{
		ReactorID:   result.ReactorID,
		ReactorName: result.ReactorName,
		Type:        "shield",
		Description: fmt.Sprintf(
			"%s casts Shield as a reaction! AC increased by %d (now %d)",
			result.ReactorName, result.ACBonus, result.NewEffectiveAC,
		),
	}
}

// buildCounterspellSummary creates a ReactionSummary for a Counterspell reaction.
func buildCounterspellSummary(result *actionsinterfaces.CounterspellReactionResult) models.ReactionSummary {
	desc := fmt.Sprintf("%s casts Counterspell", result.ReactorName)
	if result.AbilityCheck != nil {
		if result.Success {
			desc += fmt.Sprintf(" (ability check %d vs DC %d — success!)", *result.AbilityCheck, result.CheckDC)
		} else {
			desc += fmt.Sprintf(" (ability check %d vs DC %d — failed)", *result.AbilityCheck, result.CheckDC)
		}
	} else if result.Success {
		desc += " — spell countered!"
	}
	return models.ReactionSummary{
		ReactorID:   result.ReactorID,
		ReactorName: result.ReactorName,
		Type:        "counterspell",
		Description: desc,
	}
}

// deductInnateUse decrements an innate per-day spell use in AbilityUses.
// The evaluator's ensureInnateInitialized pre-populates the key with the N/day count,
// so the key should always exist when this is called. The fallback (key missing → set 0)
// is kept as a safety net but should not be hit in normal flow.
func deductInnateUse(resources *models.ResourceState, key string) {
	if resources.AbilityUses == nil {
		resources.AbilityUses = make(map[string]int)
	}
	if remaining, exists := resources.AbilityUses[key]; exists {
		if remaining > 0 {
			resources.AbilityUses[key] = remaining - 1
		}
	} else {
		// Safety fallback: key not tracked (ensureInnateInitialized was not called).
		resources.AbilityUses[key] = 0
	}
}

// buildParrySummary creates a ReactionSummary for a Parry reaction.
func buildParrySummary(result *actionsinterfaces.ParryReactionResult) models.ReactionSummary {
	return models.ReactionSummary{
		ReactorID:   result.ReactorID,
		ReactorName: result.ReactorName,
		Type:        "parry",
		Description: fmt.Sprintf(
			"%s uses Parry! Damage reduced by %d",
			result.ReactorName, result.DamageReduction,
		),
	}
}

// adjustDamageRollsForReduction subtracts a flat damage reduction (e.g. Parry)
// from the FinalDamage fields of DamageRolls so the API response is consistent
// with the actual damage applied. Reduction is distributed across components,
// subtracting from each in order until exhausted.
func adjustDamageRollsForReduction(rolls []models.ActionRollResult, reduction int) {
	remaining := reduction
	for i := range rolls {
		if remaining <= 0 {
			break
		}
		if rolls[i].FinalDamage == nil {
			continue
		}
		fd := *rolls[i].FinalDamage
		if fd <= 0 {
			continue
		}
		sub := remaining
		if sub > fd {
			sub = fd
		}
		fd -= sub
		rolls[i].FinalDamage = &fd
		remaining -= sub
	}
}
