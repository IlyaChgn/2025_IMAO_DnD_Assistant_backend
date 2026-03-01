package dungeongen

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// GenerateGraph creates an abstract dungeon graph with a linear main path
// and side branches. Port of frontend graphGenerator.ts generateAbstractGraph.
func GenerateGraph(config DungeonConfig, rng *rand.Rand) *DungeonGraph {
	sizeRange, ok := SizeRanges[config.Size]
	if !ok {
		sizeRange = SizeRanges[SizeMedium]
	}

	// Step 1: Determine total room count
	roomCount := rng.Intn(sizeRange.Max-sizeRange.Min+1) + sizeRange.Min

	// Step 2: Split into main path + branches
	mainPathLength := int(math.Ceil(float64(roomCount) * 0.7))
	branchCount := roomCount - mainPathLength

	// Step 3: Create main path rooms (linear chain)
	rooms := make([]DungeonRoom, 0, roomCount)
	for i := 0; i < mainPathLength; i++ {
		rooms = append(rooms, makeRoom(fmt.Sprintf("room_%d", i), RoomCombat, GraphPosition{X: i, Y: 0}))
	}

	// Step 4: Create main path connections
	connections := make([]RoomConnection, 0, roomCount-1)
	for i := 0; i < mainPathLength-1; i++ {
		connections = append(connections, RoomConnection{
			ID:         fmt.Sprintf("conn_%d_%d", i, i+1),
			FromRoomID: rooms[i].ID,
			ToRoomID:   rooms[i+1].ID,
		})
	}

	// Step 5: Create branch rooms
	usedParents := make(map[int]bool)
	for b := 0; b < branchCount; b++ {
		// Pick random main path room as parent (exclude entrance & last)
		parentIdx := rng.Intn(mainPathLength-2) + 1 // [1, mainPathLength-2]

		// Find next free parent (circular search)
		attempts := 0
		for usedParents[parentIdx] && attempts < mainPathLength {
			parentIdx = (parentIdx + 1) % (mainPathLength - 1)
			if parentIdx == 0 {
				parentIdx = 1
			}
			attempts++
		}

		if usedParents[parentIdx] {
			break // No free parents available
		}
		usedParents[parentIdx] = true

		// Alternate branch placement: y=+1 (above) or y=-1 (below)
		yOffset := 1
		if b%2 != 0 {
			yOffset = -1
		}

		branchRoom := makeRoom(
			fmt.Sprintf("room_%d", len(rooms)),
			RoomCombat,
			GraphPosition{X: parentIdx, Y: yOffset},
		)
		rooms = append(rooms, branchRoom)

		connections = append(connections, RoomConnection{
			ID:         fmt.Sprintf("conn_%d_%d", parentIdx, len(rooms)-1),
			FromRoomID: rooms[parentIdx].ID,
			ToRoomID:   branchRoom.ID,
		})
	}

	return &DungeonGraph{
		Rooms:          rooms,
		Connections:    connections,
		MainPathLength: mainPathLength,
	}
}

