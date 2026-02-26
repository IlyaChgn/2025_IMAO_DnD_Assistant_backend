package dungeongen

import (
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// ClassifyTile runs a 5-step classification algorithm on tile data and returns
// the tile's role, opening summary, and walkable ratio.
func ClassifyTile(walkability [][]int, edges []models.SerializedEdge) (models.TileRole, models.OpeningSummary, float64) {
	// Step 1: Extract edge signatures (rotation 0 only — sufficient for classification)
	sigs := ExtractEdgeSignatures(walkability, edges)

	// Step 2: Detect openings
	openings := detectOpenings(sigs)

	// Step 3: Compute walkable ratio
	ratio := computeWalkableRatio(walkability)

	// Step 3+: Classify by walkableRatio + openSideCount
	openCount := countOpenSides(openings)
	role := classifyByRatioAndOpenings(ratio, openCount, openings)

	return role, openings, ratio
}

// detectOpenings checks each side's signature for any '1' bit.
func detectOpenings(sigs models.EdgeSignatures) models.OpeningSummary {
	return models.OpeningSummary{
		Top:    strings.Contains(sigs.Top, "1"),
		Right:  strings.Contains(sigs.Right, "1"),
		Bottom: strings.Contains(sigs.Bottom, "1"),
		Left:   strings.Contains(sigs.Left, "1"),
	}
}

// computeWalkableRatio returns the fraction of walkable cells (value == 1).
func computeWalkableRatio(walkability [][]int) float64 {
	total := 0
	walkable := 0

	for _, row := range walkability {
		for _, cell := range row {
			total++
			if cell == 1 {
				walkable++
			}
		}
	}

	if total == 0 {
		return 0
	}

	return float64(walkable) / float64(total)
}

// countOpenSides counts how many sides have an opening.
func countOpenSides(o models.OpeningSummary) int {
	count := 0
	if o.Top {
		count++
	}
	if o.Right {
		count++
	}
	if o.Bottom {
		count++
	}
	if o.Left {
		count++
	}
	return count
}

// classifyByRatioAndOpenings implements steps 3–5 of the classification algorithm.
func classifyByRatioAndOpenings(ratio float64, openCount int, openings models.OpeningSummary) models.TileRole {
	// Step 3a: Nearly empty tiles
	if ratio < 0.1 {
		return models.TileRoleWall
	}

	// Step 3b: Very open tiles with many openings
	if ratio > 0.8 && openCount >= 3 {
		return models.TileRoleOpen
	}

	switch openCount {
	case 4:
		return models.TileRoleJunctionX
	case 3:
		return models.TileRoleJunctionT
	case 2:
		// Step 4: Corridor refinement
		return refineTwoOpenings(openings)
	case 1:
		return models.TileRoleDeadEnd
	case 0:
		// Enclosed but has some walkable area
		if ratio >= 0.1 {
			return models.TileRoleRoom
		}
		return models.TileRoleWall
	}

	// Step 5: Default
	return models.TileRoleRoom
}

// refineTwoOpenings distinguishes corridor_h, corridor_v, and corner
// when exactly 2 sides are open.
func refineTwoOpenings(o models.OpeningSummary) models.TileRole {
	if o.Left && o.Right {
		return models.TileRoleCorridorH
	}
	if o.Top && o.Bottom {
		return models.TileRoleCorridorV
	}
	// Adjacent sides open → corner
	return models.TileRoleCorner
}
