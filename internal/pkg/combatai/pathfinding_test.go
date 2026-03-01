package combatai

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// makeGrid creates a rows×cols walkability grid with all cells passable.
func makeGrid(rows, cols int) [][]bool {
	grid := make([][]bool, rows)
	for r := range grid {
		grid[r] = make([]bool, cols)
		for c := range grid[r] {
			grid[r][c] = true
		}
	}
	return grid
}

// blockCells marks the given (x, y) positions as unwalkable.
func blockCells(grid [][]bool, cells ...[2]int) {
	for _, c := range cells {
		grid[c[1]][c[0]] = false // grid[y][x]
	}
}

func cell(x, y int) models.CellsCoordinates {
	return models.CellsCoordinates{CellsX: x, CellsY: y}
}

func TestFindPath_SamePosition(t *testing.T) {
	t.Parallel()

	grid := makeGrid(5, 5)
	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(2, 2),
		Goal:  cell(2, 2),
	})

	if path == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(path) != 0 {
		t.Errorf("expected empty path, got length %d", len(path))
	}
}

func TestFindPath_StraightHorizontal(t *testing.T) {
	t.Parallel()

	grid := makeGrid(1, 10)
	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(0, 0),
		Goal:  cell(5, 0),
	})

	if path == nil {
		t.Fatal("expected path, got nil")
	}
	if len(path) != 5 {
		t.Errorf("expected path length 5, got %d", len(path))
	}
	// Verify final cell is the goal.
	if last := path[len(path)-1]; last.CellsX != 5 || last.CellsY != 0 {
		t.Errorf("expected last cell (5,0), got (%d,%d)", last.CellsX, last.CellsY)
	}
}

func TestFindPath_StraightVertical(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 1)
	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(0, 0),
		Goal:  cell(0, 5),
	})

	if path == nil {
		t.Fatal("expected path, got nil")
	}
	if len(path) != 5 {
		t.Errorf("expected path length 5, got %d", len(path))
	}
}

func TestFindPath_Diagonal(t *testing.T) {
	t.Parallel()

	grid := makeGrid(5, 5)
	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(0, 0),
		Goal:  cell(3, 3),
	})

	if path == nil {
		t.Fatal("expected path, got nil")
	}
	// Chebyshev: max(3,3) = 3 cells.
	if len(path) != 3 {
		t.Errorf("expected path length 3 (Chebyshev), got %d", len(path))
	}
}

func TestFindPath_ObstacleAvoidance(t *testing.T) {
	t.Parallel()

	// 5x5 grid with a wall across the middle (y=2), except (4,2) is open.
	// Start (0,0), Goal (0,4).
	grid := makeGrid(5, 5)
	blockCells(grid, [2]int{0, 2}, [2]int{1, 2}, [2]int{2, 2}, [2]int{3, 2})
	// (4,2) remains open — must go around.

	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(0, 0),
		Goal:  cell(0, 4),
	})

	if path == nil {
		t.Fatal("expected path around obstacle, got nil")
	}
	// Path must avoid the blocked row.
	for _, p := range path {
		if p.CellsY == 2 && p.CellsX <= 3 {
			t.Errorf("path passes through blocked cell (%d,%d)", p.CellsX, p.CellsY)
		}
	}
	// Should end at goal.
	if last := path[len(path)-1]; last.CellsX != 0 || last.CellsY != 4 {
		t.Errorf("expected last cell (0,4), got (%d,%d)", last.CellsX, last.CellsY)
	}
}

func TestFindPath_NoPath(t *testing.T) {
	t.Parallel()

	// Start at (1,1), completely surrounded by walls.
	grid := makeGrid(3, 3)
	blockCells(grid,
		[2]int{0, 0}, [2]int{1, 0}, [2]int{2, 0},
		[2]int{0, 1}, [2]int{2, 1},
		[2]int{0, 2}, [2]int{1, 2}, [2]int{2, 2},
	)

	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(1, 1),
		Goal:  cell(0, 0),
	})

	if path != nil {
		t.Errorf("expected nil (no path), got path of length %d", len(path))
	}
}

func TestFindPath_GoalUnwalkable(t *testing.T) {
	t.Parallel()

	grid := makeGrid(5, 5)
	blockCells(grid, [2]int{4, 4})

	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(0, 0),
		Goal:  cell(4, 4),
	})

	if path != nil {
		t.Errorf("expected nil (goal unwalkable), got path of length %d", len(path))
	}
}

func TestFindPath_OccupiedCellsAvoidance(t *testing.T) {
	t.Parallel()

	// 3-wide corridor: (0,0)→(4,0), with (2,0) occupied.
	grid := makeGrid(3, 5)
	occupied := map[[2]int]bool{{2, 0}: true}

	path := FindPath(&PathfindingParams{
		Grid:     grid,
		Start:    cell(0, 0),
		Goal:     cell(4, 0),
		Occupied: occupied,
	})

	if path == nil {
		t.Fatal("expected path around occupied cell, got nil")
	}
	for _, p := range path {
		if p.CellsX == 2 && p.CellsY == 0 {
			t.Error("path passes through occupied cell (2,0)")
		}
	}
}

