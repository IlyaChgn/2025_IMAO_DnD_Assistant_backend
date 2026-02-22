package combatai

import (
	"math"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// SelectTarget chooses target(s) for an NPC action based on intelligence tier.
// Returns a slice of target InstanceIDs (usually 1).
// Returns nil if no valid targets exist.
func SelectTarget(input *TurnInput, action *models.StructuredAction) []string {
	enemies := aliveEnemies(input)
	if len(enemies) == 0 {
		return nil
	}

	var target *models.ParticipantFull

	switch {
	case input.Intelligence < 0.15:
		target = selectTierZombie(input, enemies)
	case input.Intelligence < 0.35:
		target = selectTierGoblin(input, enemies, action)
	case input.Intelligence < 0.55:
		target = selectTierNormal(input, enemies, action)
	default:
		target = selectTierSmart(input, enemies, action)
	}

	if target == nil {
		// Ultimate fallback: nearest enemy.
		target = nearestEnemy(input, enemies)
	}
	if target == nil {
		return nil
	}
	return []string{target.InstanceID}
}

// selectTierZombie implements sticky targeting for low-intelligence creatures (INT 1-3).
func selectTierZombie(input *TurnInput, enemies []*models.ParticipantFull) *models.ParticipantFull {
	if input.PreviousTargetID != "" {
		for _, e := range enemies {
			if e.InstanceID == input.PreviousTargetID {
				return e
			}
		}
	}
	return nearestEnemy(input, enemies)
}

// selectTierGoblin attacks nearest, but finishes wounded enemies in reach.
func selectTierGoblin(input *TurnInput, enemies []*models.ParticipantFull, action *models.StructuredAction) *models.ParticipantFull {
	for _, e := range enemies {
		if !inRange(input, e, action) {
			continue
		}
		stats := input.CombatantStats[e.InstanceID]
		if hpPercent(e, stats) < 0.25 {
			return e
		}
	}
	return nearestEnemy(input, enemies)
}

// selectTierNormal picks lowest HP (melee) or lowest AC (ranged) in reach/range.
func selectTierNormal(input *TurnInput, enemies []*models.ParticipantFull, action *models.StructuredAction) *models.ParticipantFull {
	isMelee := isMeleeAction(action)

	var best *models.ParticipantFull
	bestScore := math.MaxFloat64

	for _, e := range enemies {
		if !inRange(input, e, action) {
			continue
		}
		var score float64
		if isMelee {
			score = float64(GetCurrentHP(e)) // lower HP = better target
		} else {
			stats := input.CombatantStats[e.InstanceID]
			score = float64(stats.AC) // lower AC = easier to hit
		}
		if score < bestScore {
			bestScore = score
			best = e
		}
	}

	if best != nil {
		return best
	}
	return nearestEnemy(input, enemies)
}

// selectTierSmart uses threat assessment to pick the best target.
// Considers concentration, HP%, distance, damage type matchup.
func selectTierSmart(input *TurnInput, enemies []*models.ParticipantFull, action *models.StructuredAction) *models.ParticipantFull {
	threats := AssessThreats(input)

	// Build lookup: TargetID → threat score.
	scoreMap := make(map[string]float64, len(threats))
	for _, t := range threats {
		scoreMap[t.TargetID] = t.Score
	}

	// Pick highest-scoring enemy that is in range for this action.
	var best *models.ParticipantFull
	bestScore := math.Inf(-1)

	for _, e := range enemies {
		if !inRange(input, e, action) {
			continue
		}
		score := scoreMap[e.InstanceID]
		if score > bestScore {
			bestScore = score
			best = e
		}
	}

	if best != nil {
		return best
	}
	// Fallback: consider all enemies including out of range.
	return nearestEnemy(input, enemies)
}

// aliveEnemies returns pointers to alive PCs (enemies of the NPC).
func aliveEnemies(input *TurnInput) []*models.ParticipantFull {
	var result []*models.ParticipantFull
	for i := range input.Participants {
		p := &input.Participants[i]
		if p.IsPlayerCharacter && IsAlive(p) {
			result = append(result, p)
		}
	}
	return result
}

// actionRange returns the effective range of an action in feet.
func actionRange(action *models.StructuredAction) int {
	if action == nil {
		return 5 // default melee reach
	}
	if action.Attack != nil {
		if action.Attack.Range != nil {
			return action.Attack.Range.Normal
		}
		if action.Attack.Reach > 0 {
			return action.Attack.Reach
		}
		return 5 // default melee reach
	}
	if action.SavingThrow != nil {
		return math.MaxInt32 // save-based abilities are treated as unlimited range
	}
	return 5
}

// inRange checks if a target is within the action's range of the active NPC.
func inRange(input *TurnInput, target *models.ParticipantFull, action *models.StructuredAction) bool {
	dist := DistanceFt(input.ActiveNPC.CellsCoords, target.CellsCoords)
	return dist <= actionRange(action)
}

// nearestEnemy returns the closest alive enemy by grid distance.
func nearestEnemy(input *TurnInput, enemies []*models.ParticipantFull) *models.ParticipantFull {
	var best *models.ParticipantFull
	bestDist := math.MaxInt32

	for _, e := range enemies {
		d := DistanceFt(input.ActiveNPC.CellsCoords, e.CellsCoords)
		if d < bestDist {
			bestDist = d
			best = e
		}
	}
	return best
}

// hpPercent returns the ratio of current HP to max HP (0.0–1.0).
// Returns 1.0 if MaxHP is unknown (0), to avoid false "wounded" detection.
func hpPercent(p *models.ParticipantFull, stats CombatantStats) float64 {
	maxHP := stats.MaxHP
	if maxHP <= 0 {
		return 1.0
	}
	return float64(GetCurrentHP(p)) / float64(maxHP)
}

// isMeleeAction returns true if the action is a melee attack.
func isMeleeAction(action *models.StructuredAction) bool {
	if action == nil || action.Attack == nil {
		return true // default: treat unknown as melee for targeting
	}
	switch action.Attack.Type {
	case models.AttackRollMeleeWeapon, models.AttackRollMeleeSpell:
		return true
	}
	return false
}
