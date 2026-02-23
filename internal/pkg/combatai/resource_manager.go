package combatai

import "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"

// ShouldSpendSpellSlot decides if spending a spell slot of the given level is
// tactically justified given current round, HP state, and intelligence.
//
// Intelligence tiers (from design plan E.8):
//   - < 0.55: no management, always spend
//   - 0.55–0.75: basic — block top-level slot on round 1
//   - >= 0.75: full round-based economy
//
// HP emergency (< 30%): always allow regardless of round/intelligence.
// Cantrips (level 0): always allowed.
func ShouldSpendSpellSlot(level int, round int, hpPercent float64,
	intelligence float64, maxAvailableLevel int) bool {
	// Cantrips are always free.
	if level <= 0 {
		return true
	}

	// HP emergency: spend everything.
	if hpPercent < 0.30 {
		return true
	}

	// Low intelligence: no resource management.
	if intelligence < 0.55 {
		return true
	}

	isTopLevel := level >= maxAvailableLevel

	// Medium intelligence (0.55–0.75): don't spend top-level slot on round 1.
	if intelligence < 0.75 {
		if isTopLevel && round <= 1 {
			return false
		}
		return true
	}

	// High intelligence (>= 0.75): full round-based economy.
	switch {
	case round <= 2:
		// Opening rounds: all levels allowed.
		return true
	case round <= 5:
		// Mid-combat: medium levels only.
		midCutoff := (maxAvailableLevel + 1) / 2
		return level <= midCutoff
	default:
		// Late combat: conserve — cantrips + level 1-2 only.
		return level <= 2
	}
}

// ShouldUseAbility decides if a limited-use ability (uses/day) should be spent,
// comparing its expected value against the best baseline (non-resource) weapon EV.
//
// Design plan E.5: "Использовать если expected_value >= 150% лучшей обычной атаки"
//
// Intelligence < 0.55: always use if available (no cost-benefit analysis).
// If baseline is 0 (no free weapons), always allow the ability.
func ShouldUseAbility(abilityEV float64, baselineEV float64, intelligence float64) bool {
	// Low intelligence: spend immediately.
	if intelligence < 0.55 {
		return true
	}

	// No baseline weapons: always allow (nothing to compare against).
	if baselineEV <= 0 {
		return true
	}

	// Smart NPC: only if EV >= 150% of baseline.
	return abilityEV >= 1.5*baselineEV
}

// maxAvailableSlotLevel returns the highest spell slot level with remaining slots.
// Returns 0 if no slots are available.
func maxAvailableSlotLevel(slots map[int]int) int {
	best := 0
	for level, count := range slots {
		if count > 0 && level > best {
			best = level
		}
	}
	return best
}

// bestBaselineEV computes the best expected value from non-resource actions
// (no Uses, no Recharge, no spells). Used as reference for limited-use decisions.
func bestBaselineEV(input *TurnInput) float64 {
	actions := actionsByCategory(input.CreatureTemplate.StructuredActions, models.ActionCategoryAction)
	best := 0.0

	for i := range actions {
		a := &actions[i]
		// Skip resource-gated actions.
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

// npcHPPercent returns the active NPC's current HP as a ratio (0.0–1.0).
// Returns 1.0 if MaxHP is 0 (unknown), to avoid false emergency triggers.
func npcHPPercent(input *TurnInput) float64 {
	hp := GetCurrentHP(&input.ActiveNPC)
	maxHP := input.ActiveNPC.RuntimeState.MaxHP
	if maxHP <= 0 {
		return 1.0
	}
	return float64(hp) / float64(maxHP)
}
