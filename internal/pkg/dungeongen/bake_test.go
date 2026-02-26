package dungeongen

import (
	"math/rand"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// makeMockTileData creates a simple tile walkability for testing.
// Walkability: center 4×4 is walkable (1), outer ring is blocked (0).
// Occlusion: inverse of walkability.
func makeMockTileData(tileID string) *models.TileWalkability {
	walk := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 0, 0, 0, 0, 0},
	}
	occl := [][]int{
		{1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 1},
		{1, 1, 1, 1, 1, 1},
	}
	return &models.TileWalkability{
		TileID:      tileID,
		Rows:        6,
		Cols:        6,
		Walkability: walk,
		Occlusion:   occl,
	}
}

func TestBakeTerrain_SingleTile(t *testing.T) {
	tile := makeMockTileData("tile_a")
	tileData := map[string]*models.TileWalkability{"tile_a": tile}

	comp := &MapComposition{
		Rows: 6,
		Cols: 6,
		Placements: []MacroTilePlacement{
			{TileID: "tile_a", NodeID: "room_0", OriginRow: 0, OriginCol: 0, Rotation: 0},
		},
	}

	baked := BakeTerrain(comp, tileData)

	if baked.Rows != 6 || baked.Cols != 6 {
		t.Errorf("dimensions: %dx%d, want 6x6", baked.Rows, baked.Cols)
	}

	// Center should be walkable
	if baked.Walkability[2][2] != 1 {
		t.Error("center walkability should be 1")
	}
	// Corner should be blocked
	if baked.Walkability[0][0] != 0 {
		t.Error("corner walkability should be 0")
	}
	// Occlusion: center transparent, corner opaque
	if baked.Occlusion[2][2] != 0 {
		t.Error("center occlusion should be 0")
	}
	if baked.Occlusion[0][0] != 1 {
		t.Error("corner occlusion should be 1")
	}
}

func TestBakeTerrain_RotatedTile(t *testing.T) {
	// Asymmetric tile: only top-left cell is walkable
	tile := &models.TileWalkability{
		TileID: "asym",
		Rows:   6, Cols: 6,
		Walkability: [][]int{
			{1, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0},
		},
		Occlusion: makeGrid(6, 6),
	}
	tileData := map[string]*models.TileWalkability{"asym": tile}

	// Rotation 1 (90° CCW): (0,0) → (0,5)
	comp := &MapComposition{
		Rows: 6, Cols: 6,
		Placements: []MacroTilePlacement{
			{TileID: "asym", NodeID: "r0", OriginRow: 0, OriginCol: 0, Rotation: 1},
		},
	}

	baked := BakeTerrain(comp, tileData)

	// After 90° CCW, (0,0) maps to (0,5)
	if baked.Walkability[0][5] != 1 {
		t.Error("rotated: (0,5) should be 1")
	}
	if baked.Walkability[0][0] != 0 {
		t.Error("rotated: (0,0) should be 0")
	}
}

func TestBakeTerrain_TwoTilesHorizontal(t *testing.T) {
	tileA := makeMockTileData("tile_a")
	tileB := makeMockTileData("tile_b")
	tileData := map[string]*models.TileWalkability{
		"tile_a": tileA,
		"tile_b": tileB,
	}

	comp := &MapComposition{
		Rows: 6, Cols: 12,
		Placements: []MacroTilePlacement{
			{TileID: "tile_a", NodeID: "room_0", OriginRow: 0, OriginCol: 0, Rotation: 0},
			{TileID: "tile_b", NodeID: "room_1", OriginRow: 0, OriginCol: 6, Rotation: 0},
		},
	}

	baked := BakeTerrain(comp, tileData)

	if baked.Rows != 6 || baked.Cols != 12 {
		t.Errorf("dimensions: %dx%d, want 6x12", baked.Rows, baked.Cols)
	}

	// Both tiles have center walkable
	if baked.Walkability[2][2] != 1 {
		t.Error("tile_a center (2,2) should be walkable")
	}
	if baked.Walkability[2][8] != 1 {
		t.Error("tile_b center (2,8) should be walkable")
	}

	// Boundary between tiles: tile_a col 5 and tile_b col 6 are both edge cells
	if baked.Walkability[2][5] != 0 {
		t.Error("tile_a edge col (2,5) should be blocked")
	}
	if baked.Walkability[2][6] != 0 {
		t.Error("tile_b edge col (2,6) should be blocked")
	}
}

