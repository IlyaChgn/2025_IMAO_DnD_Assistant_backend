package dungeongen

import (
	"log"
	"math/rand"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// tileCandidate pairs a metadata entry with a specific rotation.
type tileCandidate struct {
	meta     *models.TileMetadata
	rotation int
}

// neighborInfo describes which side of this room faces a specific neighbor.
type neighborInfo struct {
	neighborID string
	side       Side // which side of THIS room faces the neighbor
}

// AssignTiles selects a real tile + rotation for every room and connection
// in the dungeon graph, using edge-signature matching for compatibility.
func AssignTiles(graph *DungeonGraph, metadata []*models.TileMetadata, themeTags []string, rng *rand.Rand) []TileAssignment {
	roomByID := buildRoomIndex(graph)
	neighbors := buildNeighborMap(graph, roomByID)
	filtered := filterByThemeTags(metadata, themeTags)

	// Assigned tiles index: nodeID → TileAssignment
	assigned := make(map[string]TileAssignment)
	// Metadata index: tileID → *TileMetadata (for edge signature lookup)
	metaByID := make(map[string]*models.TileMetadata, len(metadata))
	for _, m := range metadata {
		metaByID[m.TileID] = m
	}

	// BFS from entrance (room 0)
	order := bfsOrder(graph)

	// Step 1: Assign tiles to rooms
	for _, roomID := range order {
		nbrs := neighbors[roomID]
		candidates := findRoomCandidates(filtered, nbrs, assigned, metaByID)

		if len(candidates) == 0 {
			// Fuzzy fallback: hamming distance ≤ 1
			candidates = findRoomCandidatesFuzzy(filtered, nbrs, assigned, metaByID)
			if len(candidates) > 0 {
				log.Printf("TileAssignment: fuzzy match for room %s", roomID)
			}
		}

		if len(candidates) == 0 {
			// Last resort: any tile with correct openings, ignore edge matching
			candidates = findRoomCandidatesOpeningsOnly(filtered, nbrs)
			if len(candidates) > 0 {
				log.Printf("TileAssignment: openings-only fallback for room %s", roomID)
			}
		}

		if len(candidates) == 0 {
			// Absolute fallback: first tile from filtered, rotation 0
			if len(filtered) > 0 {
				candidates = []tileCandidate{{meta: filtered[0], rotation: 0}}
				log.Printf("TileAssignment: absolute fallback for room %s", roomID)
			}
		}

		if len(candidates) > 0 {
			pick := candidates[rng.Intn(len(candidates))]
			assigned[roomID] = TileAssignment{
				NodeID:   roomID,
				TileID:   pick.meta.TileID,
				Rotation: pick.rotation,
			}
		}
	}

	// Step 2: Assign corridor tiles for connections
	for _, conn := range graph.Connections {
		assignCorridor(conn, graph, roomByID, filtered, assigned, metaByID, rng)
	}

	// Collect results
	results := make([]TileAssignment, 0, len(assigned))
	for _, a := range assigned {
		results = append(results, a)
	}

	return results
}

// findRoomCandidates returns tiles that satisfy both opening requirements
// and exact edge-signature matching with already-assigned neighbors.
func findRoomCandidates(
	tiles []*models.TileMetadata,
	nbrs []neighborInfo,
	assigned map[string]TileAssignment,
	metaByID map[string]*models.TileMetadata,
) []tileCandidate {
	var candidates []tileCandidate

	for _, m := range tiles {
		for rot := 0; rot < 4; rot++ {
			if !hasRequiredOpenings(m, rot, nbrs) {
				continue
			}
			if !edgesMatchExact(m, rot, nbrs, assigned, metaByID) {
				continue
			}
			candidates = append(candidates, tileCandidate{meta: m, rotation: rot})
		}
	}

	return candidates
}

// findRoomCandidatesFuzzy is like findRoomCandidates but allows hamming distance ≤ 1.
func findRoomCandidatesFuzzy(
	tiles []*models.TileMetadata,
	nbrs []neighborInfo,
	assigned map[string]TileAssignment,
	metaByID map[string]*models.TileMetadata,
) []tileCandidate {
	var candidates []tileCandidate

	for _, m := range tiles {
		for rot := 0; rot < 4; rot++ {
			if !hasRequiredOpenings(m, rot, nbrs) {
				continue
			}
			if !edgesMatchFuzzy(m, rot, nbrs, assigned, metaByID) {
				continue
			}
			candidates = append(candidates, tileCandidate{meta: m, rotation: rot})
		}
	}

	return candidates
}

// findRoomCandidatesOpeningsOnly ignores edge signatures entirely.
func findRoomCandidatesOpeningsOnly(
	tiles []*models.TileMetadata,
	nbrs []neighborInfo,
) []tileCandidate {
	var candidates []tileCandidate

	for _, m := range tiles {
		for rot := 0; rot < 4; rot++ {
			if hasRequiredOpenings(m, rot, nbrs) {
				candidates = append(candidates, tileCandidate{meta: m, rotation: rot})
			}
		}
	}

	return candidates
}

// hasRequiredOpenings checks that the tile at the given rotation
// has an opening on every side where a neighbor connection exists.
func hasRequiredOpenings(m *models.TileMetadata, rotation int, nbrs []neighborInfo) bool {
	openings := RotateOpenings(m.Openings, rotation)

	for _, n := range nbrs {
		switch n.side {
		case SideTop:
			if !openings.Top {
				return false
			}
		case SideRight:
			if !openings.Right {
				return false
			}
		case SideBottom:
			if !openings.Bottom {
				return false
			}
		case SideLeft:
			if !openings.Left {
				return false
			}
		}
	}

	return true
}

// edgesMatchExact checks all already-assigned neighbors for exact signature match.
func edgesMatchExact(
	m *models.TileMetadata, rotation int,
	nbrs []neighborInfo,
	assigned map[string]TileAssignment,
	metaByID map[string]*models.TileMetadata,
) bool {
	for _, n := range nbrs {
		nbrAssignment, ok := assigned[n.neighborID]
		if !ok {
			continue // Neighbor not yet assigned — skip
		}
		nbrMeta, ok := metaByID[nbrAssignment.TileID]
		if !ok {
			continue
		}

		mySig := GetSignatureForSide(m.EdgeSignatures, rotation, n.side)
		nbrSide := oppositeSide(n.side)
		nbrSig := GetSignatureForSide(nbrMeta.EdgeSignatures, nbrAssignment.Rotation, nbrSide)

		if mySig != nbrSig {
			return false
		}
	}

	return true
}

// edgesMatchFuzzy allows hamming distance ≤ 1 for each neighbor pair.
func edgesMatchFuzzy(
	m *models.TileMetadata, rotation int,
	nbrs []neighborInfo,
	assigned map[string]TileAssignment,
	metaByID map[string]*models.TileMetadata,
) bool {
	for _, n := range nbrs {
		nbrAssignment, ok := assigned[n.neighborID]
		if !ok {
			continue
		}
		nbrMeta, ok := metaByID[nbrAssignment.TileID]
		if !ok {
			continue
		}

		mySig := GetSignatureForSide(m.EdgeSignatures, rotation, n.side)
		nbrSide := oppositeSide(n.side)
		nbrSig := GetSignatureForSide(nbrMeta.EdgeSignatures, nbrAssignment.Rotation, nbrSide)

		if HammingDistance(mySig, nbrSig) > 1 {
			return false
		}
	}

	return true
}

// assignCorridor assigns a corridor tile for a connection.
func assignCorridor(
	conn RoomConnection,
	graph *DungeonGraph,
	roomByID map[string]*DungeonRoom,
	tiles []*models.TileMetadata,
	assigned map[string]TileAssignment,
	metaByID map[string]*models.TileMetadata,
	rng *rand.Rand,
) {
	fromRoom := roomByID[conn.FromRoomID]
	toRoom := roomByID[conn.ToRoomID]
	if fromRoom == nil || toRoom == nil {
		return
	}

	// Determine corridor orientation
	isHorizontal := fromRoom.GraphPosition.Y == toRoom.GraphPosition.Y
	var targetRole models.TileRole
	if isHorizontal {
		targetRole = models.TileRoleCorridorH
	} else {
		targetRole = models.TileRoleCorridorV
	}

	// Determine which sides the corridor faces
	var fromSide, toSide Side
	if isHorizontal {
		if fromRoom.GraphPosition.X < toRoom.GraphPosition.X {
			fromSide = SideRight // corridor's left faces fromRoom's right
			toSide = SideLeft   // corridor's right faces toRoom's left
		} else {
			fromSide = SideLeft
			toSide = SideRight
		}
	} else {
		if fromRoom.GraphPosition.Y < toRoom.GraphPosition.Y {
			fromSide = SideTop   // fromRoom is below, corridor faces up
			toSide = SideBottom  // toRoom is above, corridor faces down
		} else {
			fromSide = SideBottom
			toSide = SideTop
		}
	}

	// Filter corridor tiles
	corridorTiles := filterByRole(tiles, targetRole)
	if len(corridorTiles) == 0 {
		// Fallback: try any corridor type
		corridorTiles = filterByRole(tiles, models.TileRoleCorridorH)
		corridorTiles = append(corridorTiles, filterByRole(tiles, models.TileRoleCorridorV)...)
	}

	// Find compatible corridor
	var candidates []tileCandidate
	for _, m := range corridorTiles {
		for rot := 0; rot < 4; rot++ {
			compatible := true

			// Check against fromRoom
			if fromAssign, ok := assigned[conn.FromRoomID]; ok {
				if fromMeta, ok := metaByID[fromAssign.TileID]; ok {
					corridorSig := GetSignatureForSide(m.EdgeSignatures, rot, oppositeSide(fromSide))
					roomSig := GetSignatureForSide(fromMeta.EdgeSignatures, fromAssign.Rotation, fromSide)
					if corridorSig != roomSig && HammingDistance(corridorSig, roomSig) > 1 {
						compatible = false
					}
				}
			}

			// Check against toRoom
			if compatible {
				if toAssign, ok := assigned[conn.ToRoomID]; ok {
					if toMeta, ok := metaByID[toAssign.TileID]; ok {
						corridorSig := GetSignatureForSide(m.EdgeSignatures, rot, oppositeSide(toSide))
						roomSig := GetSignatureForSide(toMeta.EdgeSignatures, toAssign.Rotation, toSide)
						if corridorSig != roomSig && HammingDistance(corridorSig, roomSig) > 1 {
							compatible = false
						}
					}
				}
			}

			if compatible {
				candidates = append(candidates, tileCandidate{meta: m, rotation: rot})
			}
		}
	}

	if len(candidates) == 0 && len(corridorTiles) > 0 {
		// Fallback: pick any corridor tile
		candidates = []tileCandidate{{meta: corridorTiles[rng.Intn(len(corridorTiles))], rotation: 0}}
		log.Printf("TileAssignment: corridor fallback for connection %s", conn.ID)
	}

	if len(candidates) > 0 {
		pick := candidates[rng.Intn(len(candidates))]
		assigned[conn.ID] = TileAssignment{
			NodeID:   conn.ID,
			TileID:   pick.meta.TileID,
			Rotation: pick.rotation,
		}
	}
}

// --- Helpers ---

// GetSignatureForSide returns the edge signature string for a given rotation and side.
func GetSignatureForSide(sigs models.EdgeSignatures, rotation int, side Side) string {
	rotation = ((rotation % 4) + 4) % 4
	switch rotation {
	case 0:
		switch side {
		case SideTop:
			return sigs.Top
		case SideRight:
			return sigs.Right
		case SideBottom:
			return sigs.Bottom
		case SideLeft:
			return sigs.Left
		}
	case 1:
		switch side {
		case SideTop:
			return sigs.R1Top
		case SideRight:
			return sigs.R1Right
		case SideBottom:
			return sigs.R1Bottom
		case SideLeft:
			return sigs.R1Left
		}
	case 2:
		switch side {
		case SideTop:
			return sigs.R2Top
		case SideRight:
			return sigs.R2Right
		case SideBottom:
			return sigs.R2Bottom
		case SideLeft:
			return sigs.R2Left
		}
	case 3:
		switch side {
		case SideTop:
			return sigs.R3Top
		case SideRight:
			return sigs.R3Right
		case SideBottom:
			return sigs.R3Bottom
		case SideLeft:
			return sigs.R3Left
		}
	}
	return ""
}

// RotateOpenings transforms openings for a given rotation (CCW).
// Rotation 1 (90° CCW): top←right, right←bottom, bottom←left, left←top
// This matches what happens when you rotate the grid: the cells that were
// on the right side are now on top after 90° CCW rotation.
func RotateOpenings(o models.OpeningSummary, rotation int) models.OpeningSummary {
	rotation = ((rotation % 4) + 4) % 4
	result := o
	for i := 0; i < rotation; i++ {
		result = models.OpeningSummary{
			Top:    result.Right,
			Right:  result.Bottom,
			Bottom: result.Left,
			Left:   result.Top,
		}
	}
	return result
}

// HammingDistance returns the number of differing characters between two strings.
func HammingDistance(a, b string) int {
	if len(a) != len(b) {
		// Different lengths: count length difference + character mismatches
		minLen := len(a)
		if len(b) < minLen {
			minLen = len(b)
		}
		dist := len(a) - len(b)
		if dist < 0 {
			dist = -dist
		}
		for i := 0; i < minLen; i++ {
			if a[i] != b[i] {
				dist++
			}
		}
		return dist
	}

	dist := 0
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			dist++
		}
	}
	return dist
}