// AssignRoomTypes assigns gameplay types to rooms and creates extraction points.
// Port of frontend graphGenerator.ts assignRoomTypes.
func AssignRoomTypes(graph *DungeonGraph, rng *rand.Rand, size DungeonSize, secretRoomReveal bool) {
	rooms := graph.Rooms
	mainPathLength := graph.MainPathLength

	// Step 1: Fixed types
	rooms[0].Type = RoomEntrance
	rooms[mainPathLength-1].Type = RoomBoss

	remaining := len(rooms) - 2
	if remaining <= 0 {
		graph.ExtractionPoints = makeExtractionPoints(rooms, mainPathLength)
		return
	}

	// Step 2: Compute distribution with retry
	dist := computeDistribution(rng, remaining, size)
	for retry := 0; retry < 3; retry++ {
		combatPct := float64(dist.combat) / float64(remaining)
		if combatPct >= 0.35 && combatPct <= 0.55 {
			break
		}
		dist = computeDistribution(rng, remaining, size)
	}

	// Step 3: Build and shuffle type pool
	typePool := buildTypePool(dist)
	shuffleRoomTypes(typePool, rng)

	// Step 4: Identify branch room indices
	branchIndices := make(map[int]bool)
	for i := mainPathLength; i < len(rooms); i++ {
		branchIndices[i] = true
	}

	// Step 5: Build unassigned list (skip entrance and boss)
	unassigned := make([]int, 0, remaining)
	for i := 1; i < len(rooms); i++ {
		if i == mainPathLength-1 {
			continue // Skip boss
		}
		unassigned = append(unassigned, i)
	}

	// Step 6: Sort — branch rooms first (for secret room priority)
	sort.Slice(unassigned, func(a, b int) bool {
		aIsBranch := branchIndices[unassigned[a]]
		bIsBranch := branchIndices[unassigned[b]]
		if aIsBranch != bIsBranch {
			return aIsBranch // branches (true) sort before main path (false)
		}
		return unassigned[a] < unassigned[b]
	})

	// Step 7: Middle third bounds for rest room constraint
	middleThirdStart := mainPathLength / 3
	middleThirdEnd := (2 * mainPathLength) / 3

	// Step 8: Assign types with constraints
	typeIdx := 0
	for _, roomIdx := range unassigned {
		if typeIdx >= len(typePool) {
			rooms[roomIdx].Type = RoomCombat // Fallback
			continue
		}

		assignedType := typePool[typeIdx]

		// Constraint A: Secret rooms only on branches
		if assignedType == RoomSecret && !branchIndices[roomIdx] {
			swapped := false
			for j := typeIdx + 1; j < len(typePool); j++ {
				if typePool[j] != RoomSecret {
					typePool[typeIdx], typePool[j] = typePool[j], typePool[typeIdx]
					assignedType = typePool[typeIdx]
					swapped = true
					break
				}
			}
			if !swapped {
				assignedType = RoomCombat
			}
		}

		// Constraint B: Rest rooms only in middle third of main path
		if assignedType == RoomRest && roomIdx < mainPathLength {
			if roomIdx < middleThirdStart || roomIdx > middleThirdEnd {
				for j := typeIdx + 1; j < len(typePool); j++ {
					if typePool[j] != RoomRest && typePool[j] != RoomSecret {
						typePool[typeIdx], typePool[j] = typePool[j], typePool[typeIdx]
						assignedType = typePool[typeIdx]
						break
					}
				}
			}
		}

		// Constraint C: Treasure not in positions 1-2 from start
		if assignedType == RoomTreasure && roomIdx <= 2 {
			for j := typeIdx + 1; j < len(typePool); j++ {
				if typePool[j] != RoomTreasure && typePool[j] != RoomSecret {
					typePool[typeIdx], typePool[j] = typePool[j], typePool[typeIdx]
					assignedType = typePool[typeIdx]
					break
				}
			}
		}

		rooms[roomIdx].Type = assignedType
		typeIdx++
	}

	// Step 9: Secret room reveal
	if secretRoomReveal {
		for i := range rooms {
			if rooms[i].Type == RoomSecret {
				rooms[i].Discovered = true
			}
		}
	}

	graph.ExtractionPoints = makeExtractionPoints(rooms, mainPathLength)
}

// distribution holds target counts for each room type.
type distribution struct {
	combat   int
	treasure int
	trap     int
	rest     int
	secret   int
}

func computeDistribution(rng *rand.Rand, remaining int, size DungeonSize) distribution {
	r := float64(remaining)

	combat := int(math.Round(r * (0.4 + rng.Float64()*0.1)))     // 40-50%
	treasure := int(math.Round(r * (0.15 + rng.Float64()*0.05))) // 15-20%
	trap := int(math.Round(r * (0.1 + rng.Float64()*0.05)))      // 10-15%

	var rest int
	if size == SizeLong {
		rest = max(1, int(math.Round(r*0.08)))
	} else {
		rest = int(math.Round(r * rng.Float64() * 0.1)) // 0-10%
	}

	secret := max(0, int(math.Round(r*0.05))) // ~5%

	// Adjust combat to make sum equal remaining
	sum := combat + treasure + trap + rest + secret
	adjustment := remaining - sum
	combat = max(1, combat+adjustment)

	// Trim if still overshooting
	total := combat + treasure + trap + rest + secret
	if total > remaining {
		excess := total - remaining
		combat = max(1, combat-excess)
	}

	return distribution{combat, treasure, trap, rest, secret}
}

func buildTypePool(dist distribution) []RoomType {
	pool := make([]RoomType, 0, dist.combat+dist.treasure+dist.trap+dist.rest+dist.secret)

	combatOptional := dist.combat * 35 / 100
	combatRegular := dist.combat - combatOptional

	for i := 0; i < combatRegular; i++ {
		pool = append(pool, RoomCombat)
	}
	for i := 0; i < combatOptional; i++ {
		pool = append(pool, RoomCombatOptional)
	}
	for i := 0; i < dist.treasure; i++ {
		pool = append(pool, RoomTreasure)
	}
	for i := 0; i < dist.trap; i++ {
		pool = append(pool, RoomTrap)
	}
	for i := 0; i < dist.rest; i++ {
		pool = append(pool, RoomRest)
	}
	for i := 0; i < dist.secret; i++ {
		pool = append(pool, RoomSecret)
	}

	return pool
}

// shuffleRoomTypes performs Fisher-Yates shuffle in-place.
func shuffleRoomTypes(types []RoomType, rng *rand.Rand) {
	for i := len(types) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		types[i], types[j] = types[j], types[i]
	}
}

func makeRoom(id string, roomType RoomType, pos GraphPosition) DungeonRoom {
	return DungeonRoom{
		ID:            id,
		Type:          roomType,
		GraphPosition: pos,
		Discovered:    false,
	}
}

func makeExtractionPoints(rooms []DungeonRoom, mainPathLength int) []ExtractionPoint {
	return []ExtractionPoint{
		{RoomID: rooms[0].ID, Type: "entrance", InitiallyAvailable: true},
		{RoomID: rooms[mainPathLength-1].ID, Type: "boss_exit", InitiallyAvailable: false},
	}
}
