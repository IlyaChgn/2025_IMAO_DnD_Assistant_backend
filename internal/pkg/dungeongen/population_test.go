package dungeongen

import (
	"math/rand"
	"testing"
)

func testTheme() *ThemeDefinition {
	return DefaultThemes["catacombs"]
}

func TestPopulateRooms_TreasureHasLoot(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(graph, rng, SizeMedium, false)

	result := PopulateRooms(graph, testTheme(), 3, false, rng)

	for _, room := range graph.Rooms {
		if room.Type == RoomTreasure {
			loot, ok := result.Loot[room.ID]
			if !ok {
				t.Errorf("treasure room %s has no loot", room.ID)
				continue
			}
			if loot.Gold <= 0 {
				t.Errorf("treasure room %s: gold = %d, want > 0", room.ID, loot.Gold)
			}
			if loot.Container != "chest" {
				t.Errorf("treasure room %s: container = %q, want %q", room.ID, loot.Container, "chest")
			}
		}
	}
}

func TestPopulateRooms_BossHasLoot(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(graph, rng, SizeMedium, false)

	result := PopulateRooms(graph, testTheme(), 3, false, rng)

	for _, room := range graph.Rooms {
		if room.Type == RoomBoss {
			loot, ok := result.Loot[room.ID]
			if !ok {
				t.Errorf("boss room %s has no loot", room.ID)
				continue
			}
			if loot.Container != "scattered" {
				t.Errorf("boss room %s: container = %q, want %q", room.ID, loot.Container, "scattered")
			}
		}
	}
}

func TestPopulateRooms_TrapRoomsHaveTraps(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		graph := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeMedium}, rng)
		AssignRoomTypes(graph, rng, SizeMedium, false)

		result := PopulateRooms(graph, testTheme(), 3, false, rng)

		for _, room := range graph.Rooms {
			if room.Type == RoomTrap {
				traps, ok := result.Traps[room.ID]
				if !ok || len(traps) == 0 {
					t.Errorf("seed %d: trap room %s has no traps", seed, room.ID)
					continue
				}
				for _, trap := range traps {
					if trap.DetectionDC < 10 || trap.DetectionDC > 25 {
						t.Errorf("seed %d: trap DC %d out of range", seed, trap.DetectionDC)
					}
					if trap.DisarmDC < trap.DetectionDC {
						t.Errorf("seed %d: disarm DC %d < detection DC %d", seed, trap.DisarmDC, trap.DetectionDC)
					}
				}
			}
		}
	}
}

func TestPopulateRooms_SecretRoomsHaveSecrets(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		graph := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeLong}, rng)
		AssignRoomTypes(graph, rng, SizeLong, false)

		result := PopulateRooms(graph, testTheme(), 3, false, rng)

		for _, room := range graph.Rooms {
			if room.Type == RoomSecret {
				secrets, ok := result.Secrets[room.ID]
				if !ok || len(secrets) == 0 {
					t.Errorf("seed %d: secret room %s has no secrets", seed, room.ID)
					continue
				}
				if secrets[0].SecretType != "hidden_passage" {
					t.Errorf("seed %d: secret type = %q, want %q", seed, secrets[0].SecretType, "hidden_passage")
				}
				if secrets[0].DetectionDC < 10 || secrets[0].DetectionDC > 20 {
					t.Errorf("seed %d: secret DC %d out of range", seed, secrets[0].DetectionDC)
				}
			}
		}
	}
}

func TestPopulateRooms_SecretRoomRevealLowersDC(t *testing.T) {
	// Run with secretRoomReveal=true and compare DCs
	revealDCs := make([]int, 0)
	normalDCs := make([]int, 0)

	for seed := int64(0); seed < 50; seed++ {
		rng1 := rand.New(rand.NewSource(seed))
		g1 := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeLong}, rng1)
		AssignRoomTypes(g1, rng1, SizeLong, true)
		r1 := PopulateRooms(g1, testTheme(), 3, true, rng1)

		rng2 := rand.New(rand.NewSource(seed))
		g2 := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeLong}, rng2)
		AssignRoomTypes(g2, rng2, SizeLong, false)
		r2 := PopulateRooms(g2, testTheme(), 3, false, rng2)

		for id, secrets := range r1.Secrets {
			revealDCs = append(revealDCs, secrets[0].DetectionDC)
			if normalSecrets, ok := r2.Secrets[id]; ok {
				normalDCs = append(normalDCs, normalSecrets[0].DetectionDC)
			}
		}
	}

	if len(revealDCs) == 0 {
		t.Skip("no secret rooms found across seeds")
	}

	// Average reveal DC should be lower
	avgReveal := average(revealDCs)
	avgNormal := average(normalDCs)
	if avgReveal >= avgNormal && len(normalDCs) > 0 {
		t.Errorf("reveal avg DC %.1f should be lower than normal avg DC %.1f", avgReveal, avgNormal)
	}
}

