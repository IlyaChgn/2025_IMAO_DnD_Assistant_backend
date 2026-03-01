package dungeongen

import (
	"math/rand"
	"testing"
)

func makeMockCreatures() []CreatureSummary {
	return []CreatureSummary{
		{ID: "goblin", Name: "Goblin", CR: "1/4", XP: 50, CreatureType: "humanoid"},
		{ID: "skeleton", Name: "Skeleton", CR: "1/4", XP: 50, CreatureType: "undead"},
		{ID: "zombie", Name: "Zombie", CR: "1/4", XP: 50, CreatureType: "undead"},
		{ID: "wolf", Name: "Wolf", CR: "1/4", XP: 50, CreatureType: "beast"},
		{ID: "orc", Name: "Orc", CR: "1/2", XP: 100, CreatureType: "humanoid"},
		{ID: "bugbear", Name: "Bugbear", CR: "1", XP: 200, CreatureType: "humanoid"},
		{ID: "ogre", Name: "Ogre", CR: "2", XP: 450, CreatureType: "humanoid"},
		{ID: "owlbear", Name: "Owlbear", CR: "3", XP: 700, CreatureType: "monstrosity"},
		{ID: "troll", Name: "Troll", CR: "5", XP: 1800, CreatureType: "giant"},
	}
}

func TestCrToNumeric(t *testing.T) {
	tests := []struct {
		cr   string
		want float64
	}{
		{"0", 0},
		{"1/8", 0.125},
		{"1/4", 0.25},
		{"1/2", 0.5},
		{"1", 1},
		{"5", 5},
		{"10", 10},
	}
	for _, tt := range tests {
		got := crToNumeric(tt.cr)
		if got != tt.want {
			t.Errorf("crToNumeric(%q) = %v, want %v", tt.cr, got, tt.want)
		}
	}
}

func TestEncounterMultiplier(t *testing.T) {
	tests := []struct {
		count int
		want  float64
	}{
		{1, 1.0},
		{2, 1.5},
		{3, 2.0},
		{6, 2.0},
		{7, 2.5},
		{10, 2.5},
		{11, 3.0},
		{14, 3.0},
		{15, 4.0},
	}
	for _, tt := range tests {
		got := encounterMultiplier(tt.count)
		if got != tt.want {
			t.Errorf("encounterMultiplier(%d) = %v, want %v", tt.count, got, tt.want)
		}
	}
}

func TestFilterCreaturesByCR(t *testing.T) {
	creatures := makeMockCreatures()

	// CR 0.25 to 1
	result := filterCreaturesByCR(creatures, 0.25, 1.0)
	for _, c := range result {
		cr := crToNumeric(c.CR)
		if cr < 0.25 || cr > 1.0 {
			t.Errorf("creature %s CR=%s (%v) outside [0.25, 1.0]", c.ID, c.CR, cr)
		}
	}
	if len(result) == 0 {
		t.Error("expected some creatures in CR [0.25, 1.0]")
	}
}

func TestBudgetEncounters_CombatRoomsOnly(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(graph, rng, SizeMedium, false)

	creatures := makeMockCreatures()
	encounters := BudgetEncounters(graph, creatures, 3, 4, DifficultyMedium, rng)

	for roomID, enc := range encounters {
		// Find the room
		var roomType RoomType
		for _, r := range graph.Rooms {
			if r.ID == roomID {
				roomType = r.Type
				break
			}
		}
		if roomType != RoomCombat && roomType != RoomCombatOptional && roomType != RoomBoss {
			t.Errorf("room %s type %q should not have an encounter", roomID, roomType)
		}
		if enc.XPBudget <= 0 {
			t.Errorf("room %s: XP budget %d should be positive", roomID, enc.XPBudget)
		}
		if len(enc.Monsters) == 0 {
			t.Errorf("room %s: no monsters assigned", roomID)
		}
	}
}

func TestBudgetEncounters_BossIsDeadly(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(graph, rng, SizeMedium, false)

	creatures := makeMockCreatures()
	encounters := BudgetEncounters(graph, creatures, 3, 4, DifficultyMedium, rng)

	// Find boss room
	bossID := ""
	for _, r := range graph.Rooms {
		if r.Type == RoomBoss {
			bossID = r.ID
			break
		}
	}

	if bossID == "" {
		t.Fatal("no boss room found")
	}

	enc, ok := encounters[bossID]
	if !ok {
		t.Fatal("boss room has no encounter")
	}
	if enc.RoomDifficulty != DifficultyDeadly {
		t.Errorf("boss difficulty: got %q, want %q", enc.RoomDifficulty, DifficultyDeadly)
	}
}

