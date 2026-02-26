package dungeongen

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestExtractEdgeSignatures_AllWalkable(t *testing.T) {
	walkability := [][]int{
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
	}

	sigs := ExtractEdgeSignatures(walkability, nil)

	if sigs.Top != "111111" {
		t.Errorf("Top: got %q, want %q", sigs.Top, "111111")
	}
	if sigs.Right != "111111" {
		t.Errorf("Right: got %q, want %q", sigs.Right, "111111")
	}
	if sigs.Bottom != "111111" {
		t.Errorf("Bottom: got %q, want %q", sigs.Bottom, "111111")
	}
	if sigs.Left != "111111" {
		t.Errorf("Left: got %q, want %q", sigs.Left, "111111")
	}
}

func TestExtractEdgeSignatures_AllBlocked(t *testing.T) {
	walkability := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	sigs := ExtractEdgeSignatures(walkability, nil)

	if sigs.Top != "000000" {
		t.Errorf("Top: got %q, want %q", sigs.Top, "000000")
	}
	if sigs.Right != "000000" {
		t.Errorf("Right: got %q, want %q", sigs.Right, "000000")
	}
	if sigs.Bottom != "000000" {
		t.Errorf("Bottom: got %q, want %q", sigs.Bottom, "000000")
	}
	if sigs.Left != "000000" {
		t.Errorf("Left: got %q, want %q", sigs.Left, "000000")
	}
}

func TestExtractEdgeSignatures_CorridorHorizontal(t *testing.T) {
	// Horizontal corridor: rows 2-3 are walkable, rest blocked
	walkability := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	sigs := ExtractEdgeSignatures(walkability, nil)

	if sigs.Top != "000000" {
		t.Errorf("Top: got %q, want %q", sigs.Top, "000000")
	}
	if sigs.Right != "000000" {
		t.Errorf("Right: got %q, want %q", sigs.Right, "000000")
	}
	if sigs.Bottom != "000000" {
		t.Errorf("Bottom: got %q, want %q", sigs.Bottom, "000000")
	}
	if sigs.Left != "000000" {
		t.Errorf("Left: got %q, want %q", sigs.Left, "000000")
	}
}

func TestExtractEdgeSignatures_CorridorWithOpenings(t *testing.T) {
	// Corridor open on left and right (cells 2-3 of column 0 and 5)
	walkability := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	sigs := ExtractEdgeSignatures(walkability, nil)

	if sigs.Top != "000000" {
		t.Errorf("Top: got %q, want %q", sigs.Top, "000000")
	}
	if sigs.Right != "001100" {
		t.Errorf("Right: got %q, want %q", sigs.Right, "001100")
	}
	if sigs.Bottom != "000000" {
		t.Errorf("Bottom: got %q, want %q", sigs.Bottom, "000000")
	}
	if sigs.Left != "001100" {
		t.Errorf("Left: got %q, want %q", sigs.Left, "001100")
	}
}

func TestExtractEdgeSignatures_RoomWithDoorTop(t *testing.T) {
	// Room: mostly walkable, with opening at top center (cols 2-3)
	walkability := [][]int{
		{0, 0, 1, 1, 0, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 0, 0, 0, 0, 0},
	}

	sigs := ExtractEdgeSignatures(walkability, nil)

	if sigs.Top != "001100" {
		t.Errorf("Top: got %q, want %q", sigs.Top, "001100")
	}
	if sigs.Bottom != "000000" {
		t.Errorf("Bottom: got %q, want %q", sigs.Bottom, "000000")
	}
	if sigs.Left != "000000" {
		t.Errorf("Left: got %q, want %q", sigs.Left, "000000")
	}
	if sigs.Right != "000000" {
		t.Errorf("Right: got %q, want %q", sigs.Right, "000000")
	}
}

func TestExtractEdgeSignatures_WithEdgeBlocking(t *testing.T) {
	// All walkable, but an edge blocks movement at top boundary cell (0,2)
	walkability := [][]int{
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
	}

	// Edge between (0,2) and (-1,2) — blocks movement across top boundary
	edges := []models.SerializedEdge{
		{Key: "-1,2-0,2", MoveBlock: true, LosBlock: false},
	}

	sigs := ExtractEdgeSignatures(walkability, edges)

	// Cell (0,2) should be blocked at top boundary due to edge
	if sigs.Top != "110111" {
		t.Errorf("Top: got %q, want %q", sigs.Top, "110111")
	}
	// Other sides unaffected
	if sigs.Right != "111111" {
		t.Errorf("Right: got %q, want %q", sigs.Right, "111111")
	}
	if sigs.Bottom != "111111" {
		t.Errorf("Bottom: got %q, want %q", sigs.Bottom, "111111")
	}
	if sigs.Left != "111111" {
		t.Errorf("Left: got %q, want %q", sigs.Left, "111111")
	}
}

func TestExtractEdgeSignatures_LosBlockDoesNotAffectSignature(t *testing.T) {
	walkability := [][]int{
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
	}

	// LOS-only edge should NOT affect walkability signature
	edges := []models.SerializedEdge{
		{Key: "-1,2-0,2", MoveBlock: false, LosBlock: true},
	}

	sigs := ExtractEdgeSignatures(walkability, edges)

	if sigs.Top != "111111" {
		t.Errorf("Top: got %q, want %q", sigs.Top, "111111")
	}
}

func TestExtractEdgeSignatures_JunctionX(t *testing.T) {
	// All 4 sides open (openings at center of each side)
	walkability := [][]int{
		{0, 0, 1, 1, 0, 0},
		{0, 1, 1, 1, 1, 0},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{0, 1, 1, 1, 1, 0},
		{0, 0, 1, 1, 0, 0},
	}

	sigs := ExtractEdgeSignatures(walkability, nil)

	if sigs.Top != "001100" {
		t.Errorf("Top: got %q, want %q", sigs.Top, "001100")
	}
	if sigs.Right != "001100" {
		t.Errorf("Right: got %q, want %q", sigs.Right, "001100")
	}
	if sigs.Bottom != "001100" {
		t.Errorf("Bottom: got %q, want %q", sigs.Bottom, "001100")
	}
	if sigs.Left != "001100" {
		t.Errorf("Left: got %q, want %q", sigs.Left, "001100")
	}
}

func TestNormalizedEdgeKey(t *testing.T) {
	// Should always put lexicographically smaller cell first
	key1 := normalizedEdgeKey(0, 2, -1, 2)
	key2 := normalizedEdgeKey(-1, 2, 0, 2)

	if key1 != key2 {
		t.Errorf("keys should be equal: %q vs %q", key1, key2)
	}
	if key1 != "-1,2-0,2" {
		t.Errorf("expected %q, got %q", "-1,2-0,2", key1)
	}
}
