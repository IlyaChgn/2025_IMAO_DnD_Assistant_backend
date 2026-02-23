package combatai

import (
	"math/rand"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// FindOpportunityAttackCandidates identifies NPCs eligible for opportunity attacks
// when movingInstanceID moves from oldPos to newPos.
// Only checks NPCs (not PCs). Skips dead, incapacitated, ReactionUsed NPCs.
// Returns candidates sorted by initiative order (same as participants slice order).
func FindOpportunityAttackCandidates(
	participants []models.ParticipantFull,
	movingInstanceID string,
	oldPos, newPos *models.CellsCoordinates,
	creatures map[string]*models.Creature,
) []OpportunityAttackCandidate {
	if oldPos == nil || newPos == nil {
		return nil
	}

	var candidates []OpportunityAttackCandidate

	for i := range participants {
		p := &participants[i]

		// Skip self.
		if p.InstanceID == movingInstanceID {
			continue
		}

		// Only NPCs make automated opportunity attacks (Phase 2).
		if p.IsPlayerCharacter {
			continue
		}

		// Skip dead NPCs.
		if GetCurrentHP(p) <= 0 {
			continue
		}

		// Skip incapacitated NPCs.
		if IsIncapacitated(p) {
			continue
		}

		// Skip if reaction already used this round.
		if p.RuntimeState.Resources.ReactionUsed {
			continue
		}

		// Skip NPCs without grid position.
		if p.CellsCoords == nil {
			continue
		}

		// Load creature template.
		creature, ok := creatures[p.InstanceID]
		if !ok || creature == nil {
			continue
		}

		// Find a suitable melee action for opportunity attack.
		action := selectOpportunityAttackAction(creature)
		if action == nil {
			continue
		}

		// Get melee reach for the selected action.
		reach := meleeReach(action)

		// Check D&D 5e opportunity attack condition:
		// old position was within reach AND new position is outside reach.
		distOld := DistanceFt(p.CellsCoords, oldPos)
		distNew := DistanceFt(p.CellsCoords, newPos)

		if distOld <= reach && distNew > reach {
			candidates = append(candidates, OpportunityAttackCandidate{
				NPC:      p,
				Creature: creature,
				Action:   action,
			})
		}
	}

	return candidates
}

// selectOpportunityAttackAction picks the melee action for an opportunity attack.
// Priority: 1) reaction-category melee actions with Attack data,
// 2) best melee action from main actions (fallback — most monsters lack explicit reactions).
// Returns nil if no suitable melee action exists.
func selectOpportunityAttackAction(creature *models.Creature) *models.StructuredAction {
	// 1. Try explicit reaction-category melee actions.
	reactions := actionsByCategory(creature.StructuredActions, models.ActionCategoryReaction)
	best := bestMeleeAttack(reactions)
	if best != nil {
		return best
	}

	// 2. Fallback: best melee action from main actions.
	mainActions := actionsByCategory(creature.StructuredActions, models.ActionCategoryAction)
	return bestMeleeAttack(mainActions)
}

// bestMeleeAttack returns the melee attack with the highest average damage
// from the given actions. Skips ranged, recharge-gated, and actions without Attack data.
func bestMeleeAttack(actions []models.StructuredAction) *models.StructuredAction {
	var best *models.StructuredAction
	bestDmg := 0.0

	for i := range actions {
		a := &actions[i]

		// Must have attack data and be melee.
		if a.Attack == nil {
			continue
		}
		if !isMeleeAction(a) {
			continue
		}

		// Skip recharge-gated actions (opportunity attacks are not the place to use these).
		if a.Recharge != nil {
			continue
		}

		avg := avgDamageRolls(a.Attack.Damage)
		if best == nil || avg > bestDmg {
			best = a
			bestDmg = avg
		}
	}

	return best
}

// meleeReach returns the melee reach in feet for an action.
// Defaults to 5 if not specified (standard D&D 5e melee reach).
func meleeReach(action *models.StructuredAction) int {
	if action != nil && action.Attack != nil && action.Attack.Reach > 0 {
		return action.Attack.Reach
	}
	return 5
}

// ShouldTakeOpportunityAttack returns true if the NPC passes the intelligence gate.
// Dumb NPCs (low INT) may miss the opportunity — e.g., a Zombie (INT 3, intelligence ≈ 0.11)
// has only ~11% chance of reacting.
// If rng is nil, uses the global math/rand source.
func ShouldTakeOpportunityAttack(intelligence float64, rng *rand.Rand) bool {
	if rng != nil {
		return rng.Float64() < intelligence
	}
	return rand.Float64() < intelligence
}
