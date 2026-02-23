package combatai

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// MovementPlan is the result of movement planning.
// UseDash/UseDisengage indicate the Action slot is consumed by movement.
type MovementPlan struct {
	Movement     *MovementDecision
	UseDash      bool
	UseDisengage bool
}

// PlanMovement decides where and how an NPC should move this turn.
// Pure function — no DB or external state.
// Returns nil if no movement needed or grid data unavailable.
func PlanMovement(input *TurnInput, role CreatureRole, action *ActionDecision, rng *rand.Rand) *MovementPlan {
	// Early exits.
	if input.WalkabilityGrid == nil || input.ActiveNPC.CellsCoords == nil {
		return nil
	}
	if action == nil || len(action.TargetIDs) == 0 {
		return nil
	}

	speed := input.CreatureTemplate.Movement.Walk
	if speed <= 0 {
		return nil
	}
	maxCells := speed / 5

	npcPos := *input.ActiveNPC.CellsCoords

	// Resolve target.
	target := findParticipantByID(input.Participants, action.TargetIDs[0])
	if target == nil || target.CellsCoords == nil {
		return nil
	}
	targetPos := *target.CellsCoords

	// Resolve action details for range check.
	sa := findStructuredAction(input.CreatureTemplate, action.ActionID)
	actRange := actionRange(sa)
	melee := isMeleeAction(sa)

	occupied := buildOccupied(input.Participants, input.ActiveNPC.InstanceID)

	// Check if already in range.
	dist := DistanceFt(&npcPos, &targetPos)
	if dist <= actRange {
		// In range. Check if ranged/caster NPC is in melee threat → retreat.
		if !melee && inMeleeThreat(input) {
			return planRetreat(input, role, maxCells, occupied, rng)
		}
		return nil // no movement needed
	}

	// Not in range → approach target.
	return planApproach(input, role, npcPos, targetPos, actRange, melee, maxCells, occupied, rng)
}

// planApproach plans movement toward a target that is out of range.
func planApproach(
	input *TurnInput, role CreatureRole,
	npcPos, targetPos models.CellsCoordinates,
	actRange int, melee bool, maxCells int,
	occupied map[[2]int]bool, rng *rand.Rand,
) *MovementPlan {
	grid := input.WalkabilityGrid
	width := input.MapWidth
	height := input.MapHeight

	goal := selectApproachGoal(npcPos, targetPos, actRange, melee, grid, occupied, width, height)
	if goal == nil {
		return nil
	}

	// Find full path (unlimited).
	path := FindPath(&PathfindingParams{
		Grid:         grid,
		Start:        npcPos,
		Goal:         *goal,
		Occupied:     occupied,
		BlockedEdges: input.BlockedEdges,
	})
	if path == nil {
		return nil
	}

	needDisengage := inMeleeThreat(input)

	// Path fits in normal movement.
	if len(path) <= maxCells {
		plan := &MovementPlan{
			Movement: buildMovementDecision(path, "Approaching target"),
		}
		if needDisengage && shouldDisengage(role, input.Intelligence, rng) {
			plan.UseDisengage = true
			plan.Movement.Reasoning = "Disengaging to approach safely"
		}
		return plan
	}

	// Path too long — consider Dash.
	if shouldDash(role, input.Intelligence, rng) {
		dashMaxCells := maxCells * 2
		dashPath := FindPath(&PathfindingParams{
			Grid:         grid,
			Start:        npcPos,
			Goal:         *goal,
			Occupied:     occupied,
			BlockedEdges: input.BlockedEdges,
			MaxCells:     dashMaxCells,
		})
		if dashPath != nil && len(dashPath) > 0 {
			return &MovementPlan{
				Movement: buildMovementDecision(dashPath, "Dashing toward target"),
				UseDash:  true,
			}
		}
	}

	// No Dash — truncate to normal movement.
	truncated := path
	if len(truncated) > maxCells {
		truncated = truncated[:maxCells]
	}

	plan := &MovementPlan{
		Movement: buildMovementDecision(truncated, "Approaching target (partial)"),
	}
	if needDisengage && shouldDisengage(role, input.Intelligence, rng) {
		plan.UseDisengage = true
		plan.Movement.Reasoning = "Disengaging to approach safely"
	}
	return plan
}

