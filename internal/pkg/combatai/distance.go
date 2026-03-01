package combatai

import (
	"math"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// DistanceFt returns the Chebyshev distance between two grid positions in feet.
// Each cell is 5 feet. Returns math.MaxInt32 if either coordinate is nil.
func DistanceFt(a, b *models.CellsCoordinates) int {
	if a == nil || b == nil {
		return math.MaxInt32
	}

	dx := a.CellsX - b.CellsX
	if dx < 0 {
		dx = -dx
	}

	dy := a.CellsY - b.CellsY
	if dy < 0 {
		dy = -dy
	}

	d := dx
	if dy > d {
		d = dy
	}

	return d * 5
}
