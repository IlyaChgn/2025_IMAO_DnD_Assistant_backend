package dungeongen

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

var edgeKeyRe = regexp.MustCompile(`^(-?\d+),(-?\d+)-(-?\d+),(-?\d+)$`)

// RotateGrid rotates a square grid by 0/90/180/270° counter-clockwise.
// rotation: 0=0°, 1=90°CCW, 2=180°, 3=270°CCW.
func RotateGrid(grid [][]int, rotation int) [][]int {
	rotation = ((rotation % 4) + 4) % 4
	if rotation == 0 || len(grid) == 0 {
		return copyGrid(grid)
	}

	size := len(grid)
	result := makeGrid(size, size)

	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			nr, nc := RotateCellCoord(r, c, rotation, size)
			result[nr][nc] = grid[r][c]
		}
	}

	return result
}

// RotateCellCoord rotates a single cell coordinate within a square grid.
// Rotation formulas (CCW):
//
//	0: (r, c) → (r, c)
//	1: (r, c) → (c, gridSize-1-r)
//	2: (r, c) → (gridSize-1-r, gridSize-1-c)
//	3: (r, c) → (gridSize-1-c, r)
func RotateCellCoord(r, c, rotation, gridSize int) (int, int) {
	rotation = ((rotation % 4) + 4) % 4
	switch rotation {
	case 1:
		return c, gridSize - 1 - r
	case 2:
		return gridSize - 1 - r, gridSize - 1 - c
	case 3:
		return gridSize - 1 - c, r
	default:
		return r, c
	}
}

// RotateEdgeKey rotates an edge key string ("r1,c1-r2,c2") and returns
// the new normalized key after rotation.
func RotateEdgeKey(key string, rotation, gridSize int) string {
	rotation = ((rotation % 4) + 4) % 4
	if rotation == 0 {
		return key
	}

	r1, c1, r2, c2, ok := parseEdgeKey(key)
	if !ok {
		return key
	}

	nr1, nc1 := RotateCellCoord(r1, c1, rotation, gridSize)
	nr2, nc2 := RotateCellCoord(r2, c2, rotation, gridSize)

	return normalizedEdgeKey(nr1, nc1, nr2, nc2)
}

// RotateEdges rotates all edges in a slice and returns new edges with rotated keys.
func RotateEdges(edges []models.SerializedEdge, rotation, gridSize int) []models.SerializedEdge {
	rotation = ((rotation % 4) + 4) % 4
	if rotation == 0 {
		result := make([]models.SerializedEdge, len(edges))
		copy(result, edges)
		return result
	}

	result := make([]models.SerializedEdge, len(edges))
	for i, e := range edges {
		result[i] = models.SerializedEdge{
			Key:       RotateEdgeKey(e.Key, rotation, gridSize),
			MoveBlock: e.MoveBlock,
			LosBlock:  e.LosBlock,
		}
	}

	return result
}

// ComputeAllRotationSignatures computes edge signatures for all 4 rotations
// of a tile and returns a fully populated EdgeSignatures struct (16 fields).
func ComputeAllRotationSignatures(walkability [][]int, edges []models.SerializedEdge) models.EdgeSignatures {
	gridSize := len(walkability)

	// Rotation 0
	r0 := ExtractEdgeSignatures(walkability, edges)

	// Rotation 1
	grid1 := RotateGrid(walkability, 1)
	edges1 := RotateEdges(edges, 1, gridSize)
	r1 := ExtractEdgeSignatures(grid1, edges1)

	// Rotation 2
	grid2 := RotateGrid(walkability, 2)
	edges2 := RotateEdges(edges, 2, gridSize)
	r2 := ExtractEdgeSignatures(grid2, edges2)

	// Rotation 3
	grid3 := RotateGrid(walkability, 3)
	edges3 := RotateEdges(edges, 3, gridSize)
	r3 := ExtractEdgeSignatures(grid3, edges3)

	return models.EdgeSignatures{
		Top: r0.Top, Right: r0.Right, Bottom: r0.Bottom, Left: r0.Left,
		R1Top: r1.Top, R1Right: r1.Right, R1Bottom: r1.Bottom, R1Left: r1.Left,
		R2Top: r2.Top, R2Right: r2.Right, R2Bottom: r2.Bottom, R2Left: r2.Left,
		R3Top: r3.Top, R3Right: r3.Right, R3Bottom: r3.Bottom, R3Left: r3.Left,
	}
}

func parseEdgeKey(key string) (r1, c1, r2, c2 int, ok bool) {
	m := edgeKeyRe.FindStringSubmatch(key)
	if m == nil {
		return 0, 0, 0, 0, false
	}

	r1, _ = strconv.Atoi(m[1])
	c1, _ = strconv.Atoi(m[2])
	r2, _ = strconv.Atoi(m[3])
	c2, _ = strconv.Atoi(m[4])

	return r1, c1, r2, c2, true
}

func copyGrid(grid [][]int) [][]int {
	result := make([][]int, len(grid))
	for i, row := range grid {
		result[i] = make([]int, len(row))
		copy(result[i], row)
	}
	return result
}

func makeGrid(rows, cols int) [][]int {
	grid := make([][]int, rows)
	for i := range grid {
		grid[i] = make([]int, cols)
	}
	return grid
}

// formatEdgeKey is a convenience alias used in tests — same as normalizedEdgeKey.
func formatEdgeKey(r1, c1, r2, c2 int) string {
	return fmt.Sprintf("%d,%d-%d,%d", r1, c1, r2, c2)
}