// planRetreat plans movement away from melee threats for ranged/caster NPCs.
func planRetreat(
	input *TurnInput, role CreatureRole, maxCells int,
	occupied map[[2]int]bool, rng *rand.Rand,
) *MovementPlan {
	npcPos := *input.ActiveNPC.CellsCoords
	grid := input.WalkabilityGrid

	threats := meleeThreatPositions(input)
	if len(threats) == 0 {
		return nil
	}

	goal := selectRetreatGoal(npcPos, threats, maxCells, grid, occupied, input.MapWidth, input.MapHeight)
	if goal == nil {
		return nil
	}

	path := FindPath(&PathfindingParams{
		Grid:         grid,
		Start:        npcPos,
		Goal:         *goal,
		Occupied:     occupied,
		BlockedEdges: input.BlockedEdges,
		MaxCells:     maxCells,
	})
	if path == nil || len(path) == 0 {
		return nil
	}

	plan := &MovementPlan{
		Movement: buildMovementDecision(path, "Retreating from melee threat"),
	}

	if shouldDisengage(role, input.Intelligence, rng) {
		plan.UseDisengage = true
		plan.Movement.Reasoning = "Disengaging to retreat safely"
	}

	return plan
}

// selectApproachGoal picks the best cell to move toward a target.
// For melee: walkable+unoccupied cell adjacent to target, closest to NPC.
// For ranged: the NPC's current position projected toward the target boundary.
func selectApproachGoal(
	npcPos, targetPos models.CellsCoordinates,
	actRange int, melee bool,
	grid [][]bool, occupied map[[2]int]bool,
	width, height int,
) *models.CellsCoordinates {
	if melee || actRange <= 5 {
		return selectAdjacentGoal(npcPos, targetPos, grid, occupied, width, height)
	}

	// Ranged: find closest walkable cell that is within actRange of target.
	// Search expanding from NPC position toward target.
	return selectAdjacentGoal(npcPos, targetPos, grid, occupied, width, height)
}

