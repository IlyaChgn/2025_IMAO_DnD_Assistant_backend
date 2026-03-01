package dungeongen

import (
	"fmt"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// Side represents one of the four sides of a tile.
type Side int

const (
	SideTop Side = iota
	SideRight
	SideBottom
	SideLeft
)

// ExtractEdgeSignatures computes 6-bit binary signature strings for each side
// of a tile from its walkability grid and edges.
// Only fills the rotation-0 fields (Top, Right, Bottom, Left).
// Rotated signatures (R1*, R2*, R3*) are computed by rotation.go (task A3).
func ExtractEdgeSignatures(walkability [][]int, edges []models.SerializedEdge) models.EdgeSignatures {
	edgeIndex := buildEdgeIndex(edges)

	return models.EdgeSignatures{
		Top:    extractSideSignature(walkability, edgeIndex, SideTop),
		Right:  extractSideSignature(walkability, edgeIndex, SideRight),
		Bottom: extractSideSignature(walkability, edgeIndex, SideBottom),
		Left:   extractSideSignature(walkability, edgeIndex, SideLeft),
	}
}

// extractSideSignature returns a binary string (e.g. "001100") representing
// the walkability of boundary cells along the given side.
// A cell is passable ('1') if walkability == 1 AND no move-blocking edge
// exists between it and the cell just outside the tile boundary.
func extractSideSignature(walkability [][]int, edgeIndex map[string]bool, side Side) string {
	rows := len(walkability)
	if rows == 0 {
		return ""
	}
	cols := len(walkability[0])

	var cells int
	switch side {
	case SideTop, SideBottom:
		cells = cols
	case SideLeft, SideRight:
		cells = rows
	}

	var sb strings.Builder
	sb.Grow(cells)

	for i := 0; i < cells; i++ {
		var r, c int
		// Coordinates of the boundary cell
		switch side {
		case SideTop:
			r, c = 0, i
		case SideBottom:
			r, c = rows-1, i
		case SideLeft:
			r, c = i, 0
		case SideRight:
			r, c = i, cols-1
		}

		if walkability[r][c] == 0 {
			sb.WriteByte('0')
			continue
		}

		// Check if there's a move-blocking edge between this cell and
		// the adjacent cell just outside the tile boundary.
		var outerR, outerC int
		switch side {
		case SideTop:
			outerR, outerC = r-1, c
		case SideBottom:
			outerR, outerC = r+1, c
		case SideLeft:
			outerR, outerC = r, c-1
		case SideRight:
			outerR, outerC = r, c+1
		}

		key := normalizedEdgeKey(r, c, outerR, outerC)
		if edgeIndex[key] {
			sb.WriteByte('0')
		} else {
			sb.WriteByte('1')
		}
	}

	return sb.String()
}

// buildEdgeIndex creates a set of normalized edge keys that have moveBlock=true.
func buildEdgeIndex(edges []models.SerializedEdge) map[string]bool {
	idx := make(map[string]bool, len(edges))
	for _, e := range edges {
		if e.MoveBlock {
			idx[e.Key] = true
		}
	}
	return idx
}

// normalizedEdgeKey returns the canonical "r1,c1-r2,c2" key for an edge
// between two cells, with the lexicographically smaller cell first.
func normalizedEdgeKey(r1, c1, r2, c2 int) string {
	if r1 > r2 || (r1 == r2 && c1 > c2) {
		r1, c1, r2, c2 = r2, c2, r1, c1
	}
	return fmt.Sprintf("%d,%d-%d,%d", r1, c1, r2, c2)
}
