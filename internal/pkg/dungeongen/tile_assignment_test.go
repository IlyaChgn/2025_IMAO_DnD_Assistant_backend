package dungeongen

import (
	"math/rand"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// --- GetSignatureForSide ---

func TestGetSignatureForSide_Rotation0(t *testing.T) {
	sigs := models.EdgeSignatures{
		Top: "111111", Right: "100001", Bottom: "000000", Left: "110011",
	}
	if got := GetSignatureForSide(sigs, 0, SideTop); got != "111111" {
		t.Errorf("rot0 top: got %q, want %q", got, "111111")
	}
	if got := GetSignatureForSide(sigs, 0, SideRight); got != "100001" {
		t.Errorf("rot0 right: got %q, want %q", got, "100001")
	}
	if got := GetSignatureForSide(sigs, 0, SideBottom); got != "000000" {
		t.Errorf("rot0 bottom: got %q, want %q", got, "000000")
	}
	if got := GetSignatureForSide(sigs, 0, SideLeft); got != "110011" {
		t.Errorf("rot0 left: got %q, want %q", got, "110011")
	}
}

func TestGetSignatureForSide_AllRotations(t *testing.T) {
	sigs := models.EdgeSignatures{
		Top: "r0t", Right: "r0r", Bottom: "r0b", Left: "r0l",
		R1Top: "r1t", R1Right: "r1r", R1Bottom: "r1b", R1Left: "r1l",
		R2Top: "r2t", R2Right: "r2r", R2Bottom: "r2b", R2Left: "r2l",
		R3Top: "r3t", R3Right: "r3r", R3Bottom: "r3b", R3Left: "r3l",
	}

	tests := []struct {
		rot  int
		side Side
		want string
	}{
		{0, SideTop, "r0t"}, {0, SideRight, "r0r"}, {0, SideBottom, "r0b"}, {0, SideLeft, "r0l"},
		{1, SideTop, "r1t"}, {1, SideRight, "r1r"}, {1, SideBottom, "r1b"}, {1, SideLeft, "r1l"},
		{2, SideTop, "r2t"}, {2, SideRight, "r2r"}, {2, SideBottom, "r2b"}, {2, SideLeft, "r2l"},
		{3, SideTop, "r3t"}, {3, SideRight, "r3r"}, {3, SideBottom, "r3b"}, {3, SideLeft, "r3l"},
	}

	for _, tt := range tests {
		got := GetSignatureForSide(sigs, tt.rot, tt.side)
		if got != tt.want {
			t.Errorf("rot=%d side=%d: got %q, want %q", tt.rot, tt.side, got, tt.want)
		}
	}
}

func TestGetSignatureForSide_NegativeRotation(t *testing.T) {
	sigs := models.EdgeSignatures{
		R3Top: "neg1",
	}
	// -1 mod 4 should be 3
	if got := GetSignatureForSide(sigs, -1, SideTop); got != "neg1" {
		t.Errorf("rot -1 top: got %q, want %q", got, "neg1")
	}
}

// --- RotateOpenings ---

func TestRotateOpenings_Identity(t *testing.T) {
	o := models.OpeningSummary{Top: true, Right: false, Bottom: true, Left: false}
	got := RotateOpenings(o, 0)
	if got != o {
		t.Errorf("rot 0: got %+v, want %+v", got, o)
	}
}

func TestRotateOpenings_90CCW(t *testing.T) {
	o := models.OpeningSummary{Top: true, Right: false, Bottom: false, Left: false}
	got := RotateOpenings(o, 1)
	// 90° CCW: top←right, right←bottom, bottom←left, left←top
	// So original Top=true goes to Left. New Top comes from Right=false.
	want := models.OpeningSummary{Top: false, Right: false, Bottom: false, Left: true}
	if got != want {
		t.Errorf("rot 1: got %+v, want %+v", got, want)
	}
}

func TestRotateOpenings_180(t *testing.T) {
	o := models.OpeningSummary{Top: true, Right: true, Bottom: false, Left: false}
	got := RotateOpenings(o, 2)
	want := models.OpeningSummary{Top: false, Right: false, Bottom: true, Left: true}
	if got != want {
		t.Errorf("rot 2: got %+v, want %+v", got, want)
	}
}

func TestRotateOpenings_FullCycle(t *testing.T) {
	o := models.OpeningSummary{Top: true, Right: false, Bottom: true, Left: false}
	got := RotateOpenings(o, 4)
	if got != o {
		t.Errorf("rot 4 (full cycle): got %+v, want %+v", got, o)
	}
}

// --- HammingDistance ---

func TestHammingDistance_Identical(t *testing.T) {
	if d := HammingDistance("111000", "111000"); d != 0 {
		t.Errorf("identical: got %d, want 0", d)
	}
}

func TestHammingDistance_OneDiff(t *testing.T) {
	if d := HammingDistance("111000", "110000"); d != 1 {
		t.Errorf("one diff: got %d, want 1", d)
	}
}

func TestHammingDistance_AllDiff(t *testing.T) {
	if d := HammingDistance("111111", "000000"); d != 6 {
		t.Errorf("all diff: got %d, want 6", d)
	}
}

func TestHammingDistance_DifferentLengths(t *testing.T) {
	// "111" vs "1100" → shared 3: 1 mismatch at pos 2, length diff 1 → total 2
	if d := HammingDistance("111", "1100"); d != 2 {
		t.Errorf("diff len: got %d, want 2", d)
	}
}

func TestHammingDistance_Empty(t *testing.T) {
	if d := HammingDistance("", ""); d != 0 {
		t.Errorf("empty: got %d, want 0", d)
	}
}

// --- oppositeSide ---

func TestOppositeSide(t *testing.T) {
	tests := []struct {
		in, want Side
	}{
		{SideTop, SideBottom},
		{SideBottom, SideTop},
		{SideLeft, SideRight},
		{SideRight, SideLeft},
	}
	for _, tt := range tests {
		if got := oppositeSide(tt.in); got != tt.want {
			t.Errorf("oppositeSide(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// --- connectionSides ---

func TestConnectionSides_Horizontal(t *testing.T) {
	fromSide, toSide := connectionSides(
		GraphPosition{X: 0, Y: 0},
		GraphPosition{X: 1, Y: 0},
	)
	if fromSide != SideRight || toSide != SideLeft {
		t.Errorf("horizontal: fromSide=%d toSide=%d, want Right/Left", fromSide, toSide)
	}
}

func TestConnectionSides_HorizontalReverse(t *testing.T) {
	fromSide, toSide := connectionSides(
		GraphPosition{X: 2, Y: 0},
		GraphPosition{X: 1, Y: 0},
	)
	if fromSide != SideLeft || toSide != SideRight {
		t.Errorf("horizontal reverse: fromSide=%d toSide=%d, want Left/Right", fromSide, toSide)
	}
}

func TestConnectionSides_Vertical(t *testing.T) {
	fromSide, toSide := connectionSides(
		GraphPosition{X: 1, Y: 0},
		GraphPosition{X: 1, Y: 1},
	)
	if fromSide != SideTop || toSide != SideBottom {
		t.Errorf("vertical: fromSide=%d toSide=%d, want Top/Bottom", fromSide, toSide)
	}
}

func TestConnectionSides_VerticalReverse(t *testing.T) {
	fromSide, toSide := connectionSides(
		GraphPosition{X: 1, Y: 1},
		GraphPosition{X: 1, Y: 0},
	)
	if fromSide != SideBottom || toSide != SideTop {
		t.Errorf("vertical reverse: fromSide=%d toSide=%d, want Bottom/Top", fromSide, toSide)
	}
}

// --- bfsOrder ---

func TestBfsOrder_EmptyGraph(t *testing.T) {
	g := &DungeonGraph{}
	order := bfsOrder(g)
	if len(order) != 0 {
		t.Errorf("empty graph: got %d rooms, want 0", len(order))
	}
}

func TestBfsOrder_Linear(t *testing.T) {
	g := &DungeonGraph{
		Rooms: []DungeonRoom{
			{ID: "r0"}, {ID: "r1"}, {ID: "r2"},
		},
		Connections: []RoomConnection{
			{ID: "c01", FromRoomID: "r0", ToRoomID: "r1"},
			{ID: "c12", FromRoomID: "r1", ToRoomID: "r2"},
		},
	}
	order := bfsOrder(g)
	if len(order) != 3 {
		t.Fatalf("linear: got %d rooms, want 3", len(order))
	}
	if order[0] != "r0" {
		t.Errorf("linear: first room %q, want r0", order[0])
	}
	if order[1] != "r1" {
		t.Errorf("linear: second room %q, want r1", order[1])
	}
	if order[2] != "r2" {
		t.Errorf("linear: third room %q, want r2", order[2])
	}
}

// --- filterByThemeTags ---

func TestFilterByThemeTags_Match(t *testing.T) {
	tiles := []*models.TileMetadata{
		{TileID: "a", ThemeTags: []string{"dungeon", "basic"}},
		{TileID: "b", ThemeTags: []string{"cave"}},
		{TileID: "c", ThemeTags: []string{"dungeon"}},
	}
	result := filterByThemeTags(tiles, []string{"cave"})
	if len(result) != 1 || result[0].TileID != "b" {
		t.Errorf("expected 1 cave tile, got %d", len(result))
	}
}

func TestFilterByThemeTags_NoMatch_ReturnsAll(t *testing.T) {
	tiles := []*models.TileMetadata{
		{TileID: "a", ThemeTags: []string{"dungeon"}},
	}
	result := filterByThemeTags(tiles, []string{"forest"})
	if len(result) != 1 {
		t.Errorf("no match fallback: expected all tiles, got %d", len(result))
	}
}

func TestFilterByThemeTags_EmptyTags_ReturnsAll(t *testing.T) {
	tiles := []*models.TileMetadata{
		{TileID: "a"}, {TileID: "b"},
	}
	result := filterByThemeTags(tiles, nil)
	if len(result) != 2 {
		t.Errorf("empty tags: expected all tiles, got %d", len(result))
	}
}

// --- filterByRole ---

func TestFilterByRole(t *testing.T) {
	tiles := []*models.TileMetadata{
		{TileID: "a", Role: models.TileRoleRoom},
		{TileID: "b", Role: models.TileRoleCorridorH},
		{TileID: "c", Role: models.TileRoleRoom},
	}
	rooms := filterByRole(tiles, models.TileRoleRoom)
	if len(rooms) != 2 {
		t.Errorf("filterByRole room: got %d, want 2", len(rooms))
	}
	corridors := filterByRole(tiles, models.TileRoleCorridorH)
	if len(corridors) != 1 {
		t.Errorf("filterByRole corridor_h: got %d, want 1", len(corridors))
	}
}

// --- hasRequiredOpenings ---

func TestHasRequiredOpenings_AllOpen(t *testing.T) {
	m := &models.TileMetadata{
		Openings: models.OpeningSummary{Top: true, Right: true, Bottom: true, Left: true},
	}
	nbrs := []neighborInfo{
		{neighborID: "r1", side: SideRight},
		{neighborID: "r2", side: SideLeft},
	}
	if !hasRequiredOpenings(m, 0, nbrs) {
		t.Error("all-open tile should have required openings")
	}
}

func TestHasRequiredOpenings_MissingOpening(t *testing.T) {
	m := &models.TileMetadata{
		Openings: models.OpeningSummary{Top: true, Right: false, Bottom: true, Left: true},
	}
	nbrs := []neighborInfo{
		{neighborID: "r1", side: SideRight},
	}
	if hasRequiredOpenings(m, 0, nbrs) {
		t.Error("tile missing right opening should fail")
	}
}

func TestHasRequiredOpenings_WithRotation(t *testing.T) {
	// Only Top is open at rotation 0. After 90° CCW, Top moves to Left.
	m := &models.TileMetadata{
		Openings: models.OpeningSummary{Top: true, Right: false, Bottom: false, Left: false},
	}
	// At rotation 1, Left should be open (original Top), Right should be closed
	nbrsLeft := []neighborInfo{{neighborID: "r1", side: SideLeft}}
	if !hasRequiredOpenings(m, 1, nbrsLeft) {
		t.Error("after 90° CCW, Left should be open")
	}
	nbrsRight := []neighborInfo{{neighborID: "r1", side: SideRight}}
	if hasRequiredOpenings(m, 1, nbrsRight) {
		t.Error("after 90° CCW, Right should be closed")
	}
}

// --- AssignTiles integration tests ---

// makeMockMetadata creates a small set of tile metadata for testing.
func makeMockMetadata() []*models.TileMetadata {
	// A room tile: open on all 4 sides
	roomTile := &models.TileMetadata{
		TileID:    "tile_room",
		Role:      models.TileRoleRoom,
		ThemeTags: []string{"dungeon"},
		Openings:  models.OpeningSummary{Top: true, Right: true, Bottom: true, Left: true},
		EdgeSignatures: models.EdgeSignatures{
			Top: "011110", Right: "011110", Bottom: "011110", Left: "011110",
			R1Top: "011110", R1Right: "011110", R1Bottom: "011110", R1Left: "011110",
			R2Top: "011110", R2Right: "011110", R2Bottom: "011110", R2Left: "011110",
			R3Top: "011110", R3Right: "011110", R3Bottom: "011110", R3Left: "011110",
		},
	}

	// A corridor tile: open on left and right
	corridorH := &models.TileMetadata{
		TileID:    "tile_corridor_h",
		Role:      models.TileRoleCorridorH,
		ThemeTags: []string{"dungeon"},
		Openings:  models.OpeningSummary{Top: false, Right: true, Bottom: false, Left: true},
		EdgeSignatures: models.EdgeSignatures{
			Top: "000000", Right: "011110", Bottom: "000000", Left: "011110",
			R1Top: "011110", R1Right: "000000", R1Bottom: "011110", R1Left: "000000",
			R2Top: "000000", R2Right: "011110", R2Bottom: "000000", R2Left: "011110",
			R3Top: "011110", R3Right: "000000", R3Bottom: "011110", R3Left: "000000",
		},
	}

	// A corridor tile: open on top and bottom
	corridorV := &models.TileMetadata{
		TileID:    "tile_corridor_v",
		Role:      models.TileRoleCorridorV,
		ThemeTags: []string{"dungeon"},
		Openings:  models.OpeningSummary{Top: true, Right: false, Bottom: true, Left: false},
		EdgeSignatures: models.EdgeSignatures{
			Top: "011110", Right: "000000", Bottom: "011110", Left: "000000",
			R1Top: "000000", R1Right: "011110", R1Bottom: "000000", R1Left: "011110",
			R2Top: "011110", R2Right: "000000", R2Bottom: "011110", R2Left: "000000",
			R3Top: "000000", R3Right: "011110", R3Bottom: "000000", R3Left: "011110",
		},
	}

	// A dead-end tile: only open on left
	deadEnd := &models.TileMetadata{
		TileID:    "tile_dead_end",
		Role:      models.TileRoleDeadEnd,
		ThemeTags: []string{"dungeon"},
		Openings:  models.OpeningSummary{Top: false, Right: false, Bottom: false, Left: true},
		EdgeSignatures: models.EdgeSignatures{
			Top: "000000", Right: "000000", Bottom: "000000", Left: "011110",
			R1Top: "000000", R1Right: "000000", R1Bottom: "011110", R1Left: "000000",
			R2Top: "000000", R2Right: "011110", R2Bottom: "000000", R2Left: "000000",
			R3Top: "011110", R3Right: "000000", R3Bottom: "000000", R3Left: "000000",
		},
	}

	return []*models.TileMetadata{roomTile, corridorH, corridorV, deadEnd}
}

func TestAssignTiles_AllRoomsAssigned(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(graph, rng, SizeMedium, false)

	metadata := makeMockMetadata()
	assignments := AssignTiles(graph, metadata, []string{"dungeon"}, rng)

	assignedIDs := make(map[string]bool)
	for _, a := range assignments {
		assignedIDs[a.NodeID] = true
	}

	for _, room := range graph.Rooms {
		if !assignedIDs[room.ID] {
			t.Errorf("room %s not assigned a tile", room.ID)
		}
	}
}

func TestAssignTiles_CorridorsAssigned(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeMedium}, rng)
	AssignRoomTypes(graph, rng, SizeMedium, false)

	metadata := makeMockMetadata()
	assignments := AssignTiles(graph, metadata, []string{"dungeon"}, rng)

	assignedIDs := make(map[string]bool)
	for _, a := range assignments {
		assignedIDs[a.NodeID] = true
	}

	for _, conn := range graph.Connections {
		if !assignedIDs[conn.ID] {
			t.Errorf("connection %s not assigned a corridor tile", conn.ID)
		}
	}
}

func TestAssignTiles_ValidRotations(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeShort}, rng)
	AssignRoomTypes(graph, rng, SizeShort, false)

	metadata := makeMockMetadata()
	assignments := AssignTiles(graph, metadata, []string{"dungeon"}, rng)

	for _, a := range assignments {
		if a.Rotation < 0 || a.Rotation > 3 {
			t.Errorf("assignment %s: rotation %d outside [0,3]", a.NodeID, a.Rotation)
		}
	}
}

func TestAssignTiles_Determinism(t *testing.T) {
	metadata := makeMockMetadata()

	rng1 := rand.New(rand.NewSource(99))
	g1 := GenerateGraph(DungeonConfig{Seed: 99, Size: SizeShort}, rng1)
	AssignRoomTypes(g1, rng1, SizeShort, false)
	a1 := AssignTiles(g1, metadata, []string{"dungeon"}, rng1)

	rng2 := rand.New(rand.NewSource(99))
	g2 := GenerateGraph(DungeonConfig{Seed: 99, Size: SizeShort}, rng2)
	AssignRoomTypes(g2, rng2, SizeShort, false)
	a2 := AssignTiles(g2, metadata, []string{"dungeon"}, rng2)

	if len(a1) != len(a2) {
		t.Fatalf("determinism: assignment count differs: %d vs %d", len(a1), len(a2))
	}

	// Build maps for comparison (order may differ since map iteration is non-deterministic)
	m1 := make(map[string]TileAssignment)
	for _, a := range a1 {
		m1[a.NodeID] = a
	}
	m2 := make(map[string]TileAssignment)
	for _, a := range a2 {
		m2[a.NodeID] = a
	}

	for id, assign1 := range m1 {
		assign2, ok := m2[id]
		if !ok {
			t.Errorf("determinism: node %s in run1 but not run2", id)
			continue
		}
		if assign1.TileID != assign2.TileID || assign1.Rotation != assign2.Rotation {
			t.Errorf("determinism: node %s differs: %+v vs %+v", id, assign1, assign2)
		}
	}
}

func TestAssignTiles_EdgeSignaturesMatch(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	graph := GenerateGraph(DungeonConfig{Seed: 42, Size: SizeShort}, rng)
	AssignRoomTypes(graph, rng, SizeShort, false)

	metadata := makeMockMetadata()
	assignments := AssignTiles(graph, metadata, []string{"dungeon"}, rng)

	// Build lookup maps
	assignByID := make(map[string]TileAssignment)
	for _, a := range assignments {
		assignByID[a.NodeID] = a
	}
	metaByID := make(map[string]*models.TileMetadata)
	for _, m := range metadata {
		metaByID[m.TileID] = m
	}
	roomByID := buildRoomIndex(graph)

	// Check each connection: adjacent rooms should have matching edge signatures
	for _, conn := range graph.Connections {
		fromAssign, ok1 := assignByID[conn.FromRoomID]
		toAssign, ok2 := assignByID[conn.ToRoomID]
		if !ok1 || !ok2 {
			continue
		}

		fromMeta := metaByID[fromAssign.TileID]
		toMeta := metaByID[toAssign.TileID]
		if fromMeta == nil || toMeta == nil {
			continue
		}

		from := roomByID[conn.FromRoomID]
		to := roomByID[conn.ToRoomID]
		if from == nil || to == nil {
			continue
		}

		fromSide, toSide := connectionSides(from.GraphPosition, to.GraphPosition)
		fromSig := GetSignatureForSide(fromMeta.EdgeSignatures, fromAssign.Rotation, fromSide)
		toSig := GetSignatureForSide(toMeta.EdgeSignatures, toAssign.Rotation, toSide)

		// With our mock tiles (all using "011110"), signatures should match exactly
		// or be within hamming distance 1 (fuzzy fallback)
		if fromSig != toSig && HammingDistance(fromSig, toSig) > 1 {
			t.Errorf("connection %s: edge mismatch %s→%s sig %q vs %q (hamming %d)",
				conn.ID, conn.FromRoomID, conn.ToRoomID, fromSig, toSig,
				HammingDistance(fromSig, toSig))
		}
	}
}

func TestAssignTiles_MultipleSizes(t *testing.T) {
	metadata := makeMockMetadata()

	for _, size := range []DungeonSize{SizeShort, SizeMedium, SizeLong} {
		t.Run(string(size), func(t *testing.T) {
			for seed := int64(0); seed < 5; seed++ {
				rng := rand.New(rand.NewSource(seed))
				graph := GenerateGraph(DungeonConfig{Seed: seed, Size: size}, rng)
				AssignRoomTypes(graph, rng, size, false)

				assignments := AssignTiles(graph, metadata, []string{"dungeon"}, rng)

				totalExpected := len(graph.Rooms) + len(graph.Connections)
				if len(assignments) != totalExpected {
					t.Errorf("seed %d: got %d assignments, want %d (rooms=%d + connections=%d)",
						seed, len(assignments), totalExpected,
						len(graph.Rooms), len(graph.Connections))
				}
			}
		})
	}
}

// --- buildNeighborMap ---

func TestBuildNeighborMap_Simple(t *testing.T) {
	graph := &DungeonGraph{
		Rooms: []DungeonRoom{
			{ID: "r0", GraphPosition: GraphPosition{X: 0, Y: 0}},
			{ID: "r1", GraphPosition: GraphPosition{X: 1, Y: 0}},
			{ID: "r2", GraphPosition: GraphPosition{X: 1, Y: 1}},
		},
		Connections: []RoomConnection{
			{ID: "c01", FromRoomID: "r0", ToRoomID: "r1"},
			{ID: "c12", FromRoomID: "r1", ToRoomID: "r2"},
		},
	}
	roomByID := buildRoomIndex(graph)
	nbrs := buildNeighborMap(graph, roomByID)

	// r0 has one neighbor (r1) on its right
	if len(nbrs["r0"]) != 1 {
		t.Fatalf("r0: got %d neighbors, want 1", len(nbrs["r0"]))
	}
	if nbrs["r0"][0].side != SideRight || nbrs["r0"][0].neighborID != "r1" {
		t.Errorf("r0: unexpected neighbor %+v", nbrs["r0"][0])
	}

	// r1 has two neighbors: r0 on left, r2 on top
	if len(nbrs["r1"]) != 2 {
		t.Fatalf("r1: got %d neighbors, want 2", len(nbrs["r1"]))
	}

	// r2 has one neighbor (r1) on its bottom
	if len(nbrs["r2"]) != 1 {
		t.Fatalf("r2: got %d neighbors, want 1", len(nbrs["r2"]))
	}
	if nbrs["r2"][0].side != SideBottom || nbrs["r2"][0].neighborID != "r1" {
		t.Errorf("r2: unexpected neighbor %+v", nbrs["r2"][0])
	}
}
