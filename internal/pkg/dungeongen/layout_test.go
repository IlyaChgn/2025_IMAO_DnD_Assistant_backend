package dungeongen

import (
	"math/rand"
	"testing"
)

// makeLinearGraph creates a simple 3-room linear graph with 2 connections.
func makeLinearGraph() *DungeonGraph {
	return &DungeonGraph{
		Rooms: []DungeonRoom{
			{ID: "room_0", GraphPosition: GraphPosition{X: 0, Y: 0}},
			{ID: "room_1", GraphPosition: GraphPosition{X: 1, Y: 0}},
			{ID: "room_2", GraphPosition: GraphPosition{X: 2, Y: 0}},
		},
		Connections: []RoomConnection{
			{ID: "conn_0_1", FromRoomID: "room_0", ToRoomID: "room_1"},
			{ID: "conn_1_2", FromRoomID: "room_1", ToRoomID: "room_2"},
		},
		MainPathLength: 3,
	}
}

// makeDummyAssignments creates TileAssignment for every room and connection.
func makeDummyAssignments(graph *DungeonGraph) []TileAssignment {
	var assignments []TileAssignment
	for _, r := range graph.Rooms {
		assignments = append(assignments, TileAssignment{
			NodeID: r.ID, TileID: "tile_room", Rotation: 0,
		})
	}
	for _, c := range graph.Connections {
		assignments = append(assignments, TileAssignment{
			NodeID: c.ID, TileID: "tile_corridor", Rotation: 0,
		})
	}
	return assignments
}

func TestComputeLayout_LinearDungeon(t *testing.T) {
	graph := makeLinearGraph()
	assignments := makeDummyAssignments(graph)
	comp := ComputeLayout(graph, assignments)

	// Room positions
	expectBounds(t, "room_0", graph.Rooms[0].Bounds, 0, 0, 6, 6)
	expectBounds(t, "room_1", graph.Rooms[1].Bounds, 0, 12, 6, 6)
	expectBounds(t, "room_2", graph.Rooms[2].Bounds, 0, 24, 6, 6)

	// Corridor positions
	expectBounds(t, "conn_0_1", graph.Connections[0].CorridorBounds, 0, 6, 6, 6)
	expectBounds(t, "conn_1_2", graph.Connections[1].CorridorBounds, 0, 18, 6, 6)

	// Total dimensions: 3 rooms + 2 corridors = 5 tiles wide, 1 tile tall
	if comp.Rows != 6 {
		t.Errorf("Rows: got %d, want 6", comp.Rows)
	}
	if comp.Cols != 30 {
		t.Errorf("Cols: got %d, want 30", comp.Cols)
	}
}

func TestComputeLayout_WithBranchAbove(t *testing.T) {
	graph := &DungeonGraph{
		Rooms: []DungeonRoom{
			{ID: "room_0", GraphPosition: GraphPosition{X: 0, Y: 0}},
			{ID: "room_1", GraphPosition: GraphPosition{X: 1, Y: 0}},
			{ID: "room_2", GraphPosition: GraphPosition{X: 2, Y: 0}},
			{ID: "room_3", GraphPosition: GraphPosition{X: 1, Y: 1}}, // branch above room_1
		},
		Connections: []RoomConnection{
			{ID: "conn_0_1", FromRoomID: "room_0", ToRoomID: "room_1"},
			{ID: "conn_1_2", FromRoomID: "room_1", ToRoomID: "room_2"},
			{ID: "conn_1_3", FromRoomID: "room_1", ToRoomID: "room_3"},
		},
		MainPathLength: 3,
	}
	assignments := makeDummyAssignments(graph)
	ComputeLayout(graph, assignments)

	// After normalization, the branch room is at the top (row 0),
	// corridor at row 6, main path at row 12
	expectBounds(t, "room_3 (branch)", graph.Rooms[3].Bounds, 0, 12, 6, 6)
	expectBounds(t, "conn_1_3 (branch corridor)", graph.Connections[2].CorridorBounds, 6, 12, 6, 6)
	expectBounds(t, "room_1 (main path)", graph.Rooms[1].Bounds, 12, 12, 6, 6)
}

func TestComputeLayout_WithBranchBelow(t *testing.T) {
	graph := &DungeonGraph{
		Rooms: []DungeonRoom{
			{ID: "room_0", GraphPosition: GraphPosition{X: 0, Y: 0}},
			{ID: "room_1", GraphPosition: GraphPosition{X: 1, Y: 0}},
			{ID: "room_2", GraphPosition: GraphPosition{X: 2, Y: 0}},
			{ID: "room_3", GraphPosition: GraphPosition{X: 1, Y: -1}}, // branch below room_1
		},
		Connections: []RoomConnection{
			{ID: "conn_0_1", FromRoomID: "room_0", ToRoomID: "room_1"},
			{ID: "conn_1_2", FromRoomID: "room_1", ToRoomID: "room_2"},
			{ID: "conn_1_3", FromRoomID: "room_1", ToRoomID: "room_3"},
		},
		MainPathLength: 3,
	}
	assignments := makeDummyAssignments(graph)
	ComputeLayout(graph, assignments)

	// Main path at row 0, corridor at row 6, branch at row 12
	expectBounds(t, "room_1 (main path)", graph.Rooms[1].Bounds, 0, 12, 6, 6)
	expectBounds(t, "conn_1_3 (branch corridor)", graph.Connections[2].CorridorBounds, 6, 12, 6, 6)
	expectBounds(t, "room_3 (branch)", graph.Rooms[3].Bounds, 12, 12, 6, 6)
}