// selectAdjacentGoal finds the best walkable, unoccupied cell adjacent to
// the target that is closest to the NPC.
func selectAdjacentGoal(
	npcPos, targetPos models.CellsCoordinates,
	grid [][]bool, occupied map[[2]int]bool,
	width, height int,
) *models.CellsCoordinates {
	dirs := [8][2]int{
		{0, -1}, {0, 1}, {-1, 0}, {1, 0},
		{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
	}

	var best *models.CellsCoordinates
	bestDist := math.MaxInt32

	tx, ty := targetPos.CellsX, targetPos.CellsY

	for _, d := range dirs {
		cx, cy := tx+d[0], ty+d[1]

		if !inBounds(cx, cy, width, height) {
			continue
		}
		if !grid[cy][cx] {
			continue
		}
		// NPC's own current cell is valid.
		if cx == npcPos.CellsX && cy == npcPos.CellsY {
			c := models.CellsCoordinates{CellsX: cx, CellsY: cy}
			return &c
		}
		if occupied[[2]int{cx, cy}] {
			continue
		}

		dist := chebyshevCells(npcPos.CellsX, npcPos.CellsY, cx, cy)
		if dist < bestDist {
			bestDist = dist
			c := models.CellsCoordinates{CellsX: cx, CellsY: cy}
			best = &c
		}
	}

	return best
}

// selectRetreatGoal picks a cell away from melee threats.
// Direction: away from average threat position, clamped to grid.
func selectRetreatGoal(
	npcPos models.CellsCoordinates, threats []models.CellsCoordinates,
	maxCells int, grid [][]bool, occupied map[[2]int]bool,
	width, height int,
) *models.CellsCoordinates {
	if len(threats) == 0 {
		return nil
	}

	// Compute average threat position.
	avgX, avgY := 0.0, 0.0
	for _, t := range threats {
		avgX += float64(t.CellsX)
		avgY += float64(t.CellsY)
	}
	avgX /= float64(len(threats))
	avgY /= float64(len(threats))

	// Direction away from threats.
	dx := float64(npcPos.CellsX) - avgX
	dy := float64(npcPos.CellsY) - avgY

	// Normalize.
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 0.001 {
		// NPC on top of threats — pick any direction.
		dx, dy = 1, 0
		length = 1
	}
	dx /= length
	dy /= length

	// Try decreasing distances from maxCells to 1.
	for dist := maxCells; dist >= 1; dist-- {
		gx := npcPos.CellsX + int(math.Round(dx*float64(dist)))
		gy := npcPos.CellsY + int(math.Round(dy*float64(dist)))

		// Clamp to grid.
		if gx < 0 {
			gx = 0
		}
		if gx >= width {
			gx = width - 1
		}
		if gy < 0 {
			gy = 0
		}
		if gy >= height {
			gy = height - 1
		}

		if !grid[gy][gx] {
			continue
		}
		if occupied[[2]int{gx, gy}] {
			continue
		}
		// Don't retreat to same position.
		if gx == npcPos.CellsX && gy == npcPos.CellsY {
			continue
		}

		c := models.CellsCoordinates{CellsX: gx, CellsY: gy}
		return &c
	}

	return nil
}

// inMeleeThreat returns true if there's an alive enemy adjacent (≤5ft) to the NPC.
func inMeleeThreat(input *TurnInput) bool {
	if input.ActiveNPC.CellsCoords == nil {
		return false
	}
	npcPos := input.ActiveNPC.CellsCoords

	for i := range input.Participants {
		p := &input.Participants[i]
		if !p.IsPlayerCharacter || !IsAlive(p) || p.CellsCoords == nil {
			continue
		}
		if DistanceFt(npcPos, p.CellsCoords) <= 5 {
			return true
		}
	}
	return false
}

// meleeThreatPositions returns positions of alive enemies adjacent (≤5ft) to the NPC.
func meleeThreatPositions(input *TurnInput) []models.CellsCoordinates {
	if input.ActiveNPC.CellsCoords == nil {
		return nil
	}
	npcPos := input.ActiveNPC.CellsCoords

	var threats []models.CellsCoordinates
	for i := range input.Participants {
		p := &input.Participants[i]
		if !p.IsPlayerCharacter || !IsAlive(p) || p.CellsCoords == nil {
			continue
		}
		if DistanceFt(npcPos, p.CellsCoords) <= 5 {
			threats = append(threats, *p.CellsCoords)
		}
	}
	return threats
}

// shouldDash returns true if the NPC should use Dash (intelligence gate).
// Brute/Tank/Skirmisher: always consider. Ranged/Caster/Controller: only when retreating.
func shouldDash(role CreatureRole, intelligence float64, rng *rand.Rand) bool {
	switch role {
	case RoleBrute, RoleTank, RoleSkirmisher:
		return rng.Float64() < intelligence
	default:
		// Ranged/Caster/Controller don't Dash to approach — only used in retreat path.
		return false
	}
}

// shouldDisengage returns true if the NPC should use Disengage (intelligence gate).
// Higher intelligence → more likely to Disengage safely.
func shouldDisengage(role CreatureRole, intelligence float64, rng *rand.Rand) bool {
	switch role {
	case RoleBrute, RoleTank:
		// Melee roles rarely Disengage — they want to stay in combat.
		return false
	default:
		return rng.Float64() < intelligence
	}
}

// buildOccupied constructs the occupied cell map from Participants.
// Excludes the given NPC (selfID) and dead participants.
func buildOccupied(participants []models.ParticipantFull, selfID string) map[[2]int]bool {
	occupied := make(map[[2]int]bool)
	for i := range participants {
		p := &participants[i]
		if p.InstanceID == selfID {
			continue
		}
		if GetCurrentHP(p) <= 0 {
			continue
		}
		if p.CellsCoords != nil {
			occupied[[2]int{p.CellsCoords.CellsX, p.CellsCoords.CellsY}] = true
		}
	}
	return occupied
}

// findParticipantByID looks up a participant by InstanceID.
func findParticipantByID(participants []models.ParticipantFull, id string) *models.ParticipantFull {
	for i := range participants {
		if participants[i].InstanceID == id {
			return &participants[i]
		}
	}
	return nil
}

// findStructuredAction looks up a StructuredAction by ID in the creature template.
// Returns nil if not found (e.g., universal actions like dodge/dash).
func findStructuredAction(creature models.Creature, actionID string) *models.StructuredAction {
	for i := range creature.StructuredActions {
		if creature.StructuredActions[i].ID == actionID {
			return &creature.StructuredActions[i]
		}
	}
	return nil
}

// buildMovementDecision creates a MovementDecision from a path.
func buildMovementDecision(path []models.CellsCoordinates, reasoning string) *MovementDecision {
	if len(path) == 0 {
		return nil
	}
	dest := path[len(path)-1]
	return &MovementDecision{
		TargetX:   dest.CellsX,
		TargetY:   dest.CellsY,
		Path:      path,
		Reasoning: fmt.Sprintf("%s → (%d,%d)", reasoning, dest.CellsX, dest.CellsY),
	}
}