func TestFindPath_OccupiedGoalAllowed(t *testing.T) {
	t.Parallel()

	grid := makeGrid(5, 5)
	occupied := map[[2]int]bool{{3, 3}: true}

	path := FindPath(&PathfindingParams{
		Grid:     grid,
		Start:    cell(0, 0),
		Goal:     cell(3, 3),
		Occupied: occupied,
	})

	if path == nil {
		t.Fatal("expected path to occupied goal, got nil")
	}
	if last := path[len(path)-1]; last.CellsX != 3 || last.CellsY != 3 {
		t.Errorf("expected last cell (3,3), got (%d,%d)", last.CellsX, last.CellsY)
	}
}

func TestFindPath_OccupiedStartAllowed(t *testing.T) {
	t.Parallel()

	grid := makeGrid(5, 5)
	// Start is marked occupied (the creature itself occupies it).
	occupied := map[[2]int]bool{{0, 0}: true}

	path := FindPath(&PathfindingParams{
		Grid:     grid,
		Start:    cell(0, 0),
		Goal:     cell(3, 3),
		Occupied: occupied,
	})

	if path == nil {
		t.Fatal("expected path from occupied start, got nil")
	}
}

func TestFindPath_EdgeBlocking(t *testing.T) {
	t.Parallel()

	// 1x3 corridor: (0,0) → (1,0) → (2,0).
	// Block edge between (0,0) and (1,0) in both directions.
	grid := makeGrid(3, 3)
	edges := map[[4]int]bool{
		{0, 0, 1, 0}: true,
		{1, 0, 0, 0}: true,
	}

	path := FindPath(&PathfindingParams{
		Grid:         grid,
		Start:        cell(0, 0),
		Goal:         cell(2, 0),
		BlockedEdges: edges,
	})

	if path == nil {
		t.Fatal("expected path around blocked edge, got nil")
	}
	// Verify path doesn't go directly (0,0)→(1,0).
	if len(path) >= 1 && path[0].CellsX == 1 && path[0].CellsY == 0 {
		t.Error("path takes blocked edge (0,0)→(1,0)")
	}
}

func TestFindPath_MaxCellsTruncation(t *testing.T) {
	t.Parallel()

	grid := makeGrid(1, 10)
	path := FindPath(&PathfindingParams{
		Grid:     grid,
		Start:    cell(0, 0),
		Goal:     cell(8, 0),
		MaxCells: 3,
	})

	if path == nil {
		t.Fatal("expected truncated path, got nil")
	}
	if len(path) != 3 {
		t.Errorf("expected path length 3 (truncated), got %d", len(path))
	}
	// First 3 cells of the straight-line path.
	for i, p := range path {
		if p.CellsX != i+1 || p.CellsY != 0 {
			t.Errorf("path[%d] = (%d,%d), want (%d,0)", i, p.CellsX, p.CellsY, i+1)
		}
	}
}

func TestFindPath_MaxCellsZeroUnlimited(t *testing.T) {
	t.Parallel()

	grid := makeGrid(1, 10)
	path := FindPath(&PathfindingParams{
		Grid:     grid,
		Start:    cell(0, 0),
		Goal:     cell(8, 0),
		MaxCells: 0,
	})

	if path == nil {
		t.Fatal("expected full path, got nil")
	}
	if len(path) != 8 {
		t.Errorf("expected path length 8, got %d", len(path))
	}
}

func TestFindPath_NilGrid(t *testing.T) {
	t.Parallel()

	path := FindPath(&PathfindingParams{
		Grid:  nil,
		Start: cell(0, 0),
		Goal:  cell(1, 1),
	})

	if path != nil {
		t.Errorf("expected nil for nil grid, got path of length %d", len(path))
	}
}

func TestFindPath_NilParams(t *testing.T) {
	t.Parallel()

	path := FindPath(nil)
	if path != nil {
		t.Errorf("expected nil for nil params, got path of length %d", len(path))
	}
}

func TestFindPath_OutOfBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start models.CellsCoordinates
		goal  models.CellsCoordinates
	}{
		{"start out of bounds", cell(-1, 0), cell(2, 2)},
		{"goal out of bounds", cell(0, 0), cell(10, 10)},
		{"both out of bounds", cell(-1, -1), cell(10, 10)},
	}

	grid := makeGrid(5, 5)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := FindPath(&PathfindingParams{
				Grid:  grid,
				Start: tt.start,
				Goal:  tt.goal,
			})

			if path != nil {
				t.Errorf("expected nil for %s, got path of length %d", tt.name, len(path))
			}
		})
	}
}