func TestPopulateRooms_NarrativesAssigned(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(graph, rng, SizeMedium, false)

	theme := testTheme()
	result := PopulateRooms(graph, theme, 3, false, rng)

	narrativeCount := 0
	for _, room := range graph.Rooms {
		if narr, ok := result.Narratives[room.ID]; ok {
			if narr.En == "" {
				t.Errorf("room %s: empty English narrative", room.ID)
			}
			narrativeCount++
		}
	}

	if narrativeCount == 0 {
		t.Error("no narratives assigned to any room")
	}
}

func TestPopulateRooms_NonCombatNoTraps(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(graph, rng, SizeMedium, false)

	result := PopulateRooms(graph, testTheme(), 3, false, rng)

	for _, room := range graph.Rooms {
		if room.Type != RoomTrap {
			if _, ok := result.Traps[room.ID]; ok {
				t.Errorf("non-trap room %s (type=%q) should not have traps", room.ID, room.Type)
			}
		}
	}
}

func TestPopulateRooms_Determinism(t *testing.T) {
	theme := testTheme()

	rng1 := rand.New(rand.NewSource(99))
	g1 := GenerateGraph(DungeonConfig{Seed: 99, Size: SizeShort}, rng1)
	AssignRoomTypes(g1, rng1, SizeShort, false)
	r1 := PopulateRooms(g1, theme, 3, false, rng1)

	rng2 := rand.New(rand.NewSource(99))
	g2 := GenerateGraph(DungeonConfig{Seed: 99, Size: SizeShort}, rng2)
	AssignRoomTypes(g2, rng2, SizeShort, false)
	r2 := PopulateRooms(g2, theme, 3, false, rng2)

	if len(r1.Loot) != len(r2.Loot) {
		t.Errorf("loot count differs: %d vs %d", len(r1.Loot), len(r2.Loot))
	}
	if len(r1.Traps) != len(r2.Traps) {
		t.Errorf("traps count differs: %d vs %d", len(r1.Traps), len(r2.Traps))
	}
	if len(r1.Narratives) != len(r2.Narratives) {
		t.Errorf("narratives count differs: %d vs %d", len(r1.Narratives), len(r2.Narratives))
	}
}

func TestTrapDamageByTier(t *testing.T) {
	// Tier 0 (levels 1-4)
	if d := trapDamage[TrapPit][0]; d != "1d6" {
		t.Errorf("pit tier 0: got %q, want %q", d, "1d6")
	}
	// Tier 1 (levels 5+)
	if d := trapDamage[TrapPit][1]; d != "2d6" {
		t.Errorf("pit tier 1: got %q, want %q", d, "2d6")
	}
	// Alarm has no damage
	if d := trapDamage[TrapAlarm][0]; d != "" {
		t.Errorf("alarm tier 0: got %q, want empty", d)
	}
}

func TestRandomInteriorCell(t *testing.T) {
	bounds := RoomBounds{Rows: 6, Cols: 6}
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 100; i++ {
		pos := randomInteriorCell(bounds, rng)
		if pos[0] < 1 || pos[0] >= bounds.Rows-1 {
			t.Errorf("row %d outside interior [1, %d)", pos[0], bounds.Rows-1)
		}
		if pos[1] < 1 || pos[1] >= bounds.Cols-1 {
			t.Errorf("col %d outside interior [1, %d)", pos[1], bounds.Cols-1)
		}
	}
}

func TestRandomWallCell(t *testing.T) {
	bounds := RoomBounds{Rows: 6, Cols: 6}
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 100; i++ {
		pos := randomWallCell(bounds, rng)
		onWall := pos[0] == 0 || pos[0] == bounds.Rows-1 || pos[1] == 0 || pos[1] == bounds.Cols-1
		if !onWall {
			t.Errorf("position (%d,%d) not on wall", pos[0], pos[1])
		}
	}
}

// helper
func average(vals []int) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0
	for _, v := range vals {
		sum += v
	}
	return float64(sum) / float64(len(vals))
}
