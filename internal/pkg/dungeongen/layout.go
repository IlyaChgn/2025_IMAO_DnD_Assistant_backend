package dungeongen

import "log"

// gridPos identifies a macro-tile slot on the layout grid.
type gridPos struct {
	macroRow int
	macroCol int
}

// ComputeLayout positions all rooms and corridors on a 2D grid.
// It mutates graph.Rooms[*].Bounds and graph.Connections[*].CorridorBounds,
// and returns a MapComposition with all placements and total dimensions.
func ComputeLayout(graph *DungeonGraph, assignments []TileAssignment) *MapComposition {
	assignIdx := buildAssignmentIndex(assignments)
	connIdx := buildConnectionIndex(graph)

	// Place main path rooms + corridors
	occupied := placeMainPath(graph)

	// Place branch rooms + corridors
	placeBranches(graph, connIdx, occupied)

	// Normalize so min origin = (0,0)
	normalizeCoordinates(graph)

	// Compute total dimensions
	totalRows, totalCols := computeDimensions(graph)

	// Build placements
	placements := buildPlacements(graph, assignIdx)

	return &MapComposition{
		Rows:       totalRows,
		Cols:       totalCols,
		Placements: placements,
	}
}

// buildAssignmentIndex creates a map from nodeID to TileAssignment.
func buildAssignmentIndex(assignments []TileAssignment) map[string]TileAssignment {
	idx := make(map[string]TileAssignment, len(assignments))
	for _, a := range assignments {
		idx[a.NodeID] = a
	}
	return idx
}

// buildConnectionIndex creates a bidirectional lookup for connections.
func buildConnectionIndex(graph *DungeonGraph) map[string]*RoomConnection {
	idx := make(map[string]*RoomConnection, len(graph.Connections)*2)
	for i := range graph.Connections {
		c := &graph.Connections[i]
		idx[c.FromRoomID+"|"+c.ToRoomID] = c
		idx[c.ToRoomID+"|"+c.FromRoomID] = c
	}
	return idx
}

// findConnection looks up the connection between two rooms.
func findConnection(idx map[string]*RoomConnection, roomA, roomB string) *RoomConnection {
	return idx[roomA+"|"+roomB]
}

// placeMainPath sets bounds for main path rooms (Y=0) and their corridors.
func placeMainPath(graph *DungeonGraph) map[gridPos]string {
	occupied := make(map[gridPos]string)

	// Place rooms along main path left-to-right
	for i := 0; i < graph.MainPathLength; i++ {
		room := &graph.Rooms[i]
		room.Bounds = RoomBounds{
			OriginRow: 0,
			OriginCol: i * 2 * TileSize,
			Rows:      TileSize,
			Cols:      TileSize,
		}
		markOccupied(occupied, 0, i*2*TileSize, room.ID)
	}

	// Place corridors between consecutive main path rooms
	for i := range graph.Connections {
		conn := &graph.Connections[i]

		// Find the rooms
		var fromIdx, toIdx int
		fromFound, toFound := false, false
		for j := 0; j < graph.MainPathLength; j++ {
			if graph.Rooms[j].ID == conn.FromRoomID {
				fromIdx = j
				fromFound = true
			}
			if graph.Rooms[j].ID == conn.ToRoomID {
				toIdx = j
				toFound = true
			}
		}

		// Only handle main-path-to-main-path connections here
		if !fromFound || !toFound {
			continue
		}

		// Ensure left room is first
		leftIdx := fromIdx
		if toIdx < fromIdx {
			leftIdx = toIdx
		}

		_ = toIdx // both are on main path

		conn.CorridorBounds = RoomBounds{
			OriginRow: 0,
			OriginCol: leftIdx*2*TileSize + TileSize,
			Rows:      TileSize,
			Cols:      TileSize,
		}
		markOccupied(occupied, 0, conn.CorridorBounds.OriginCol, conn.ID)
	}

	return occupied
}

