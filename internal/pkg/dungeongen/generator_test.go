package dungeongen

import (
	"encoding/json"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func makeTestInput() *GenerationInput {
	return &GenerationInput{
		Config: DungeonConfig{
			Seed:       42,
			Size:       SizeShort,
			PartyLevel: 3,
			PartySize:  4,
		},
		Difficulty:       DifficultyMedium,
		Theme:            DefaultThemes["catacombs"],
		ThemeTags:        []string{"dungeon"},
		SecretRoomReveal: false,
		TileMetadata:     makeMockMetadataForGen(),
		TileWalkability:  makeMockWalkability(),
		Creatures:        makeMockCreatures(),
	}
}

// makeMockMetadataForGen creates minimal tile metadata for the generator.
func makeMockMetadataForGen() []*models.TileMetadata {
	sym := "011110"
	allSym := models.EdgeSignatures{
		Top: sym, Right: sym, Bottom: sym, Left: sym,
		R1Top: sym, R1Right: sym, R1Bottom: sym, R1Left: sym,
		R2Top: sym, R2Right: sym, R2Bottom: sym, R2Left: sym,
		R3Top: sym, R3Right: sym, R3Bottom: sym, R3Left: sym,
	}
	return []*models.TileMetadata{
		{
			TileID:         "tile_room",
			Role:           models.TileRoleRoom,
			ThemeTags:      []string{"dungeon"},
			Openings:       models.OpeningSummary{Top: true, Right: true, Bottom: true, Left: true},
			EdgeSignatures: allSym,
		},
		{
			TileID:         "tile_corridor",
			Role:           models.TileRoleCorridorH,
			ThemeTags:      []string{"dungeon"},
			Openings:       models.OpeningSummary{Top: true, Right: true, Bottom: true, Left: true},
			EdgeSignatures: allSym,
		},
	}
}

// makeMockWalkability creates walkability data for mock tiles.
func makeMockWalkability() map[string]*models.TileWalkability {
	walk := make([][]int, TileSize)
	occl := make([][]int, TileSize)
	for r := 0; r < TileSize; r++ {
		walk[r] = make([]int, TileSize)
		occl[r] = make([]int, TileSize)
		for c := 0; c < TileSize; c++ {
			if r >= 1 && r < TileSize-1 && c >= 1 && c < TileSize-1 {
				walk[r][c] = 1
			}
		}
	}

	return map[string]*models.TileWalkability{
		"tile_room": {
			TileID:      "tile_room",
			Rows:        TileSize,
			Cols:        TileSize,
			Walkability: walk,
			Occlusion:   occl,
		},
		"tile_corridor": {
			TileID:      "tile_corridor",
			Rows:        TileSize,
			Cols:        TileSize,
			Walkability: walk,
			Occlusion:   occl,
		},
	}
}

func TestGenerateDungeon_ReturnsNonNil(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())
	if resp == nil {
		t.Fatal("GenerateDungeon returned nil")
	}
}

func TestGenerateDungeon_SeedAndSize(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())

	if resp.Seed != 42 {
		t.Errorf("Seed = %d, want 42", resp.Seed)
	}
	if resp.Size != SizeShort {
		t.Errorf("Size = %q, want %q", resp.Size, SizeShort)
	}
	if resp.ThemeName != "catacombs" {
		t.Errorf("ThemeName = %q, want %q", resp.ThemeName, "catacombs")
	}
}

func TestGenerateDungeon_HasRoomsAndConnections(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())

	if len(resp.Rooms) == 0 {
		t.Error("no rooms in response")
	}
	if len(resp.Connections) == 0 {
		t.Error("no connections in response")
	}
	if resp.MainPathLength <= 0 {
		t.Errorf("MainPathLength = %d, want > 0", resp.MainPathLength)
	}
}

func TestGenerateDungeon_HasExtractionPoints(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())

	if len(resp.ExtractionPoints) == 0 {
		t.Error("no extraction points")
	}
}

func TestGenerateDungeon_HasComposition(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())

	if resp.Composition == nil {
		t.Fatal("Composition is nil")
	}
	if resp.Composition.Rows <= 0 || resp.Composition.Cols <= 0 {
		t.Errorf("Composition dimensions %dx%d invalid", resp.Composition.Rows, resp.Composition.Cols)
	}
	if len(resp.Composition.Placements) == 0 {
		t.Error("no tile placements")
	}
}

func TestGenerateDungeon_HasTerrain(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())

	if resp.Terrain == nil {
		t.Fatal("Terrain is nil")
	}
	if resp.Terrain.Rows != resp.Composition.Rows || resp.Terrain.Cols != resp.Composition.Cols {
		t.Errorf("Terrain dimensions %dx%d != Composition %dx%d",
			resp.Terrain.Rows, resp.Terrain.Cols,
			resp.Composition.Rows, resp.Composition.Cols)
	}
	if len(resp.Terrain.Walkability) != resp.Terrain.Rows {
		t.Errorf("Walkability rows = %d, want %d", len(resp.Terrain.Walkability), resp.Terrain.Rows)
	}
}

