package combatai

import (
	"fmt"
	"math/rand"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// SelectAction chooses the best action for the active NPC.
// Uses expected value scoring, resource checks, and an intelligence gate.
// The rng parameter allows deterministic testing; pass nil for default.
// Returns nil if no valid action exists (caller should fall back to Dodge).
func SelectAction(input *TurnInput, role CreatureRole, rng *rand.Rand) *ActionDecision {
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	resources := &input.ActiveNPC.RuntimeState.Resources
	creature := &input.CreatureTemplate
	actions := actionsByCategory(creature.StructuredActions, models.ActionCategoryAction)

	var candidates []*ActionDecision

	// 1. Recharge-ready ability — always use immediately.
	// Per D&D design, these are a creature's most powerful abilities (dragon breath, etc.).
	if recharge := findRechargeReady(input, actions, resources); recharge != nil {
		return recharge
	}

	// 2. Multiattack groups.
	if ma := EvaluateMultiattack(input, creature.Multiattacks); ma != nil {
		candidates = append(candidates, ma)
	}

	// 3. Single actions.
	// Compute baseline EV for limited-use ability cost-benefit check.
	baseline := bestBaselineEV(input)

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

		// Limited-use ability economy: only spend if EV justifies it.
		if a.Uses != nil && !ShouldUseAbility(ev, baseline, input.Intelligence) {
			continue
		}

		candidates = append(candidates, &ActionDecision{
			ActionType:     actionTypeFromAction(a),
			ActionID:       a.ID,
			TargetIDs:      targetIDs,
			ActionName:     a.Name,
			ExpectedDamage: ev,
			Reasoning:      fmt.Sprintf("single action %q: EV=%.1f", a.Name, ev),
		})
	}

	// 4. Spells.
	if creature.Spellcasting != nil {
		spellCandidates := evaluateSpells(input, creature.Spellcasting, resources)
		candidates = append(candidates, spellCandidates...)
	}

	if len(candidates) == 0 {
		return nil
	}

	// Intelligence gate: optimal pick vs random.
	if rng.Float64() < input.Intelligence {
		return bestByEV(candidates)
	}
	return candidates[rng.Intn(len(candidates))]
}

// findRechargeReady returns an ActionDecision for a recharge-ready ability,
// or nil if none is ready.
func findRechargeReady(input *TurnInput, actions []models.StructuredAction, resources *models.ResourceState) *ActionDecision {
	for i := range actions {
		a := &actions[i]
		if a.Recharge == nil {
			continue
		}
		if !resources.RechargeReady[a.ID] {
			continue
		}
		targetIDs := SelectTarget(input, a)
		if len(targetIDs) == 0 {
			continue
		}
		targetStats := input.CombatantStats[targetIDs[0]]
		ev := ComputeExpectedDamage(*a, targetStats)

		return &ActionDecision{
			ActionType:     actionTypeFromAction(a),
			ActionID:       a.ID,
			TargetIDs:      targetIDs,
			ActionName:     a.Name,
			ExpectedDamage: ev,
			Reasoning:      fmt.Sprintf("recharge-ready %q: EV=%.1f", a.Name, ev),
		}
	}
	return nil
}

// isActionAvailable checks resource constraints for a single action.
func isActionAvailable(action *models.StructuredAction, resources *models.ResourceState) bool {
	if action.Uses != nil && resources.AbilityUses[action.ID] <= 0 {
		return false
	}
	if action.Recharge != nil && !resources.RechargeReady[action.ID] {
		return false
	}
	return true
}

// evaluateSpells scores all available spells and returns ActionDecisions.
func evaluateSpells(input *TurnInput, sc *models.Spellcasting, resources *models.ResourceState) []*ActionDecision {
	var results []*ActionDecision

	// Evaluate spells organized by level.
	for level, spells := range sc.SpellsByLevel {
		for _, spell := range spells {
			decision := evaluateSingleSpell(input, spell, level, sc, resources)
			if decision != nil {
				results = append(results, decision)
			}
		}
	}

	// Evaluate flat spell list.
	for _, spell := range sc.Spells {
		decision := evaluateSingleSpell(input, spell, spell.Level, sc, resources)
		if decision != nil {
			results = append(results, decision)
		}
	}

	return results
}

// evaluateSingleSpell scores one spell and returns an ActionDecision or nil.
func evaluateSingleSpell(input *TurnInput, spell models.SpellKnown, level int, sc *models.Spellcasting, resources *models.ResourceState) *ActionDecision {
	// Cantrips are always available; leveled spells need slots.
	if level > 0 && resources.SpellSlots[level] <= 0 {
		return nil
	}

	// Spell slot economy: intelligence-gated round-based spending.
	if level > 0 {
		hpPct := npcHPPercent(input)
		maxLevel := maxAvailableSlotLevel(resources.SpellSlots)
		if !ShouldSpendSpellSlot(level, input.CurrentRound, hpPct,
			input.Intelligence, maxLevel) {
			return nil
		}
	}

	// Need a target to evaluate against.
	targetIDs := SelectTarget(input, nil)
	if len(targetIDs) == 0 {
		return nil
	}
	targetStats := input.CombatantStats[targetIDs[0]]

	ev := EstimateSpellDamage(spell, *sc, targetStats)
	if ev <= 0 {
		return nil // non-damage spell — skip in Phase 1
	}

	return &ActionDecision{
		ActionType:     models.ActionSpellCast,
		ActionID:       spell.SpellID,
		TargetIDs:      targetIDs,
		SlotLevel:      level,
		ActionName:     spell.Name,
		ExpectedDamage: ev,
		Reasoning:      fmt.Sprintf("spell %q (level %d): EV=%.1f", spell.Name, level, ev),
	}
}

// bestByEV returns the candidate with the highest ExpectedDamage.
func bestByEV(candidates []*ActionDecision) *ActionDecision {
	var best *ActionDecision
	for _, c := range candidates {
		if best == nil || c.ExpectedDamage > best.ExpectedDamage {
			best = c
		}
	}
	return best
}