func TestBudgetEncounters_WithinBudget(t *testing.T) {
	creatures := makeMockCreatures()

	for seed := int64(0); seed < 10; seed++ {
		rng := rand.New(rand.NewSource(seed))
		graph := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeMedium}, rng)
		AssignRoomTypes(graph, rng, SizeMedium, false)

		encounters := BudgetEncounters(graph, creatures, 5, 4, DifficultyMedium, rng)

		for roomID, enc := range encounters {
			totalXP := 0
			totalCount := 0
			for _, m := range enc.Monsters {
				totalXP += m.XP * m.Count
				totalCount += m.Count
			}

			adjusted := adjustedXP(totalXP, totalCount)
			maxAllowed := float64(enc.XPBudget) * 1.2 // Allow some tolerance beyond the 10% overage

			if adjusted > maxAllowed {
				t.Errorf("seed %d room %s: adjusted XP %.0f exceeds budget %d * 1.2 = %.0f",
					seed, roomID, adjusted, enc.XPBudget, maxAllowed)
			}
		}
	}
}

func TestBudgetEncounters_MonstersHavePositions(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeShort}, rng)
	AssignRoomTypes(graph, rng, SizeShort, false)
	// Set room bounds (layout)
	assignments := makeDummyAssignments(graph)
	ComputeLayout(graph, assignments)

	creatures := makeMockCreatures()
	encounters := BudgetEncounters(graph, creatures, 3, 4, DifficultyMedium, rng)

	for _, enc := range encounters {
		for _, m := range enc.Monsters {
			if len(m.Positions) != m.Count {
				t.Errorf("creature %s: %d positions for count %d", m.CreatureID, len(m.Positions), m.Count)
			}
		}
	}
}

func TestBudgetEncounters_EmptyCreatures(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeShort}, rng)
	AssignRoomTypes(graph, rng, SizeShort, false)

	// Empty creature pool
	encounters := BudgetEncounters(graph, []CreatureSummary{}, 3, 4, DifficultyMedium, rng)

	// Should still produce encounters (with empty monster lists)
	for _, enc := range encounters {
		if len(enc.Monsters) != 0 {
			t.Error("expected no monsters with empty creature pool")
		}
	}
}

func TestBudgetEncounters_Determinism(t *testing.T) {
	creatures := makeMockCreatures()

	rng1 := rand.New(rand.NewSource(99))
	g1 := GenerateGraph(DungeonConfig{Seed: 99, Size: SizeShort}, rng1)
	AssignRoomTypes(g1, rng1, SizeShort, false)
	e1 := BudgetEncounters(g1, creatures, 3, 4, DifficultyMedium, rng1)

	rng2 := rand.New(rand.NewSource(99))
	g2 := GenerateGraph(DungeonConfig{Seed: 99, Size: SizeShort}, rng2)
	AssignRoomTypes(g2, rng2, SizeShort, false)
	e2 := BudgetEncounters(g2, creatures, 3, 4, DifficultyMedium, rng2)

	if len(e1) != len(e2) {
		t.Fatalf("encounter count differs: %d vs %d", len(e1), len(e2))
	}

	for id, enc1 := range e1 {
		enc2, ok := e2[id]
		if !ok {
			t.Errorf("room %s in e1 but not e2", id)
			continue
		}
		if enc1.XPBudget != enc2.XPBudget {
			t.Errorf("room %s: budget %d vs %d", id, enc1.XPBudget, enc2.XPBudget)
		}
		if len(enc1.Monsters) != len(enc2.Monsters) {
			t.Errorf("room %s: monster count %d vs %d", id, len(enc1.Monsters), len(enc2.Monsters))
		}
	}
}

func TestGreedySelectMonsters_NonEmpty(t *testing.T) {
	creatures := makeMockCreatures()
	rng := rand.New(rand.NewSource(42))

	// Budget = 400 XP (4 characters × 100 easy)
	spawns := greedySelectMonsters(creatures, 400, rng)

	if len(spawns) == 0 {
		t.Error("expected at least one monster spawn")
	}

	totalXP := 0
	totalCount := 0
	for _, s := range spawns {
		totalXP += s.XP * s.Count
		totalCount += s.Count
	}

	if totalXP == 0 {
		t.Error("total XP should be > 0")
	}
}
