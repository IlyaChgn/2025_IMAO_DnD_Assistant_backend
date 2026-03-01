package dungeongen

import (
	"math/rand"
	"testing"
)

func TestGenerateGraph_Determinism(t *testing.T) {
	seed := int64(42)
	config := DungeonConfig{Seed: seed, Size: SizeMedium}

	rng1 := rand.New(rand.NewSource(seed))
	g1 := GenerateGraph(config, rng1)

	rng2 := rand.New(rand.NewSource(seed))
	g2 := GenerateGraph(config, rng2)

	if len(g1.Rooms) != len(g2.Rooms) {
		t.Fatalf("determinism: room counts differ: %d vs %d", len(g1.Rooms), len(g2.Rooms))
	}
	for i := range g1.Rooms {
		if g1.Rooms[i].ID != g2.Rooms[i].ID {
			t.Errorf("determinism: room %d IDs differ: %q vs %q", i, g1.Rooms[i].ID, g2.Rooms[i].ID)
		}
	}
	if len(g1.Connections) != len(g2.Connections) {
		t.Fatalf("determinism: connection counts differ: %d vs %d", len(g1.Connections), len(g2.Connections))
	}
}

func TestGenerateGraph_DifferentSeeds(t *testing.T) {
	config1 := DungeonConfig{Seed: 1, Size: SizeMedium}
	config2 := DungeonConfig{Seed: 99999, Size: SizeMedium}

	rng1 := rand.New(rand.NewSource(1))
	g1 := GenerateGraph(config1, rng1)

	rng2 := rand.New(rand.NewSource(99999))
	g2 := GenerateGraph(config2, rng2)

	// Very unlikely to be identical with different seeds
	different := len(g1.Rooms) != len(g2.Rooms)
	if !different {
		for i := range g1.Rooms {
			if g1.Rooms[i].GraphPosition != g2.Rooms[i].GraphPosition {
				different = true
				break
			}
		}
	}
	if !different {
		t.Error("different seeds produced identical graphs (extremely unlikely)")
	}
}

func TestGenerateGraph_RoomCountWithinRange(t *testing.T) {
	sizes := []DungeonSize{SizeShort, SizeMedium, SizeLong}

	for _, size := range sizes {
		t.Run(string(size), func(t *testing.T) {
			sr := SizeRanges[size]
			for seed := int64(0); seed < 20; seed++ {
				config := DungeonConfig{Seed: seed, Size: size}
				rng := rand.New(rand.NewSource(seed))
				g := GenerateGraph(config, rng)

				if len(g.Rooms) < sr.Min || len(g.Rooms) > sr.Max {
					t.Errorf("seed %d: room count %d outside [%d, %d]",
						seed, len(g.Rooms), sr.Min, sr.Max)
				}
			}
		})
	}
}

func TestGenerateGraph_MainPathLinear(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)

	if g.MainPathLength < 2 {
		t.Fatal("main path too short")
	}

	// All main path rooms should have Y=0
	for i := 0; i < g.MainPathLength; i++ {
		if g.Rooms[i].GraphPosition.Y != 0 {
			t.Errorf("main path room %d has Y=%d, want 0", i, g.Rooms[i].GraphPosition.Y)
		}
		if g.Rooms[i].GraphPosition.X != i {
			t.Errorf("main path room %d has X=%d, want %d", i, g.Rooms[i].GraphPosition.X, i)
		}
	}
}

func TestGenerateGraph_BranchRoomsOffMainPath(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeLong}, rng)

	for i := g.MainPathLength; i < len(g.Rooms); i++ {
		if g.Rooms[i].GraphPosition.Y == 0 {
			t.Errorf("branch room %d has Y=0, should be +1 or -1", i)
		}
	}
}

func TestGenerateGraph_MaxOneBranchPerParent(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		rng := rand.New(rand.NewSource(seed))
		g := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeLong}, rng)

		parentCount := make(map[int]int)
		for i := g.MainPathLength; i < len(g.Rooms); i++ {
			parentCount[g.Rooms[i].GraphPosition.X]++
		}
		for parent, count := range parentCount {
			if count > 1 {
				t.Errorf("seed %d: parent %d has %d branches, max 1", seed, parent, count)
			}
		}
	}
}

