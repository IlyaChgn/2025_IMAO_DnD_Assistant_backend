package combatai

import (
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// ThreatScore holds the computed threat score for a single enemy target.
// Used by selectTierSmart() to prioritize targets.
type ThreatScore struct {
	TargetID        string
	Score           float64
	Distance        int // feet (Chebyshev)
	IsConcentrating bool
	HPPercent       float64
}

// AssessThreats computes threat scores for all alive enemies.
func AssessThreats(input *TurnInput) []ThreatScore {
	enemies := aliveEnemies(input)
	scores := make([]ThreatScore, 0, len(enemies))

	for _, e := range enemies {
		stats := input.CombatantStats[e.InstanceID]
		scores = append(scores, assessSingleThreat(input, e, stats))
	}

	return scores
}

// assessSingleThreat computes the threat score for one enemy using the design
// plan formula (section E.4).
func assessSingleThreat(input *TurnInput, enemy *models.ParticipantFull, stats CombatantStats) ThreatScore {
	dist := DistanceFt(input.ActiveNPC.CellsCoords, enemy.CellsCoords)
	hp := hpPercent(enemy, stats)
	concentrating := enemy.RuntimeState.Concentration != nil

	// Base: estimated target DPR (heuristic — no PC weapon data in TurnInput).
	score := estimateTargetDPR(stats)

	// Concentration bonus — breaking concentration is high priority.
	if concentrating {
		score += 30
	}

	// Low HP bonuses — finish off wounded targets.
	if hp < 0.25 {
		score += 25
	} else if hp < 0.50 {
		score += 10
	}

	// Distance penalty — targets beyond movement+reach are harder to reach.
	movement := input.CreatureTemplate.Movement.Walk
	reach := maxMeleeReach(input.CreatureTemplate)
	if dist > movement+reach {
		score -= 15
	}

	// Damage type matchup — avoid immune/resistant targets, prefer vulnerable.
	dmgType := mainDamageType(&input.CreatureTemplate)
	if dmgType != "" {
		if containsDamageType(stats.Immunities, dmgType) {
			score -= 50
		} else if containsDamageType(stats.Resistances, dmgType) {
			score -= 20
		}
		if containsDamageType(stats.Vulnerabilities, dmgType) {
			score += 20
		}
	}

	return ThreatScore{
		TargetID:        enemy.InstanceID,
		Score:           score,
		Distance:        dist,
		IsConcentrating: concentrating,
		HPPercent:       hp,
	}
}

// estimateTargetDPR returns a heuristic estimate of the target's damage per round.
// TurnInput does not include PC weapon/spell data, so we use a fixed default.
// Can be improved in a future PR by adding EstimatedDPR to CombatantStats.
func estimateTargetDPR(_ CombatantStats) float64 {
	return 10.0
}

// mainDamageType returns the primary damage type of the creature's most damaging
// StructuredAction. Returns "" if the creature has no damage-dealing actions.
func mainDamageType(creature *models.Creature) string {
	bestType := ""
	bestAvg := 0.0

	for i := range creature.StructuredActions {
		a := &creature.StructuredActions[i]

		var rolls []models.DamageRoll
		if a.Attack != nil {
			rolls = a.Attack.Damage
		} else if a.SavingThrow != nil {
			rolls = a.SavingThrow.Damage
		}

		for _, dr := range rolls {
			avg := float64(dr.DiceCount)*(float64(parseDiceMax(dr.DiceType))+1)/2 + float64(dr.Bonus)
			if avg > bestAvg {
				bestAvg = avg
				bestType = dr.DamageType
			}
		}
	}

	return bestType
}

// maxMeleeReach returns the maximum melee reach from the creature's StructuredActions.
// Defaults to 5 feet (standard D&D melee reach) if no melee actions found.
func maxMeleeReach(creature models.Creature) int {
	best := 0
	for i := range creature.StructuredActions {
		a := &creature.StructuredActions[i]
		if a.Attack == nil {
			continue
		}
		switch a.Attack.Type {
		case models.AttackRollMeleeWeapon, models.AttackRollMeleeSpell:
			reach := a.Attack.Reach
			if reach > best {
				best = reach
			}
		}
	}
	if best == 0 {
		return 5
	}
	return best
}

// containsDamageType checks if a string slice contains a damage type (case-insensitive).
func containsDamageType(types []string, dmgType string) bool {
	lower := strings.ToLower(dmgType)
	for _, t := range types {
		if strings.ToLower(t) == lower {
			return true
		}
	}
	return false
}