func oppositeSide(s Side) Side {
	switch s {
	case SideTop:
		return SideBottom
	case SideBottom:
		return SideTop
	case SideLeft:
		return SideRight
	case SideRight:
		return SideLeft
	}
	return s
}

// buildRoomIndex creates a map from room ID to room pointer.
func buildRoomIndex(graph *DungeonGraph) map[string]*DungeonRoom {
	idx := make(map[string]*DungeonRoom, len(graph.Rooms))
	for i := range graph.Rooms {
		idx[graph.Rooms[i].ID] = &graph.Rooms[i]
	}
	return idx
}

// buildNeighborMap computes which sides of each room connect to which neighbors,
// based on graph positions.
func buildNeighborMap(graph *DungeonGraph, roomByID map[string]*DungeonRoom) map[string][]neighborInfo {
	nbrs := make(map[string][]neighborInfo)

	for _, conn := range graph.Connections {
		from := roomByID[conn.FromRoomID]
		to := roomByID[conn.ToRoomID]
		if from == nil || to == nil {
			continue
		}

		fromSide, toSide := connectionSides(from.GraphPosition, to.GraphPosition)

		nbrs[conn.FromRoomID] = append(nbrs[conn.FromRoomID], neighborInfo{
			neighborID: conn.ToRoomID,
			side:       fromSide,
		})
		nbrs[conn.ToRoomID] = append(nbrs[conn.ToRoomID], neighborInfo{
			neighborID: conn.FromRoomID,
			side:       toSide,
		})
	}

	return nbrs
}