func TestBakeTerrain_DefaultCells(t *testing.T) {
	// Tile placed in center of a larger grid
	tile := makeMockTileData("tile_a")
	tileData := map[string]*models.TileWalkability{"tile_a": tile}

	comp := &MapComposition{
		Rows: 12, Cols: 12,
		Placements: []MacroTilePlacement{
			{TileID: "tile_a", NodeID: "r0", OriginRow: 3, OriginCol: 3, Rotation: 0},
		},
	}

	baked := BakeTerrain(comp, tileData)

	// Uncovered cells default to blocked walkability (0) and opaque occlusion (1)
	if baked.Walkability[0][0] != 0 {
		t.Error("uncovered cell walkability should be 0")
	}
	if baked.Occlusion[0][0] != 1 {
		t.Error("uncovered cell occlusion should be 1")
	}
}

func TestBakeTerrain_EdgesCollected(t *testing.T) {
	tile := &models.TileWalkability{
		TileID:      "edgy",
		Rows:        6,
		Cols:        6,
		Walkability: makeGrid(6, 6),
		Occlusion:   makeGrid(6, 6),
		Edges: []models.SerializedEdge{
			{Key: "1,1-1,2", MoveBlock: true, LosBlock: false},
		},
	}
	tileData := map[string]*models.TileWalkability{"edgy": tile}

	comp := &MapComposition{
		Rows: 6, Cols: 6,
		Placements: []MacroTilePlacement{
			{TileID: "edgy", NodeID: "r0", OriginRow: 0, OriginCol: 0, Rotation: 0},
		},
	}

	baked := BakeTerrain(comp, tileData)

	if len(baked.Edges) != 1 {
		t.Fatalf("edges: got %d, want 1", len(baked.Edges))
	}
	if baked.Edges[0].Key != "1,1-1,2" {
		t.Errorf("edge key: got %q, want %q", baked.Edges[0].Key, "1,1-1,2")
	}
	if !baked.Edges[0].MoveBlock || baked.Edges[0].LosBlock {
		t.Errorf("edge props: moveBlock=%v losBlock=%v", baked.Edges[0].MoveBlock, baked.Edges[0].LosBlock)
	}
}

func TestBakeTerrain_EdgesTranslated(t *testing.T) {
	tile := &models.TileWalkability{
		TileID:      "edgy",
		Rows:        6,
		Cols:        6,
		Walkability: makeGrid(6, 6),
		Occlusion:   makeGrid(6, 6),
		Edges: []models.SerializedEdge{
			{Key: "0,0-0,1", MoveBlock: true, LosBlock: true},
		},
	}
	tileData := map[string]*models.TileWalkability{"edgy": tile}

	// Place at origin (6, 12)
	comp := &MapComposition{
		Rows: 12, Cols: 18,
		Placements: []MacroTilePlacement{
			{TileID: "edgy", NodeID: "r0", OriginRow: 6, OriginCol: 12, Rotation: 0},
		},
	}

	baked := BakeTerrain(comp, tileData)

	if len(baked.Edges) != 1 {
		t.Fatalf("edges: got %d, want 1", len(baked.Edges))
	}
	// Original "0,0-0,1" translated by (+6,+12) = "6,12-6,13"
	if baked.Edges[0].Key != "6,12-6,13" {
		t.Errorf("translated edge key: got %q, want %q", baked.Edges[0].Key, "6,12-6,13")
	}
}

func TestBakeTerrain_EdgesRotatedAndTranslated(t *testing.T) {
	tile := &models.TileWalkability{
		TileID:      "edgy",
		Rows:        6,
		Cols:        6,
		Walkability: makeGrid(6, 6),
		Occlusion:   makeGrid(6, 6),
		Edges: []models.SerializedEdge{
			{Key: "0,0-0,1", MoveBlock: true, LosBlock: false},
		},
	}
	tileData := map[string]*models.TileWalkability{"edgy": tile}

	// Rotation 1 (90° CCW) at origin (0,0)
	comp := &MapComposition{
		Rows: 6, Cols: 6,
		Placements: []MacroTilePlacement{
			{TileID: "edgy", NodeID: "r0", OriginRow: 0, OriginCol: 0, Rotation: 1},
		},
	}

	baked := BakeTerrain(comp, tileData)

	if len(baked.Edges) != 1 {
		t.Fatalf("edges: got %d, want 1", len(baked.Edges))
	}
	// "0,0-0,1" rotated 90° CCW in 6×6: (0,0)→(0,5), (0,1)→(1,5) → "0,5-1,5"
	if baked.Edges[0].Key != "0,5-1,5" {
		t.Errorf("rotated edge key: got %q, want %q", baked.Edges[0].Key, "0,5-1,5")
	}
}