func TestComputeLayout_AllBoundsNonZero(t *testing.T) {
	for seed := int64(0); seed < 10; seed++ {
		rng := rand.New(rand.NewSource(seed))
		graph := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeMedium}, rng)
		AssignRoomTypes(graph, rng, SizeMedium, false)
		assignments := makeDummyAssignments(graph)
		ComputeLayout(graph, assignments)

		for _, r := range graph.Rooms {
			if r.Bounds.Rows != TileSize || r.Bounds.Cols != TileSize {
				t.Errorf("seed %d: room %s has bounds %dx%d, want %dx%d",
					seed, r.ID, r.Bounds.Rows, r.Bounds.Cols, TileSize, TileSize)
			}
		}
		for _, c := range graph.Connections {
			if c.CorridorBounds.Rows != TileSize || c.CorridorBounds.Cols != TileSize {
				t.Errorf("seed %d: connection %s has corridor bounds %dx%d, want %dx%d",
					seed, c.ID, c.CorridorBounds.Rows, c.CorridorBounds.Cols, TileSize, TileSize)
			}
		}
	}
}

func TestComputeLayout_NoOverlaps(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		graph := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeLong}, rng)
		AssignRoomTypes(graph, rng, SizeLong, false)
		assignments := makeDummyAssignments(graph)
		comp := ComputeLayout(graph, assignments)

		// Check all placement pairs for overlap
		for i := 0; i < len(comp.Placements); i++ {
			for j := i + 1; j < len(comp.Placements); j++ {
				a := comp.Placements[i]
				b := comp.Placements[j]
				if boundsOverlap(a.OriginRow, a.OriginCol, b.OriginRow, b.OriginCol) {
					t.Errorf("seed %d: overlap between %s (%d,%d) and %s (%d,%d)",
						seed, a.NodeID, a.OriginRow, a.OriginCol,
						b.NodeID, b.OriginRow, b.OriginCol)
				}
			}
		}
	}
}

func TestComputeLayout_CorridorsAdjacentToRooms(t *testing.T) {
	for seed := int64(0); seed < 10; seed++ {
		rng := rand.New(rand.NewSource(seed))
		graph := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeMedium}, rng)
		AssignRoomTypes(graph, rng, SizeMedium, false)
		assignments := makeDummyAssignments(graph)
		ComputeLayout(graph, assignments)

		roomByID := buildRoomIndex(graph)
		for _, conn := range graph.Connections {
			from := roomByID[conn.FromRoomID]
			to := roomByID[conn.ToRoomID]
			cb := conn.CorridorBounds

			fromAdj := isAdjacent(from.Bounds, cb)
			toAdj := isAdjacent(to.Bounds, cb)

			if !fromAdj {
				t.Errorf("seed %d: corridor %s not adjacent to from-room %s", seed, conn.ID, conn.FromRoomID)
			}
			if !toAdj {
				t.Errorf("seed %d: corridor %s not adjacent to to-room %s", seed, conn.ID, conn.ToRoomID)
			}
		}
	}
}

func TestComputeLayout_Normalization(t *testing.T) {
	// Graph with a branch above — forces negative rows before normalization
	graph := &DungeonGraph{
		Rooms: []DungeonRoom{
			{ID: "room_0", GraphPosition: GraphPosition{X: 0, Y: 0}},
			{ID: "room_1", GraphPosition: GraphPosition{X: 1, Y: 0}},
			{ID: "room_2", GraphPosition: GraphPosition{X: 0, Y: 1}}, // branch above room_0
		},
		Connections: []RoomConnection{
			{ID: "conn_0_1", FromRoomID: "room_0", ToRoomID: "room_1"},
			{ID: "conn_0_2", FromRoomID: "room_0", ToRoomID: "room_2"},
		},
		MainPathLength: 2,
	}
	assignments := makeDummyAssignments(graph)
	ComputeLayout(graph, assignments)

	// Find minimum origin
	minRow := graph.Rooms[0].Bounds.OriginRow
	minCol := graph.Rooms[0].Bounds.OriginCol
	for _, r := range graph.Rooms {
		if r.Bounds.OriginRow < minRow {
			minRow = r.Bounds.OriginRow
		}
		if r.Bounds.OriginCol < minCol {
			minCol = r.Bounds.OriginCol
		}
	}
	for _, c := range graph.Connections {
		if c.CorridorBounds.OriginRow < minRow {
			minRow = c.CorridorBounds.OriginRow
		}
		if c.CorridorBounds.OriginCol < minCol {
			minCol = c.CorridorBounds.OriginCol
		}
	}

	if minRow != 0 {
		t.Errorf("min OriginRow = %d, want 0", minRow)
	}
	if minCol != 0 {
		t.Errorf("min OriginCol = %d, want 0", minCol)
	}
}

