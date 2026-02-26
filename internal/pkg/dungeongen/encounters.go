package dungeongen

import "math/rand"

// DungeonDifficulty describes the overall difficulty of a dungeon or room.
type DungeonDifficulty string

const (
	DifficultyEasy   DungeonDifficulty = "easy"
	DifficultyMedium DungeonDifficulty = "medium"
	DifficultyHard   DungeonDifficulty = "hard"
	DifficultyDeadly DungeonDifficulty = "deadly"
)

// CreatureSummary is a lightweight creature reference for encounter budgeting.
type CreatureSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CR           string `json:"cr"`
	XP           int    `json:"xp"`
	CreatureType string `json:"creatureType"` // "undead", "beast", "humanoid", etc.
}

// MonsterSpawn describes a group of identical creatures placed in a room.
type MonsterSpawn struct {
	CreatureID   string   `json:"creatureId"`
	CreatureName string   `json:"creatureName"`
	CR           string   `json:"cr"`
	XP           int      `json:"xp"`
	Count        int      `json:"count"`
	Positions    [][2]int `json:"positions"`
}

// EncounterSetup describes the combat encounter for a single room.
type EncounterSetup struct {
	XPBudget       int               `json:"xpBudget"`
	RoomDifficulty DungeonDifficulty `json:"roomDifficulty"`
	Monsters       []MonsterSpawn    `json:"monsters"`
}

// xpThresholds maps party level → per-character XP thresholds by difficulty.
// Source: DMG p.82.
var xpThresholds = map[int][4]int{
	// [easy, medium, hard, deadly]
	1:  {25, 50, 75, 100},
	2:  {50, 100, 150, 200},
	3:  {75, 150, 225, 400},
	4:  {125, 250, 375, 500},
	5:  {250, 500, 750, 1100},
	6:  {300, 600, 900, 1400},
	7:  {350, 750, 1100, 1700},
	8:  {450, 900, 1400, 2100},
	9:  {550, 1100, 1600, 2400},
	10: {600, 1200, 1900, 2800},
}

// difficultyIndex maps difficulty to threshold array index.
var difficultyIndex = map[DungeonDifficulty]int{
	DifficultyEasy:   0,
	DifficultyMedium: 1,
	DifficultyHard:   2,
	DifficultyDeadly: 3,
}

// difficultyOffset is added to party level to determine max CR.
var difficultyOffset = map[DungeonDifficulty]int{
	DifficultyEasy:   0,
	DifficultyMedium: 1,
	DifficultyHard:   2,
	DifficultyDeadly: 3,
}

// crToNumeric converts a CR string to a float for comparison.
func crToNumeric(cr string) float64 {
	switch cr {
	case "0":
		return 0
	case "1/8":
		return 0.125
	case "1/4":
		return 0.25
	case "1/2":
		return 0.5
	default:
		// Try integer
		val := 0
		for _, c := range cr {
			if c >= '0' && c <= '9' {
				val = val*10 + int(c-'0')
			}
		}
		return float64(val)
	}
}

// encounterMultiplier returns the XP multiplier based on monster count (DMG p.82).
func encounterMultiplier(count int) float64 {
	switch {
	case count <= 1:
		return 1.0
	case count == 2:
		return 1.5
	case count <= 6:
		return 2.0
	case count <= 10:
		return 2.5
	case count <= 14:
		return 3.0
	default:
		return 4.0
	}
}

// adjustedXP returns the adjusted XP for an encounter (raw XP * multiplier).
func adjustedXP(rawXP, monsterCount int) float64 {
	return float64(rawXP) * encounterMultiplier(monsterCount)
}

