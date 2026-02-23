package combatai

import (
	"math"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// FindAoETargets returns the InstanceIDs of alive enemies that would be hit
// by an optimally-placed instance of the given area of effect.
// Uses NPC position as origin for "self"-origin shapes (cone, line).
// Tries each enemy position as candidate center for "point"-origin shapes (sphere, cube, cylinder).
// Returns nil if area is nil, enemies is empty, or NPC coordinates are nil.
func FindAoETargets(npcCoords *models.CellsCoordinates, area *models.AreaOfEffect,
	enemies []*models.ParticipantFull) []string {
	if area == nil || len(enemies) == 0 || npcCoords == nil {
		return nil
	}

	// Filter to enemies that have grid coordinates.
	var positioned []*models.ParticipantFull
	for _, e := range enemies {
		if e.CellsCoords != nil {
			positioned = append(positioned, e)
		}
	}
	if len(positioned) == 0 {
		return nil
	}

	switch area.Shape {
	case models.AreaShapeSphere, models.AreaShapeCylinder:
		return bestPointOriginTargets(npcCoords, area.Size, positioned, chebyshevCheck)
	case models.AreaShapeCone:
		return bestConeTargets(npcCoords, area.Size, positioned)
	case models.AreaShapeCube:
		return bestPointOriginTargets(npcCoords, area.Size, positioned, cubeCheck)
	case models.AreaShapeLine:
		width := area.Width
		if width <= 0 {
			width = 5 // D&D default line width
		}
		return bestLineTargets(npcCoords, area.Size, width, positioned)
	default:
		return nil
	}
}

// containsCheck is a function that decides whether a target is within an area
// centered at the given point with the given size (in feet).
type containsCheck func(center, target *models.CellsCoordinates, sizeFt int) bool

// chebyshevCheck returns true if the target is within sizeFt (Chebyshev distance).
// Used for sphere and cylinder shapes.
func chebyshevCheck(center, target *models.CellsCoordinates, sizeFt int) bool {
	return DistanceFt(center, target) <= sizeFt
}

// cubeCheck returns true if the target is within a cube of side length sizeFt
// centered at the given point. Both axes must be within sizeFt/2.
func cubeCheck(center, target *models.CellsCoordinates, sizeFt int) bool {
	halfCells := float64(sizeFt) / 10.0 // half-side in cells (sizeFt/2 / 5)
	dx := math.Abs(float64(target.CellsX - center.CellsX))
	dy := math.Abs(float64(target.CellsY - center.CellsY))
	return dx <= halfCells && dy <= halfCells
}

// bestPointOriginTargets finds the optimal center point (among enemy positions)
// that captures the most enemies using the given containment check.
func bestPointOriginTargets(npcCoords *models.CellsCoordinates, sizeFt int,
	enemies []*models.ParticipantFull, check containsCheck) []string {
	var bestIDs []string

	for _, candidate := range enemies {
		var ids []string
		for _, e := range enemies {
			if check(candidate.CellsCoords, e.CellsCoords, sizeFt) {
				ids = append(ids, e.InstanceID)
			}
		}
		if len(ids) > len(bestIDs) {
			bestIDs = ids
		}
	}

	return bestIDs
}

// bestConeTargets finds the optimal direction (toward each enemy) for a cone
// of the given length originating from the NPC, and returns the target set
// that captures the most enemies.
//
// D&D 5e cone rule: at distance d from origin, the cone's full width equals d.
// Geometry check: along > 0, along*5 <= length, perp <= along/2.
func bestConeTargets(origin *models.CellsCoordinates, lengthFt int,
	enemies []*models.ParticipantFull) []string {
	var bestIDs []string

	for _, ref := range enemies {
		// Direction from NPC toward this reference enemy.
		dx := float64(ref.CellsCoords.CellsX - origin.CellsX)
		dy := float64(ref.CellsCoords.CellsY - origin.CellsY)
		dirLen := math.Sqrt(dx*dx + dy*dy)
		if dirLen < 0.001 {
			continue // reference is on the NPC — can't form a direction
		}
		ndx := dx / dirLen
		ndy := dy / dirLen

		var ids []string
		for _, e := range enemies {
			vx := float64(e.CellsCoords.CellsX - origin.CellsX)
			vy := float64(e.CellsCoords.CellsY - origin.CellsY)

			along := vx*ndx + vy*ndy
			if along <= 0 {
				continue // behind the NPC
			}
			if along*5 > float64(lengthFt) {
				continue // beyond cone length
			}
			perp := math.Abs(vx*ndy - vy*ndx)
			if perp <= along/2 {
				ids = append(ids, e.InstanceID)
			}
		}
		if len(ids) > len(bestIDs) {
			bestIDs = ids
		}
	}

	return bestIDs
}

// bestLineTargets finds the optimal direction for a line of the given length
// and width originating from the NPC. Returns the target set that captures
// the most enemies.
func bestLineTargets(origin *models.CellsCoordinates, lengthFt, widthFt int,
	enemies []*models.ParticipantFull) []string {
	var bestIDs []string
	halfWidthCells := float64(widthFt) / 10.0 // half-width in cells

	for _, ref := range enemies {
		dx := float64(ref.CellsCoords.CellsX - origin.CellsX)
		dy := float64(ref.CellsCoords.CellsY - origin.CellsY)
		dirLen := math.Sqrt(dx*dx + dy*dy)
		if dirLen < 0.001 {
			continue
		}
		ndx := dx / dirLen
		ndy := dy / dirLen

		var ids []string
		for _, e := range enemies {
			vx := float64(e.CellsCoords.CellsX - origin.CellsX)
			vy := float64(e.CellsCoords.CellsY - origin.CellsY)

			along := vx*ndx + vy*ndy
			if along <= 0 {
				continue
			}
			if along*5 > float64(lengthFt) {
				continue
			}
			perp := math.Abs(vx*ndy - vy*ndx)
			if perp <= halfWidthCells {
				ids = append(ids, e.InstanceID)
			}
		}
		if len(ids) > len(bestIDs) {
			bestIDs = ids
		}
	}

	return bestIDs
}