func TestGenerateGraph_AllRoomsConnected(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeLong}, rng)

	// BFS from room_0
	adj := make(map[string][]string)
	for _, c := range g.Connections {
		adj[c.FromRoomID] = append(adj[c.FromRoomID], c.ToRoomID)
		adj[c.ToRoomID] = append(adj[c.ToRoomID], c.FromRoomID)
	}

	visited := make(map[string]bool)
	queue := []string{g.Rooms[0].ID}
	visited[g.Rooms[0].ID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, neighbor := range adj[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	if len(visited) != len(g.Rooms) {
		t.Errorf("BFS reached %d rooms, but graph has %d", len(visited), len(g.Rooms))
	}
}

// --- B2 tests: AssignRoomTypes ---

func TestAssignRoomTypes_EntranceAndBoss(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		g := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeMedium}, rng)
		AssignRoomTypes(g, rng, SizeMedium, false)

		if g.Rooms[0].Type != RoomEntrance {
			t.Errorf("seed %d: first room type %q, want %q", seed, g.Rooms[0].Type, RoomEntrance)
		}
		if g.Rooms[g.MainPathLength-1].Type != RoomBoss {
			t.Errorf("seed %d: boss room type %q, want %q", seed, g.Rooms[g.MainPathLength-1].Type, RoomBoss)
		}
	}
}

func TestAssignRoomTypes_ExtractionPoints(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(g, rng, SizeMedium, false)

	if len(g.ExtractionPoints) != 2 {
		t.Fatalf("expected 2 extraction points, got %d", len(g.ExtractionPoints))
	}

	if g.ExtractionPoints[0].Type != "entrance" || !g.ExtractionPoints[0].InitiallyAvailable {
		t.Errorf("extraction point 0: %+v", g.ExtractionPoints[0])
	}
	if g.ExtractionPoints[1].Type != "boss_exit" || g.ExtractionPoints[1].InitiallyAvailable {
		t.Errorf("extraction point 1: %+v", g.ExtractionPoints[1])
	}
}

func TestAssignRoomTypes_SecretOnlyOnBranches(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		rng := rand.New(rand.NewSource(seed))
		g := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeLong}, rng)
		AssignRoomTypes(g, rng, SizeLong, false)

		for i := 0; i < g.MainPathLength; i++ {
			if g.Rooms[i].Type == RoomSecret {
				t.Errorf("seed %d: main path room %d is secret", seed, i)
			}
		}
	}
}

func TestAssignRoomTypes_CombatPercentage(t *testing.T) {
	for seed := int64(0); seed < 30; seed++ {
		rng := rand.New(rand.NewSource(seed))
		g := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeMedium}, rng)
		AssignRoomTypes(g, rng, SizeMedium, false)

		combatCount := 0
		for _, r := range g.Rooms {
			if r.Type == RoomCombat || r.Type == RoomCombatOptional {
				combatCount++
			}
		}

		remaining := len(g.Rooms) - 2 // exclude entrance & boss
		if remaining <= 0 {
			continue
		}
		pct := float64(combatCount) / float64(remaining)
		// Allow wide tolerance: constraint swaps + fallback can push combat up
		if pct < 0.15 || pct > 0.85 {
			t.Errorf("seed %d: combat%% = %.2f, outside [0.15, 0.85]", seed, pct)
		}
	}
}

func TestAssignRoomTypes_SecretRoomReveal(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeLong}, rng)
	AssignRoomTypes(g, rng, SizeLong, true)

	for _, r := range g.Rooms {
		if r.Type == RoomSecret && !r.Discovered {
			t.Errorf("secret room %s not discovered with secretRoomReveal=true", r.ID)
		}
	}
}

func TestAssignRoomTypes_AllRoomsAssigned(t *testing.T) {
	validTypes := map[RoomType]bool{
		RoomEntrance: true, RoomCombat: true, RoomCombatOptional: true,
		RoomTreasure: true, RoomTrap: true, RoomRest: true,
		RoomBoss: true, RoomExtraction: true, RoomSecret: true,
	}

	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		g := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeMedium}, rng)
		AssignRoomTypes(g, rng, SizeMedium, false)

		for _, r := range g.Rooms {
			if !validTypes[r.Type] {
				t.Errorf("seed %d: room %s has invalid type %q", seed, r.ID, r.Type)
			}
		}
	}
}