// connectionSides determines which side of each room faces the other
// based on their graph positions.
func connectionSides(from, to GraphPosition) (fromSide, toSide Side) {
	dx := to.X - from.X
	dy := to.Y - from.Y

	if dx > 0 {
		return SideRight, SideLeft
	}
	if dx < 0 {
		return SideLeft, SideRight
	}
	// dx == 0: vertical connection
	if dy > 0 {
		return SideTop, SideBottom
	}
	return SideBottom, SideTop
}

// bfsOrder returns room IDs in BFS order starting from the entrance (room 0).
func bfsOrder(graph *DungeonGraph) []string {
	if len(graph.Rooms) == 0 {
		return nil
	}

	adj := make(map[string][]string)
	for _, c := range graph.Connections {
		adj[c.FromRoomID] = append(adj[c.FromRoomID], c.ToRoomID)
		adj[c.ToRoomID] = append(adj[c.ToRoomID], c.FromRoomID)
	}

	visited := make(map[string]bool)
	order := make([]string, 0, len(graph.Rooms))
	queue := []string{graph.Rooms[0].ID}
	visited[graph.Rooms[0].ID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, neighbor := range adj[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	return order
}

// filterByThemeTags returns tiles that have at least one matching theme tag.
func filterByThemeTags(metadata []*models.TileMetadata, tags []string) []*models.TileMetadata {
	if len(tags) == 0 {
		return metadata
	}

	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}

	var result []*models.TileMetadata
	for _, m := range metadata {
		for _, t := range m.ThemeTags {
			if tagSet[t] {
				result = append(result, m)
				break
			}
		}
	}

	if len(result) == 0 {
		return metadata // Fallback: return all if no theme matches
	}
	return result
}

// filterByRole returns tiles with a specific role.
func filterByRole(metadata []*models.TileMetadata, role models.TileRole) []*models.TileMetadata {
	var result []*models.TileMetadata
	for _, m := range metadata {
		if m.Role == role {
			result = append(result, m)
		}
	}
	return result
}
