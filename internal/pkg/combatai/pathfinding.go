package combatai

import (
	"container/heap"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// PathfindingParams contains all inputs for A* pathfinding.
type PathfindingParams struct {
	// Grid is the walkability grid. grid[row][col] == true means passable.
	// row corresponds to CellsY, col corresponds to CellsX.
	Grid [][]bool

	// Start is the starting cell (current creature position).
	Start models.CellsCoordinates

	// Goal is the target cell.
	Goal models.CellsCoordinates

	// Occupied cells that are impassable (other creatures' positions).
	// Key: [2]int{CellsX, CellsY}. Start and goal cells are always passable
	// regardless of this set.
	Occupied map[[2]int]bool

	// BlockedEdges marks directed edges between adjacent cells that are
	// impassable (walls, closed doors). Key: [4]int{fromX, fromY, toX, toY}.
	// Both directions must be set for a bidirectional wall.
	BlockedEdges map[[4]int]bool

	// MaxCells is the maximum number of cells the path may contain
	// (movement budget = speed / 5). 0 means unlimited.
	// When set, the returned path is truncated to at most MaxCells entries.
	MaxCells int
}

// FindPath uses A* with Chebyshev heuristic to find the shortest path on an
// 8-directional grid. Returns the path as a slice of cell coordinates from
// start (exclusive) to goal (inclusive). Returns nil if no path exists.
// Returns an empty non-nil slice if start == goal.
func FindPath(params *PathfindingParams) []models.CellsCoordinates {
	if params == nil || len(params.Grid) == 0 {
		return nil
	}

	height := len(params.Grid)
	width := len(params.Grid[0])

	sx, sy := params.Start.CellsX, params.Start.CellsY
	gx, gy := params.Goal.CellsX, params.Goal.CellsY

	// Bounds check.
	if !inBounds(sx, sy, width, height) || !inBounds(gx, gy, width, height) {
		return nil
	}

	// Goal must be walkable (occupied goal is allowed).
	if !params.Grid[gy][gx] {
		return nil
	}

	// Already at goal.
	if sx == gx && sy == gy {
		return []models.CellsCoordinates{}
	}

	// A* initialization.
	type point = [2]int

	gScore := make(map[point]int)
	gScore[point{sx, sy}] = 0

	cameFrom := make(map[point]point)

	open := &nodeHeap{}
	heap.Init(open)
	heap.Push(open, &astarNode{
		x: sx, y: sy,
		g: 0,
		f: chebyshevCells(sx, sy, gx, gy),
	})

	// 8-directional neighbor offsets.
	dirs := [8][2]int{
		{0, -1}, {0, 1}, {-1, 0}, {1, 0}, // cardinal
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1}, // diagonal
	}

	goalPt := point{gx, gy}

	for open.Len() > 0 {
		cur := heap.Pop(open).(*astarNode)
		curPt := point{cur.x, cur.y}

		if curPt == goalPt {
			return reconstructPath(cameFrom, goalPt, point{sx, sy}, params.MaxCells)
		}

		// Skip if we already found a better path to this node.
		if g, ok := gScore[curPt]; ok && cur.g > g {
			continue
		}

		for _, d := range dirs {
			nx, ny := cur.x+d[0], cur.y+d[1]

			if !inBounds(nx, ny, width, height) {
				continue
			}
			if !params.Grid[ny][nx] {
				continue
			}

			nPt := point{nx, ny}

			// Occupied check (start and goal exempt).
			if nPt != goalPt && params.Occupied != nil && params.Occupied[nPt] {
				continue
			}

			// Blocked edge check.
			if params.BlockedEdges != nil && params.BlockedEdges[[4]int{cur.x, cur.y, nx, ny}] {
				continue
			}

			// Diagonal squeeze: block if BOTH adjacent cardinal cells are impassable.
			if d[0] != 0 && d[1] != 0 {
				c1x, c1y := cur.x+d[0], cur.y // horizontal neighbor
				c2x, c2y := cur.x, cur.y+d[1] // vertical neighbor
				if !cellPassable(params.Grid, c1x, c1y, width, height) &&
					!cellPassable(params.Grid, c2x, c2y, width, height) {
					continue
				}
			}

			tentativeG := cur.g + 1
			if prev, ok := gScore[nPt]; ok && tentativeG >= prev {
				continue
			}

			gScore[nPt] = tentativeG
			cameFrom[nPt] = curPt

			heap.Push(open, &astarNode{
				x: nx, y: ny,
				g: tentativeG,
				f: tentativeG + chebyshevCells(nx, ny, gx, gy),
			})
		}
	}

	return nil
}

// reconstructPath walks cameFrom from goal back to start (exclusive),
// reverses the result, and truncates to maxCells if > 0.
func reconstructPath(cameFrom map[[2]int][2]int, goal, start [2]int, maxCells int) []models.CellsCoordinates {
	var path []models.CellsCoordinates

	cur := goal
	for cur != start {
		path = append(path, models.CellsCoordinates{CellsX: cur[0], CellsY: cur[1]})
		cur = cameFrom[cur]
	}

	// Reverse to get start→goal order.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	if maxCells > 0 && len(path) > maxCells {
		path = path[:maxCells]
	}

	return path
}

// chebyshevCells returns Chebyshev distance in cells: max(|dx|, |dy|).
func chebyshevCells(ax, ay, bx, by int) int {
	dx := ax - bx
	if dx < 0 {
		dx = -dx
	}
	dy := ay - by
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// inBounds checks if (x, y) is within grid dimensions.
func inBounds(x, y, width, height int) bool {
	return x >= 0 && x < width && y >= 0 && y < height
}

// cellPassable returns true if (x, y) is in bounds and walkable.
func cellPassable(grid [][]bool, x, y, width, height int) bool {
	if !inBounds(x, y, width, height) {
		return false
	}
	return grid[y][x]
}

// astarNode represents a position in the A* open set.
type astarNode struct {
	x, y  int
	g     int // cost from start
	f     int // g + heuristic
	index int // heap index
}

// nodeHeap implements heap.Interface for A* open set (min-heap by f-score).
type nodeHeap []*astarNode

func (h nodeHeap) Len() int           { return len(h) }
func (h nodeHeap) Less(i, j int) bool { return h[i].f < h[j].f }
func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *nodeHeap) Push(x any) {
	n := x.(*astarNode)
	n.index = len(*h)
	*h = append(*h, n)
}

func (h *nodeHeap) Pop() any {
	old := *h
	n := len(old)
	node := old[n-1]
	old[n-1] = nil
	node.index = -1
	*h = old[:n-1]
	return node
}
