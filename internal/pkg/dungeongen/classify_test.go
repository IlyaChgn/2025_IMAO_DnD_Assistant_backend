package dungeongen

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestClassifyTile_Wall(t *testing.T) {
	// All blocked → wall
	walkability := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	role, openings, ratio := ClassifyTile(walkability, nil)

	if role != models.TileRoleWall {
		t.Errorf("role: got %q, want %q", role, models.TileRoleWall)
	}
	if ratio != 0 {
		t.Errorf("ratio: got %f, want 0", ratio)
	}
	if openings.Top || openings.Right || openings.Bottom || openings.Left {
		t.Errorf("expected no openings, got %+v", openings)
	}
}

func TestClassifyTile_JunctionX(t *testing.T) {
	// 4 openings, moderate walkable ratio
	walkability := [][]int{
		{0, 0, 1, 1, 0, 0},
		{0, 1, 1, 1, 1, 0},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{0, 1, 1, 1, 1, 0},
		{0, 0, 1, 1, 0, 0},
	}

	role, openings, _ := ClassifyTile(walkability, nil)

	if role != models.TileRoleJunctionX {
		t.Errorf("role: got %q, want %q", role, models.TileRoleJunctionX)
	}
	if !openings.Top || !openings.Right || !openings.Bottom || !openings.Left {
		t.Errorf("expected all 4 openings, got %+v", openings)
	}
}

func TestClassifyTile_Open(t *testing.T) {
	// Very high walkable ratio + 3+ openings → open
	walkability := [][]int{
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
	}

	role, _, ratio := ClassifyTile(walkability, nil)

	if role != models.TileRoleOpen {
		t.Errorf("role: got %q, want %q", role, models.TileRoleOpen)
	}
	if ratio != 1.0 {
		t.Errorf("ratio: got %f, want 1.0", ratio)
	}
}

func TestClassifyTile_JunctionT(t *testing.T) {
	// 3 openings: top, left, right (T-junction)
	walkability := [][]int{
		{0, 0, 1, 1, 0, 0},
		{0, 1, 1, 1, 1, 0},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	role, openings, _ := ClassifyTile(walkability, nil)

	if role != models.TileRoleJunctionT {
		t.Errorf("role: got %q, want %q", role, models.TileRoleJunctionT)
	}
	if !openings.Top || !openings.Right || !openings.Left {
		t.Errorf("expected top+right+left open, got %+v", openings)
	}
	if openings.Bottom {
		t.Errorf("expected bottom closed, got %+v", openings)
	}
}

func TestClassifyTile_CorridorH(t *testing.T) {
	// Horizontal corridor: left+right open
	walkability := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{1, 1, 1, 1, 1, 1},
		{1, 1, 1, 1, 1, 1},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	role, openings, _ := ClassifyTile(walkability, nil)

	if role != models.TileRoleCorridorH {
		t.Errorf("role: got %q, want %q", role, models.TileRoleCorridorH)
	}
	if !openings.Left || !openings.Right {
		t.Errorf("expected left+right open, got %+v", openings)
	}
}

func TestClassifyTile_CorridorV(t *testing.T) {
	// Vertical corridor: top+bottom open
	walkability := [][]int{
		{0, 0, 1, 1, 0, 0},
		{0, 0, 1, 1, 0, 0},
		{0, 0, 1, 1, 0, 0},
		{0, 0, 1, 1, 0, 0},
		{0, 0, 1, 1, 0, 0},
		{0, 0, 1, 1, 0, 0},
	}

	role, openings, _ := ClassifyTile(walkability, nil)

	if role != models.TileRoleCorridorV {
		t.Errorf("role: got %q, want %q", role, models.TileRoleCorridorV)
	}
	if !openings.Top || !openings.Bottom {
		t.Errorf("expected top+bottom open, got %+v", openings)
	}
}

func TestClassifyTile_Corner(t *testing.T) {
	// Corner: top+right open (adjacent sides)
	walkability := [][]int{
		{0, 0, 1, 1, 0, 0},
		{0, 0, 1, 1, 1, 0},
		{0, 0, 1, 1, 1, 1},
		{0, 0, 1, 1, 1, 1},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	role, openings, _ := ClassifyTile(walkability, nil)

	if role != models.TileRoleCorner {
		t.Errorf("role: got %q, want %q", role, models.TileRoleCorner)
	}
	if !openings.Top || !openings.Right {
		t.Errorf("expected top+right open, got %+v", openings)
	}
	if openings.Bottom || openings.Left {
		t.Errorf("expected bottom+left closed, got %+v", openings)
	}
}

func TestClassifyTile_DeadEnd(t *testing.T) {
	// Dead end: only right side open
	walkability := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 1, 1, 1, 1},
		{0, 0, 1, 1, 1, 1},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}

	role, openings, _ := ClassifyTile(walkability, nil)

	if role != models.TileRoleDeadEnd {
		t.Errorf("role: got %q, want %q", role, models.TileRoleDeadEnd)
	}
	if !openings.Right {
		t.Errorf("expected right open, got %+v", openings)
	}
}

func TestClassifyTile_Room(t *testing.T) {
	// Enclosed room: walkable interior but no boundary openings
	walkability := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 1, 1, 1, 1, 0},
		{0, 0, 0, 0, 0, 0},
	}

	role, openings, ratio := ClassifyTile(walkability, nil)

	if role != models.TileRoleRoom {
		t.Errorf("role: got %q, want %q", role, models.TileRoleRoom)
	}
	if openings.Top || openings.Right || openings.Bottom || openings.Left {
		t.Errorf("expected no openings, got %+v", openings)
	}
	// 16 walkable out of 36 ≈ 0.444
	expectedRatio := 16.0 / 36.0
	if ratio < expectedRatio-0.01 || ratio > expectedRatio+0.01 {
		t.Errorf("ratio: got %f, want ~%f", ratio, expectedRatio)
	}
}

func TestComputeWalkableRatio(t *testing.T) {
	cases := []struct {
		name     string
		grid     [][]int
		expected float64
	}{
		{"empty", [][]int{}, 0},
		{"all zeros", [][]int{{0, 0}, {0, 0}}, 0},
		{"all ones", [][]int{{1, 1}, {1, 1}}, 1.0},
		{"half", [][]int{{1, 0}, {1, 0}}, 0.5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := computeWalkableRatio(tc.grid)
			if result < tc.expected-0.001 || result > tc.expected+0.001 {
				t.Errorf("got %f, want %f", result, tc.expected)
			}
		})
	}
}

func TestCountOpenSides(t *testing.T) {
	cases := []struct {
		openings models.OpeningSummary
		expected int
	}{
		{models.OpeningSummary{}, 0},
		{models.OpeningSummary{Top: true}, 1},
		{models.OpeningSummary{Top: true, Bottom: true}, 2},
		{models.OpeningSummary{Top: true, Right: true, Left: true}, 3},
		{models.OpeningSummary{Top: true, Right: true, Bottom: true, Left: true}, 4},
	}

	for _, tc := range cases {
		result := countOpenSides(tc.openings)
		if result != tc.expected {
			t.Errorf("openings %+v: got %d, want %d", tc.openings, result, tc.expected)
		}
	}
}
