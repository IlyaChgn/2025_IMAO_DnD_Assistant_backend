package dungeongen

import (
	"reflect"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestRotateCellCoord_Identity(t *testing.T) {
	r, c := RotateCellCoord(2, 3, 0, 6)
	if r != 2 || c != 3 {
		t.Errorf("rotation 0: got (%d,%d), want (2,3)", r, c)
	}
}

func TestRotateCellCoord_90CCW(t *testing.T) {
	// (0,0) → (0,5), (0,5) → (5,5), (5,5) → (5,0), (5,0) → (0,0)
	cases := [][4]int{
		{0, 0, 0, 5},
		{0, 5, 5, 5},
		{5, 5, 5, 0},
		{5, 0, 0, 0},
	}
	for _, tc := range cases {
		r, c := RotateCellCoord(tc[0], tc[1], 1, 6)
		if r != tc[2] || c != tc[3] {
			t.Errorf("rotate(%d,%d) 90°CCW: got (%d,%d), want (%d,%d)",
				tc[0], tc[1], r, c, tc[2], tc[3])
		}
	}
}

func TestRotateCellCoord_180(t *testing.T) {
	r, c := RotateCellCoord(0, 0, 2, 6)
	if r != 5 || c != 5 {
		t.Errorf("rotation 2: got (%d,%d), want (5,5)", r, c)
	}
}

func TestRotateCellCoord_270CCW(t *testing.T) {
	r, c := RotateCellCoord(0, 0, 3, 6)
	if r != 5 || c != 0 {
		t.Errorf("rotation 3: got (%d,%d), want (5,0)", r, c)
	}
}

func TestRotateCellCoord_FullCycle(t *testing.T) {
	r, c := 2, 3
	for i := 0; i < 4; i++ {
		r, c = RotateCellCoord(r, c, 1, 6)
	}
	if r != 2 || c != 3 {
		t.Errorf("4×90°: got (%d,%d), want (2,3)", r, c)
	}
}

func TestRotateGrid_Identity(t *testing.T) {
	grid := [][]int{
		{1, 0},
		{0, 1},
	}
	result := RotateGrid(grid, 0)
	if !reflect.DeepEqual(result, grid) {
		t.Errorf("rotation 0 should be identity")
	}
}

func TestRotateGrid_90CCW(t *testing.T) {
	grid := [][]int{
		{1, 2},
		{3, 4},
	}
	// 90° CCW: top-right corner goes to top-left
	// (0,0)→(0,1), (0,1)→(1,1), (1,0)→(0,0), (1,1)→(1,0)
	// result[0][1]=1, result[1][1]=2, result[0][0]=3, result[1][0]=4
	expected := [][]int{
		{3, 1},
		{4, 2},
	}
	result := RotateGrid(grid, 1)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("rotation 1: got %v, want %v", result, expected)
	}
}

func TestRotateGrid_360(t *testing.T) {
	grid := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	result := RotateGrid(grid, 4) // 4 = full cycle
	if !reflect.DeepEqual(result, grid) {
		t.Errorf("rotation 4 (360°) should be identity, got %v", result)
	}
}

func TestRotateGrid_SequentialRotations(t *testing.T) {
	grid := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	// Rotating 4 times by 90° should equal identity
	result := grid
	for i := 0; i < 4; i++ {
		result = RotateGrid(result, 1)
	}
	if !reflect.DeepEqual(result, grid) {
		t.Errorf("4 sequential 90° rotations should be identity, got %v", result)
	}
}

func TestRotateEdgeKey_Identity(t *testing.T) {
	key := "0,0-0,1"
	result := RotateEdgeKey(key, 0, 6)
	if result != key {
		t.Errorf("rotation 0: got %q, want %q", result, key)
	}
}

func TestRotateEdgeKey_90CCW(t *testing.T) {
	// Edge (0,0)-(0,1) rotated 90° CCW in 6×6:
	// (0,0) → (0,5), (0,1) → (1,5)
	// Normalized: "0,5-1,5"
	result := RotateEdgeKey("0,0-0,1", 1, 6)
	if result != "0,5-1,5" {
		t.Errorf("got %q, want %q", result, "0,5-1,5")
	}
}

func TestRotateEdgeKey_FullCycle(t *testing.T) {
	key := "2,3-2,4"
	result := key
	for i := 0; i < 4; i++ {
		result = RotateEdgeKey(result, 1, 6)
	}
	if result != key {
		t.Errorf("4×90°: got %q, want %q", result, key)
	}
}

func TestRotateEdges(t *testing.T) {
	edges := []models.SerializedEdge{
		{Key: "0,0-0,1", MoveBlock: true, LosBlock: false},
		{Key: "2,3-3,3", MoveBlock: false, LosBlock: true},
	}

	rotated := RotateEdges(edges, 1, 6)

	if len(rotated) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(rotated))
	}
	// First edge: (0,0)-(0,1) → (0,5)-(1,5) = "0,5-1,5"
	if rotated[0].Key != "0,5-1,5" {
		t.Errorf("edge 0: got key %q, want %q", rotated[0].Key, "0,5-1,5")
	}
	if !rotated[0].MoveBlock || rotated[0].LosBlock {
		t.Errorf("edge 0: properties should be preserved")
	}
}

func TestComputeAllRotationSignatures_Symmetric(t *testing.T) {
	// Symmetric tile: openings at center of all 4 sides
	walkability := [][]int{
		{0, 0, 1, 1, 0, 0},
		{0, 1, 1, 1, 1, 0},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{0, 1, 1, 1, 1, 0},
		{0, 0, 1, 1, 0, 0},
	}

	sigs := ComputeAllRotationSignatures(walkability, nil)

	// All rotations should produce same signatures for a symmetric tile
	expected := "001100"
	for _, s := range []string{
		sigs.Top, sigs.Right, sigs.Bottom, sigs.Left,
		sigs.R1Top, sigs.R1Right, sigs.R1Bottom, sigs.R1Left,
		sigs.R2Top, sigs.R2Right, sigs.R2Bottom, sigs.R2Left,
		sigs.R3Top, sigs.R3Right, sigs.R3Bottom, sigs.R3Left,
	} {
		if s != expected {
			t.Errorf("expected all %q for symmetric tile, got %q", expected, s)
		}
	}
}

func TestComputeAllRotationSignatures_Corridor(t *testing.T) {
	// Horizontal corridor: open left+right, closed top+bottom
	walkability := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	sigs := ComputeAllRotationSignatures(walkability, nil)

	// Rotation 0: top=000000, right=001100, bottom=000000, left=001100
	if sigs.Top != "000000" || sigs.Bottom != "000000" {
		t.Errorf("R0: top=%q bottom=%q, want 000000", sigs.Top, sigs.Bottom)
	}
	if sigs.Right != "001100" || sigs.Left != "001100" {
		t.Errorf("R0: right=%q left=%q, want 001100", sigs.Right, sigs.Left)
	}

	// Rotation 1 (90° CCW): horizontal corridor becomes vertical
	// After rotation, openings should be on top+bottom
	if sigs.R1Top != "001100" || sigs.R1Bottom != "001100" {
		t.Errorf("R1: top=%q bottom=%q, want 001100", sigs.R1Top, sigs.R1Bottom)
	}
	if sigs.R1Left != "000000" || sigs.R1Right != "000000" {
		t.Errorf("R1: left=%q right=%q, want 000000", sigs.R1Left, sigs.R1Right)
	}
}