func TestComputeLayout_CompositionDimensions(t *testing.T) {
	for seed := int64(0); seed < 10; seed++ {
		rng := rand.New(rand.NewSource(seed))
		graph := GenerateGraph(DungeonConfig{Seed: seed, Size: SizeMedium}, rng)
		AssignRoomTypes(graph, rng, SizeMedium, false)
		assignments := makeDummyAssignments(graph)
		comp := ComputeLayout(graph, assignments)

		// Verify dimensions match max extent
		maxRow, maxCol := 0, 0
		for _, p := range comp.Placements {
			if end := p.OriginRow + TileSize; end > maxRow {
				maxRow = end
			}
			if end := p.OriginCol + TileSize; end > maxCol {
				maxCol = end
			}
		}

		if comp.Rows != maxRow {
			t.Errorf("seed %d: Rows = %d, want %d", seed, comp.Rows, maxRow)
		}
		if comp.Cols != maxCol {
			t.Errorf("seed %d: Cols = %d, want %d", seed, comp.Cols, maxCol)
		}
	}
}

func TestComputeLayout_PlacementsMatchAssignments(t *testing.T) {
	graph := makeLinearGraph()
	assignments := makeDummyAssignments(graph)
	// Give different tile IDs for variety
	assignments[0].TileID = "entrance_tile"
	assignments[0].Rotation = 1
	assignments[2].TileID = "boss_tile"
	assignments[2].Rotation = 2

	comp := ComputeLayout(graph, assignments)

	expectedCount := len(graph.Rooms) + len(graph.Connections)
	if len(comp.Placements) != expectedCount {
		t.Fatalf("placements: got %d, want %d", len(comp.Placements), expectedCount)
	}

	// Build placement lookup
	placementByNode := make(map[string]MacroTilePlacement)
	for _, p := range comp.Placements {
		placementByNode[p.NodeID] = p
	}

	// Verify assignments match
	for _, a := range assignments {
		p, ok := placementByNode[a.NodeID]
		if !ok {
			t.Errorf("no placement for node %s", a.NodeID)
			continue
		}
		if p.TileID != a.TileID {
			t.Errorf("node %s: TileID = %q, want %q", a.NodeID, p.TileID, a.TileID)
		}
		if p.Rotation != a.Rotation {
			t.Errorf("node %s: Rotation = %d, want %d", a.NodeID, p.Rotation, a.Rotation)
		}
	}
}

func TestComputeLayout_MultipleSizes(t *testing.T) {
	for _, size := range []DungeonSize{SizeShort, SizeMedium, SizeLong} {
		t.Run(string(size), func(t *testing.T) {
			for seed := int64(0); seed < 5; seed++ {
				rng := rand.New(rand.NewSource(seed))
				graph := GenerateGraph(DungeonConfig{Seed: seed, Size: size}, rng)
				AssignRoomTypes(graph, rng, size, false)
				assignments := makeDummyAssignments(graph)
				comp := ComputeLayout(graph, assignments)

				expectedCount := len(graph.Rooms) + len(graph.Connections)
				if len(comp.Placements) != expectedCount {
					t.Errorf("seed %d: placements = %d, want %d",
						seed, len(comp.Placements), expectedCount)
				}

				if comp.Rows <= 0 || comp.Cols <= 0 {
					t.Errorf("seed %d: dimensions %dx%d invalid", seed, comp.Rows, comp.Cols)
				}
			}
		})
	}
}

// --- Test helpers ---

func expectBounds(t *testing.T, label string, got RoomBounds, wantRow, wantCol, wantRows, wantCols int) {
	t.Helper()
	if got.OriginRow != wantRow || got.OriginCol != wantCol || got.Rows != wantRows || got.Cols != wantCols {
		t.Errorf("%s: bounds = {%d,%d,%d,%d}, want {%d,%d,%d,%d}",
			label, got.OriginRow, got.OriginCol, got.Rows, got.Cols,
			wantRow, wantCol, wantRows, wantCols)
	}
}

// boundsOverlap checks if two TileSize×TileSize tiles at the given origins overlap.
func boundsOverlap(aRow, aCol, bRow, bCol int) bool {
	return aCol < bCol+TileSize &&
		aCol+TileSize > bCol &&
		aRow < bRow+TileSize &&
		aRow+TileSize > bRow
}

// isAdjacent checks if two bounds share an edge (touching but not overlapping).
func isAdjacent(a, b RoomBounds) bool {
	// Horizontal adjacency: same row range, columns touch
	if a.OriginRow == b.OriginRow {
		if a.OriginCol+a.Cols == b.OriginCol || b.OriginCol+b.Cols == a.OriginCol {
			return true
		}
	}
	// Vertical adjacency: same column range, rows touch
	if a.OriginCol == b.OriginCol {
		if a.OriginRow+a.Rows == b.OriginRow || b.OriginRow+b.Rows == a.OriginRow {
			return true
		}
	}
	return false
}