func TestGenerateDungeon_HasEncounters(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())

	if len(resp.Encounters) == 0 {
		t.Error("no encounters assigned")
	}

	// All encounters should be for combat-type rooms
	roomType := make(map[string]RoomType)
	for _, r := range resp.Rooms {
		roomType[r.ID] = r.Type
	}
	for roomID := range resp.Encounters {
		rt := roomType[roomID]
		if rt != RoomCombat && rt != RoomCombatOptional && rt != RoomBoss {
			t.Errorf("encounter in non-combat room %s (type=%q)", roomID, rt)
		}
	}
}

func TestGenerateDungeon_HasPopulation(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())

	// Should have loot for treasure and/or boss rooms
	hasTreasure := false
	for _, r := range resp.Rooms {
		if r.Type == RoomTreasure || r.Type == RoomBoss {
			if _, ok := resp.Loot[r.ID]; ok {
				hasTreasure = true
			}
		}
	}
	if !hasTreasure {
		t.Error("no loot found for treasure/boss rooms")
	}

	// Should have narratives
	if len(resp.Narratives) == 0 {
		t.Error("no narratives assigned")
	}
}

func TestGenerateDungeon_Determinism(t *testing.T) {
	r1 := GenerateDungeon(makeTestInput())
	r2 := GenerateDungeon(makeTestInput())

	if len(r1.Rooms) != len(r2.Rooms) {
		t.Fatalf("room count: %d vs %d", len(r1.Rooms), len(r2.Rooms))
	}
	if len(r1.Connections) != len(r2.Connections) {
		t.Errorf("connection count: %d vs %d", len(r1.Connections), len(r2.Connections))
	}
	if len(r1.Encounters) != len(r2.Encounters) {
		t.Errorf("encounter count: %d vs %d", len(r1.Encounters), len(r2.Encounters))
	}
	if r1.Composition.Rows != r2.Composition.Rows || r1.Composition.Cols != r2.Composition.Cols {
		t.Errorf("composition dimensions differ: %dx%d vs %dx%d",
			r1.Composition.Rows, r1.Composition.Cols,
			r2.Composition.Rows, r2.Composition.Cols)
	}
	if r1.Terrain.Rows != r2.Terrain.Rows || r1.Terrain.Cols != r2.Terrain.Cols {
		t.Errorf("terrain dimensions differ: %dx%d vs %dx%d",
			r1.Terrain.Rows, r1.Terrain.Cols,
			r2.Terrain.Rows, r2.Terrain.Cols)
	}
}

func TestGenerateDungeon_JSONSerializable(t *testing.T) {
	resp := GenerateDungeon(makeTestInput())

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("JSON output is empty")
	}

	// Verify round-trip
	var decoded DungeonResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if decoded.Seed != resp.Seed {
		t.Errorf("decoded Seed = %d, want %d", decoded.Seed, resp.Seed)
	}
	if len(decoded.Rooms) != len(resp.Rooms) {
		t.Errorf("decoded rooms = %d, want %d", len(decoded.Rooms), len(resp.Rooms))
	}
}

func TestGenerateDungeon_NilTheme(t *testing.T) {
	input := makeTestInput()
	input.Theme = nil

	resp := GenerateDungeon(input)

	if resp == nil {
		t.Fatal("GenerateDungeon returned nil with nil theme")
	}
	if resp.ThemeName != "" {
		t.Errorf("ThemeName = %q, want empty", resp.ThemeName)
	}
}

func TestGenerateDungeon_EmptyCreatures(t *testing.T) {
	input := makeTestInput()
	input.Creatures = nil

	resp := GenerateDungeon(input)

	if resp == nil {
		t.Fatal("GenerateDungeon returned nil with empty creatures")
	}
	// Encounters should still exist but with no monsters
	for roomID, enc := range resp.Encounters {
		if len(enc.Monsters) != 0 {
			t.Errorf("room %s: expected no monsters with empty creature pool, got %d", roomID, len(enc.Monsters))
		}
	}
}

func TestGenerateDungeon_AllSizes(t *testing.T) {
	for _, size := range []DungeonSize{SizeShort, SizeMedium, SizeLong} {
		t.Run(string(size), func(t *testing.T) {
			input := makeTestInput()
			input.Config.Size = size

			resp := GenerateDungeon(input)

			if resp == nil {
				t.Fatal("nil response")
			}
			if resp.Size != size {
				t.Errorf("Size = %q, want %q", resp.Size, size)
			}
			if len(resp.Rooms) == 0 {
				t.Error("no rooms")
			}
		})
	}
}

func TestGenerateDungeon_AllDifficulties(t *testing.T) {
	for _, diff := range []DungeonDifficulty{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyDeadly} {
		t.Run(string(diff), func(t *testing.T) {
			input := makeTestInput()
			input.Difficulty = diff

			resp := GenerateDungeon(input)

			if resp == nil {
				t.Fatal("nil response")
			}
			if len(resp.Encounters) == 0 {
				t.Error("no encounters")
			}
		})
	}
}