func TestBakeTerrain_EdgesMergedBlockWins(t *testing.T) {
	// Test merge: a tile with duplicate edge keys that differ in properties
	tileM := &models.TileWalkability{
		TileID: "tm", Rows: 6, Cols: 6,
		Walkability: makeGrid(6, 6),
		Occlusion:   makeGrid(6, 6),
		Edges: []models.SerializedEdge{
			{Key: "2,5-3,5", MoveBlock: true, LosBlock: false},
			{Key: "2,5-3,5", MoveBlock: false, LosBlock: true}, // duplicate in same tile
		},
	}
	tileDataM := map[string]*models.TileWalkability{"tm": tileM}

	compM := &MapComposition{
		Rows: 6, Cols: 6,
		Placements: []MacroTilePlacement{
			{TileID: "tm", NodeID: "r0", OriginRow: 0, OriginCol: 0, Rotation: 0},
		},
	}

	baked := BakeTerrain(compM, tileDataM)

	if len(baked.Edges) != 1 {
		t.Fatalf("merged edges: got %d, want 1", len(baked.Edges))
	}
	// Block-wins: moveBlock = true || false = true, losBlock = false || true = true
	if !baked.Edges[0].MoveBlock || !baked.Edges[0].LosBlock {
		t.Errorf("merged edge: moveBlock=%v losBlock=%v, want both true",
			baked.Edges[0].MoveBlock, baked.Edges[0].LosBlock)
	}
}

func TestBakeTerrain_IntegrationWithLayout(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeShort}, rng)
	AssignRoomTypes(graph, rng, SizeShort, false)
	assignments := makeDummyAssignments(graph)
	comp := ComputeLayout(graph, assignments)

	// Create tile data for dummy tiles
	tileData := map[string]*models.TileWalkability{
		"tile_room":     makeMockTileData("tile_room"),
		"tile_corridor": makeMockTileData("tile_corridor"),
	}

	baked := BakeTerrain(comp, tileData)

	if baked.Rows != comp.Rows {
		t.Errorf("rows: got %d, want %d", baked.Rows, comp.Rows)
	}
	if baked.Cols != comp.Cols {
		t.Errorf("cols: got %d, want %d", baked.Cols, comp.Cols)
	}
	if len(baked.Walkability) != comp.Rows {
		t.Errorf("walkability rows: got %d, want %d", len(baked.Walkability), comp.Rows)
	}
	if len(baked.Walkability[0]) != comp.Cols {
		t.Errorf("walkability cols: got %d, want %d", len(baked.Walkability[0]), comp.Cols)
	}

	// Verify some cells are walkable (tile centers) and some are blocked (tile edges)
	hasWalkable := false
	hasBlocked := false
	for r := range baked.Walkability {
		for c := range baked.Walkability[r] {
			if baked.Walkability[r][c] == 1 {
				hasWalkable = true
			}
			if baked.Walkability[r][c] == 0 {
				hasBlocked = true
			}
		}
	}
	if !hasWalkable {
		t.Error("no walkable cells in baked terrain")
	}
	if !hasBlocked {
		t.Error("no blocked cells in baked terrain")
	}
}

func TestBakeTerrain_MissingTileSkipped(t *testing.T) {
	tileData := map[string]*models.TileWalkability{} // empty

	comp := &MapComposition{
		Rows: 6, Cols: 6,
		Placements: []MacroTilePlacement{
			{TileID: "nonexistent", NodeID: "r0", OriginRow: 0, OriginCol: 0, Rotation: 0},
		},
	}

	baked := BakeTerrain(comp, tileData)

	// All cells should remain default (blocked/opaque)
	for r := range baked.Walkability {
		for c := range baked.Walkability[r] {
			if baked.Walkability[r][c] != 0 {
				t.Errorf("cell (%d,%d) walkability = %d, want 0 (default)", r, c, baked.Walkability[r][c])
			}
		}
	}
}

// --- translateEdgeKey ---

func TestTranslateEdgeKey(t *testing.T) {
	tests := []struct {
		key       string
		row, col  int
		want      string
	}{
		{"0,0-0,1", 0, 0, "0,0-0,1"},
		{"0,0-0,1", 6, 12, "6,12-6,13"},
		{"2,3-3,3", 0, 6, "2,9-3,9"},
		{"0,0-1,0", 10, 10, "10,10-11,10"},
	}

	for _, tt := range tests {
		got := translateEdgeKey(tt.key, tt.row, tt.col)
		if got != tt.want {
			t.Errorf("translateEdgeKey(%q, %d, %d) = %q, want %q",
				tt.key, tt.row, tt.col, got, tt.want)
		}
	}
}

// --- stampTile ---

func TestStampTile(t *testing.T) {
	dst := makeFilledGrid(12, 12, 0)
	src := [][]int{
		{1, 1},
		{1, 1},
	}

	stampTile(dst, src, 3, 5)

	if dst[3][5] != 1 || dst[3][6] != 1 || dst[4][5] != 1 || dst[4][6] != 1 {
		t.Error("stamped cells should be 1")
	}
	// Adjacent cells should remain 0
	if dst[2][5] != 0 || dst[3][4] != 0 {
		t.Error("non-stamped cells should be 0")
	}
}