func TestFindPath_DiagonalSqueeze(t *testing.T) {
	t.Parallel()

	// 5x5 grid. Block (3,2) and (2,3) so diagonal from (2,2) to (3,3)
	// would squeeze through a solid corner. An alternate route exists.
	grid := makeGrid(5, 5)
	blockCells(grid, [2]int{3, 2}, [2]int{2, 3})

	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(2, 2),
		Goal:  cell(4, 4),
	})

	if path == nil {
		t.Fatal("expected path (longer route around squeeze), got nil")
	}
	// Verify (2,2)→(3,3) direct diagonal is NOT the first step.
	if len(path) > 0 && path[0].CellsX == 3 && path[0].CellsY == 3 {
		t.Error("path squeezes diagonally through solid corner (2,2)→(3,3)")
	}
	// Path must reach goal.
	if last := path[len(path)-1]; last.CellsX != 4 || last.CellsY != 4 {
		t.Errorf("expected last cell (4,4), got (%d,%d)", last.CellsX, last.CellsY)
	}
}

func TestFindPath_DiagonalPastSingleWall(t *testing.T) {
	t.Parallel()

	// 3x3 grid. Block only (1,0). Diagonal from (0,0) to (1,1) should be
	// allowed because (0,1) is passable (only ONE adjacent cardinal is blocked).
	grid := makeGrid(3, 3)
	blockCells(grid, [2]int{1, 0})

	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(0, 0),
		Goal:  cell(1, 1),
	})

	if path == nil {
		t.Fatal("expected diagonal path, got nil")
	}
	// Optimal: 1 step diagonal.
	if len(path) != 1 {
		t.Errorf("expected path length 1 (diagonal), got %d", len(path))
	}
}

func TestFindPath_LargeGrid(t *testing.T) {
	t.Parallel()

	// 50x50 grid with a wall at y=25, opening at x=49.
	grid := makeGrid(50, 50)
	for x := 0; x < 49; x++ {
		blockCells(grid, [2]int{x, 25})
	}

	path := FindPath(&PathfindingParams{
		Grid:  grid,
		Start: cell(0, 0),
		Goal:  cell(0, 49),
	})

	if path == nil {
		t.Fatal("expected path on large grid, got nil")
	}
	// Path must end at goal.
	if last := path[len(path)-1]; last.CellsX != 0 || last.CellsY != 49 {
		t.Errorf("expected last cell (0,49), got (%d,%d)", last.CellsX, last.CellsY)
	}
}

func TestFindPath_NilMaps(t *testing.T) {
	t.Parallel()

	grid := makeGrid(5, 5)
	path := FindPath(&PathfindingParams{
		Grid:         grid,
		Start:        cell(0, 0),
		Goal:         cell(4, 4),
		Occupied:     nil,
		BlockedEdges: nil,
	})

	if path == nil {
		t.Fatal("expected path with nil maps, got nil")
	}
	if len(path) != 4 {
		t.Errorf("expected path length 4 (Chebyshev diagonal), got %d", len(path))
	}
}

func TestFindPath_OptimalPathLength(t *testing.T) {
	t.Parallel()

	// Verify A* produces optimal (shortest) paths.
	tests := []struct {
		name    string
		start   models.CellsCoordinates
		goal    models.CellsCoordinates
		wantLen int
	}{
		{"adjacent horizontal", cell(0, 0), cell(1, 0), 1},
		{"adjacent vertical", cell(0, 0), cell(0, 1), 1},
		{"adjacent diagonal", cell(0, 0), cell(1, 1), 1},
		{"L-shape 2,1", cell(0, 0), cell(2, 1), 2},
		{"far diagonal", cell(0, 0), cell(9, 9), 9},
		{"far horizontal", cell(0, 0), cell(9, 0), 9},
	}

	grid := makeGrid(10, 10)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := FindPath(&PathfindingParams{
				Grid:  grid,
				Start: tt.start,
				Goal:  tt.goal,
			})

			if path == nil {
				t.Fatalf("expected path, got nil")
			}
			if len(path) != tt.wantLen {
				t.Errorf("expected path length %d, got %d", tt.wantLen, len(path))
			}
		})
	}
}

func TestChebyshevCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		ax, ay, bx, by int
		want           int
	}{
		{"same point", 0, 0, 0, 0, 0},
		{"horizontal", 0, 0, 5, 0, 5},
		{"vertical", 0, 0, 0, 3, 3},
		{"diagonal", 0, 0, 3, 3, 3},
		{"L-shape", 0, 0, 3, 4, 4},
		{"negative", -2, -3, 2, 1, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := chebyshevCells(tt.ax, tt.ay, tt.bx, tt.by)
			if got != tt.want {
				t.Errorf("chebyshevCells(%d,%d,%d,%d) = %d, want %d",
					tt.ax, tt.ay, tt.bx, tt.by, got, tt.want)
			}
		})
	}
}
