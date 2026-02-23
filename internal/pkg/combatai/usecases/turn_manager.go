package usecases

import (
	"math/rand"
	"sort"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai"
)

const maxRounds = 100

// initTurnOrder returns NPC InstanceIDs sorted by initiative (descending).
// On initiative ties, NPCs go before PCs (convention). Only NPCs are returned.
func initTurnOrder(participants []models.ParticipantFull) []string {
	// Make a copy to avoid mutating the original slice order.
	sorted := make([]models.ParticipantFull, len(participants))
	copy(sorted, participants)

	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Initiative != sorted[j].Initiative {
			return sorted[i].Initiative > sorted[j].Initiative
		}
		// NPC first on ties.
		return !sorted[i].IsPlayerCharacter && sorted[j].IsPlayerCharacter
	})

	var npcIDs []string
	for i := range sorted {
		if !sorted[i].IsPlayerCharacter {
			npcIDs = append(npcIDs, sorted[i].InstanceID)
		}
	}

	return npcIDs
}

// processStartOfTurn handles start-of-turn effects BEFORE the AI decision.
// Mutates npc.RuntimeState in-place. Returns true if any state changed.
func processStartOfTurn(npc *models.ParticipantFull, creature *models.Creature) bool {
	changed := false

	// 1. Recharge roll: for each action with Recharge, roll d6.
	changed = processRechargeRolls(npc, creature) || changed

	// 2. Reaction reset.
	if npc.RuntimeState.Resources.ReactionUsed {
		npc.RuntimeState.Resources.ReactionUsed = false
		changed = true
	}

	// 3. Legendary actions restore.
	changed = restoreLegendaryActions(npc, creature) || changed

	// 4. Bonus action reset.
	if npc.RuntimeState.Resources.BonusActionUsed {
		npc.RuntimeState.Resources.BonusActionUsed = false
		changed = true
	}

	// 5. Condition duration tick.
	changed = tickConditions(npc) || changed

	return changed
}

// processRechargeRolls rolls d6 for each spent recharge ability.
func processRechargeRolls(npc *models.ParticipantFull, creature *models.Creature) bool {
	changed := false

	for i := range creature.StructuredActions {
		action := &creature.StructuredActions[i]
		if action.Recharge == nil {
			continue
		}

		if npc.RuntimeState.Resources.RechargeReady == nil {
			npc.RuntimeState.Resources.RechargeReady = make(map[string]bool)
		}

		// If already ready, skip.
		if npc.RuntimeState.Resources.RechargeReady[action.ID] {
			continue
		}

		// Roll d6.
		roll := rand.Intn(6) + 1
		if roll >= action.Recharge.MinRoll {
			npc.RuntimeState.Resources.RechargeReady[action.ID] = true
			changed = true
		}
	}

	return changed
}

// restoreLegendaryActions restores legendary actions to max (D&D 5e: start of creature's turn).
func restoreLegendaryActions(npc *models.ParticipantFull, creature *models.Creature) bool {
	count := parseLegendaryCount(creature)
	if count <= 0 {
		return false
	}

	if npc.RuntimeState.Resources.LegendaryActions == count {
		return false
	}

	npc.RuntimeState.Resources.LegendaryActions = count
	return true
}

// parseLegendaryCount extracts the legendary action count from creature.Legendary.Count.
// Count is interface{} — may be float64 (JSON), int32/int64 (BSON), int, or string.
func parseLegendaryCount(creature *models.Creature) int {
	switch v := creature.Legendary.Count.(type) {
	case float64:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

// tickConditions processes condition durations at start of turn.
// Removes expired conditions and decrements round counters.
func tickConditions(npc *models.ParticipantFull) bool {
	if len(npc.RuntimeState.Conditions) == 0 {
		return false
	}

	original := len(npc.RuntimeState.Conditions)
	kept := npc.RuntimeState.Conditions[:0]

	for i := range npc.RuntimeState.Conditions {
		c := &npc.RuntimeState.Conditions[i]

		// Remove conditions that end at start of this NPC's turn.
		if c.EndsOnTurn == "start" && c.TurnEntityID == npc.InstanceID {
			continue
		}

		// Decrement round-based conditions.
		if c.Duration == models.DurationRounds && c.RoundsLeft > 0 {
			c.RoundsLeft--
			if c.RoundsLeft <= 0 {
				continue
			}
		}

		kept = append(kept, *c)
	}

	npc.RuntimeState.Conditions = kept
	return len(kept) != original
}

// checkCombatEnd checks if combat should end.
func checkCombatEnd(participants []models.ParticipantFull, currentRound int) (bool, string) {
	if currentRound > maxRounds {
		return true, "timeout"
	}

	pcAlive := false
	npcAlive := false

	for i := range participants {
		p := &participants[i]
		if combatai.GetCurrentHP(p) <= 0 {
			continue
		}
		if p.IsPlayerCharacter {
			pcAlive = true
		} else {
			npcAlive = true
		}
	}

	if !pcAlive {
		return true, "defeat"
	}
	if !npcAlive {
		return true, "victory"
	}

	return false, ""
}