// BudgetEncounters assigns combat encounters to all combat/boss rooms.
func BudgetEncounters(
	graph *DungeonGraph,
	creatures []CreatureSummary,
	partyLevel, partySize int,
	dungeonDifficulty DungeonDifficulty,
	rng *rand.Rand,
) map[string]*EncounterSetup {
	// Filter creatures by CR range
	crMin := float64(partyLevel / 3)
	crMax := float64(partyLevel + difficultyOffset[dungeonDifficulty])

	eligible := filterCreaturesByCR(creatures, crMin, crMax)
	if len(eligible) == 0 {
		// Fallback: use all creatures CR ≤ 0.5
		eligible = filterCreaturesByCR(creatures, 0, 0.5)
	}
	if len(eligible) == 0 {
		// Fallback 2: use all creatures
		eligible = creatures
	}

	encounters := make(map[string]*EncounterSetup)

	for i := range graph.Rooms {
		room := &graph.Rooms[i]

		// Only combat/boss rooms get encounters
		var roomDiff DungeonDifficulty
		switch room.Type {
		case RoomBoss:
			roomDiff = DifficultyDeadly
		case RoomCombat:
			if rng.Intn(2) == 0 {
				roomDiff = DifficultyMedium
			} else {
				roomDiff = DifficultyHard
			}
		case RoomCombatOptional:
			if rng.Intn(2) == 0 {
				roomDiff = DifficultyEasy
			} else {
				roomDiff = DifficultyMedium
			}
		default:
			continue
		}

		// Calculate XP budget
		thresholds := xpThresholds[partyLevel]
		if partyLevel < 1 || partyLevel > 10 {
			thresholds = xpThresholds[1]
		}
		budget := partySize * thresholds[difficultyIndex[roomDiff]]

		// Greedy monster selection (max 20 iterations)
		monsters := greedySelectMonsters(eligible, budget, rng)

		// Place monsters in room
		placeMonsters(monsters, room.Bounds, rng)

		encounters[room.ID] = &EncounterSetup{
			XPBudget:       budget,
			RoomDifficulty: roomDiff,
			Monsters:       monsters,
		}
	}

	return encounters
}

// greedySelectMonsters fills an encounter with creatures up to the budget.
func greedySelectMonsters(eligible []CreatureSummary, budget int, rng *rand.Rand) []MonsterSpawn {
	if len(eligible) == 0 {
		return nil
	}

	var spawns []MonsterSpawn
	totalXP := 0
	monsterCount := 0
	maxBudget := float64(budget) * 1.1 // Allow 10% overage

	for iter := 0; iter < 20; iter++ {
		candidate := eligible[rng.Intn(len(eligible))]
		newCount := monsterCount + 1
		newAdjusted := adjustedXP(totalXP+candidate.XP, newCount)

		if newAdjusted > maxBudget {
			break
		}

		// Add or increment monster
		added := false
		for j := range spawns {
			if spawns[j].CreatureID == candidate.ID {
				spawns[j].Count++
				added = true
				break
			}
		}
		if !added {
			spawns = append(spawns, MonsterSpawn{
				CreatureID:   candidate.ID,
				CreatureName: candidate.Name,
				CR:           candidate.CR,
				XP:           candidate.XP,
				Count:        1,
			})
		}

		totalXP += candidate.XP
		monsterCount++
	}

	// Zero-monster fallback: add cheapest creature
	if len(spawns) == 0 && len(eligible) > 0 {
		cheapest := eligible[0]
		for _, c := range eligible[1:] {
			if c.XP < cheapest.XP {
				cheapest = c
			}
		}
		spawns = append(spawns, MonsterSpawn{
			CreatureID:   cheapest.ID,
			CreatureName: cheapest.Name,
			CR:           cheapest.CR,
			XP:           cheapest.XP,
			Count:        1,
		})
	}

	return spawns
}

// placeMonsters assigns positions to each monster spawn within room bounds.
func placeMonsters(spawns []MonsterSpawn, bounds RoomBounds, rng *rand.Rand) {
	used := make(map[[2]int]bool)

	for i := range spawns {
		spawns[i].Positions = make([][2]int, 0, spawns[i].Count)

		for j := 0; j < spawns[i].Count; j++ {
			minR := 2
			maxR := bounds.Rows - 3
			if maxR < minR {
				maxR = minR
			}
			minC := 2
			maxC := bounds.Cols - 3
			if maxC < minC {
				maxC = minC
			}

			var pos [2]int
			for attempt := 0; attempt < 20; attempt++ {
				r := minR + rng.Intn(maxR-minR+1)
				c := minC + rng.Intn(maxC-minC+1)
				pos = [2]int{r, c}
				if !used[pos] {
					break
				}
			}

			used[pos] = true
			spawns[i].Positions = append(spawns[i].Positions, pos)
		}
	}
}

// filterCreaturesByCR returns creatures within the given CR range.
func filterCreaturesByCR(creatures []CreatureSummary, crMin, crMax float64) []CreatureSummary {
	var result []CreatureSummary
	for _, c := range creatures {
		cr := crToNumeric(c.CR)
		if cr >= crMin && cr <= crMax {
			result = append(result, c)
		}
	}
	return result
}