// placeBranches sets bounds for branch rooms (Y!=0) and their corridors.
func placeBranches(graph *DungeonGraph, connIdx map[string]*RoomConnection, occupied map[gridPos]string) {
	for i := graph.MainPathLength; i < len(graph.Rooms); i++ {
		branch := &graph.Rooms[i]
		parentX := branch.GraphPosition.X

		// Find parent room on main path
		var parent *DungeonRoom
		for j := 0; j < graph.MainPathLength; j++ {
			if graph.Rooms[j].GraphPosition.X == parentX {
				parent = &graph.Rooms[j]
				break
			}
		}
		if parent == nil {
			log.Printf("layout: branch room %s has no parent at X=%d", branch.ID, parentX)
			continue
		}

		var branchRow, corridorRow int
		if branch.GraphPosition.Y > 0 {
			// Above: negative row direction
			corridorRow = parent.Bounds.OriginRow - TileSize
			branchRow = parent.Bounds.OriginRow - 2*TileSize
		} else {
			// Below: positive row direction
			corridorRow = parent.Bounds.OriginRow + TileSize
			branchRow = parent.Bounds.OriginRow + 2*TileSize
		}

		// Collision check on branch position
		branchPos := gridPos{branchRow / TileSize, parent.Bounds.OriginCol / TileSize}
		if existingID, ok := occupied[branchPos]; ok {
			log.Printf("layout: collision at (%d,%d) — %s conflicts with %s",
				branchRow, parent.Bounds.OriginCol, branch.ID, existingID)
		}

		branch.Bounds = RoomBounds{
			OriginRow: branchRow,
			OriginCol: parent.Bounds.OriginCol,
			Rows:      TileSize,
			Cols:      TileSize,
		}
		markOccupied(occupied, branchRow, parent.Bounds.OriginCol, branch.ID)

		// Place corridor between parent and branch
		conn := findConnection(connIdx, parent.ID, branch.ID)
		if conn != nil {
			conn.CorridorBounds = RoomBounds{
				OriginRow: corridorRow,
				OriginCol: parent.Bounds.OriginCol,
				Rows:      TileSize,
				Cols:      TileSize,
			}
			markOccupied(occupied, corridorRow, parent.Bounds.OriginCol, conn.ID)
		}
	}
}

// normalizeCoordinates shifts all bounds so the minimum origin is (0, 0).
func normalizeCoordinates(graph *DungeonGraph) {
	minRow, minCol := 0, 0

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

	if minRow == 0 && minCol == 0 {
		return
	}

	for i := range graph.Rooms {
		graph.Rooms[i].Bounds.OriginRow -= minRow
		graph.Rooms[i].Bounds.OriginCol -= minCol
	}
	for i := range graph.Connections {
		graph.Connections[i].CorridorBounds.OriginRow -= minRow
		graph.Connections[i].CorridorBounds.OriginCol -= minCol
	}
}

// computeDimensions returns total (rows, cols) based on max extent of all tiles.
func computeDimensions(graph *DungeonGraph) (int, int) {
	maxRow, maxCol := 0, 0

	for _, r := range graph.Rooms {
		if end := r.Bounds.OriginRow + r.Bounds.Rows; end > maxRow {
			maxRow = end
		}
		if end := r.Bounds.OriginCol + r.Bounds.Cols; end > maxCol {
			maxCol = end
		}
	}
	for _, c := range graph.Connections {
		if end := c.CorridorBounds.OriginRow + c.CorridorBounds.Rows; end > maxRow {
			maxRow = end
		}
		if end := c.CorridorBounds.OriginCol + c.CorridorBounds.Cols; end > maxCol {
			maxCol = end
		}
	}

	return maxRow, maxCol
}

// buildPlacements creates the final placement list from bounds + assignments.
func buildPlacements(graph *DungeonGraph, assignIdx map[string]TileAssignment) []MacroTilePlacement {
	placements := make([]MacroTilePlacement, 0, len(graph.Rooms)+len(graph.Connections))

	for _, r := range graph.Rooms {
		assign := assignIdx[r.ID]
		placements = append(placements, MacroTilePlacement{
			TileID:    assign.TileID,
			NodeID:    r.ID,
			OriginRow: r.Bounds.OriginRow,
			OriginCol: r.Bounds.OriginCol,
			Rotation:  assign.Rotation,
		})
	}

	for _, c := range graph.Connections {
		assign := assignIdx[c.ID]
		placements = append(placements, MacroTilePlacement{
			TileID:    assign.TileID,
			NodeID:    c.ID,
			OriginRow: c.CorridorBounds.OriginRow,
			OriginCol: c.CorridorBounds.OriginCol,
			Rotation:  assign.Rotation,
		})
	}

	return placements
}

// markOccupied records a tile position in the occupancy grid.
func markOccupied(occupied map[gridPos]string, row, col int, nodeID string) {
	pos := gridPos{row / TileSize, col / TileSize}
	if existing, ok := occupied[pos]; ok {
		log.Printf("layout: slot (%d,%d) already occupied by %s, overwriting with %s",
			pos.macroRow, pos.macroCol, existing, nodeID)
	}
	occupied[pos] = nodeID
}
