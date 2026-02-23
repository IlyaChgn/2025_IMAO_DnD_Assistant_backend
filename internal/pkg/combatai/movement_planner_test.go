package combatai

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// buildTestInput constructs a minimal TurnInput for movement planner tests.
// grid must be pre-built. npcPos and targetPos are cell coordinates.
func buildTestInput(
	grid [][]bool,
	npcPos, targetPos models.CellsCoordinates,
	npcID, targetID string,
	speed int,
	actions []models.StructuredAction,
) *TurnInput {
	height := len(grid)
	width := 0
	if height > 0 {
		width = len(grid[0])
	}

	return &TurnInput{
		ActiveNPC: models.ParticipantFull{
			InstanceID:  npcID,
			CellsCoords: &npcPos,
		},
		CreatureTemplate: models.Creature{
			Movement:          models.CreatureMovement{Walk: speed},
			StructuredActions: actions,
		},
		Participants: []models.ParticipantFull{
			{
				InstanceID:        npcID,
				CellsCoords:       &npcPos,
				RuntimeState:      models.CreatureRuntimeState{CurrentHP: 20},
				IsPlayerCharacter: false,
			},
			{
				InstanceID:        targetID,
				CellsCoords:       &targetPos,
				RuntimeState:      models.CreatureRuntimeState{CurrentHP: 20},
				IsPlayerCharacter: true,
			},
		},
		Intelligence:    0.8,
		MapWidth:        width,
		MapHeight:       height,
		WalkabilityGrid: grid,
	}
}

// meleeAction returns a simple melee StructuredAction for testing.
func meleeAction(id string) models.StructuredAction {
	return models.StructuredAction{
		ID:       id,
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollMeleeWeapon,
			Reach: 5,
		},
	}
}

// rangedAction returns a simple ranged StructuredAction for testing.
func rangedAction(id string, normalRange int) models.StructuredAction {
	return models.StructuredAction{
		ID:       id,
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollRangedWeapon,
			Range: &models.RangeData{Normal: normalRange},
		},
	}
}

func TestPlanMovement_NilGrid(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC: models.ParticipantFull{
			CellsCoords: &models.CellsCoordinates{CellsX: 0, CellsY: 0},
		},
		WalkabilityGrid: nil,
	}
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan != nil {
		t.Errorf("expected nil plan for nil grid, got %+v", plan)
	}
}

func TestPlanMovement_NilPosition(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	input := &TurnInput{
		ActiveNPC:       models.ParticipantFull{CellsCoords: nil},
		WalkabilityGrid: grid,
		MapWidth:        10,
		MapHeight:       10,
	}
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan != nil {
		t.Errorf("expected nil plan for nil position, got %+v", plan)
	}
}

func TestPlanMovement_NilAction(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	input := buildTestInput(grid, cell(0, 0), cell(5, 5), "npc-1", "pc-1", 30, nil)
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, nil, rng)
	if plan != nil {
		t.Errorf("expected nil plan for nil action, got %+v", plan)
	}
}

func TestPlanMovement_NoTargetIDs(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	input := buildTestInput(grid, cell(0, 0), cell(5, 5), "npc-1", "pc-1", 30, nil)
	action := &ActionDecision{ActionID: "slash", TargetIDs: nil}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan != nil {
		t.Errorf("expected nil plan for empty TargetIDs, got %+v", plan)
	}
}

func TestPlanMovement_ZeroSpeed(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	input := buildTestInput(grid, cell(0, 0), cell(5, 5), "npc-1", "pc-1", 0,
		[]models.StructuredAction{meleeAction("slash")})
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan != nil {
		t.Errorf("expected nil plan for zero speed, got %+v", plan)
	}
}

func TestPlanMovement_AlreadyInMeleeRange(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	// NPC at (3,3), target at (4,3) — adjacent, distance 5ft.
	input := buildTestInput(grid, cell(3, 3), cell(4, 3), "npc-1", "pc-1", 30,
		[]models.StructuredAction{meleeAction("slash")})
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan != nil {
		t.Errorf("expected nil plan when already in melee range, got %+v", plan)
	}
}

func TestPlanMovement_AlreadyInRangedRange(t *testing.T) {
	t.Parallel()

	grid := makeGrid(20, 20)
	// NPC at (0,0), target at (6,0) — distance 30ft, weapon range 80ft.
	// No enemy adjacent to NPC → no melee threat.
	input := buildTestInput(grid, cell(0, 0), cell(6, 0), "npc-1", "pc-1", 30,
		[]models.StructuredAction{rangedAction("bow", 80)})
	action := &ActionDecision{ActionID: "bow", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleRanged, action, rng)
	if plan != nil {
		t.Errorf("expected nil plan when already in ranged range, got %+v", plan)
	}
}

func TestPlanMovement_MeleeApproach(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	// NPC at (0,0), target at (4,0) — distance 20ft, reach 5ft.
	// Need to move to (3,0) which is adjacent.
	input := buildTestInput(grid, cell(0, 0), cell(4, 0), "npc-1", "pc-1", 30,
		[]models.StructuredAction{meleeAction("slash")})
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan for melee approach")
	}
	if plan.Movement == nil {
		t.Fatal("expected non-nil Movement")
	}
	if plan.UseDash {
		t.Error("expected UseDash=false for reachable target")
	}
	// Path should end adjacent to target.
	dest := plan.Movement.Path[len(plan.Movement.Path)-1]
	dist := chebyshevCells(dest.CellsX, dest.CellsY, 4, 0)
	if dist > 1 {
		t.Errorf("destination (%d,%d) not adjacent to target (4,0), dist=%d", dest.CellsX, dest.CellsY, dist)
	}
	// Path should be at most 6 cells (30ft speed).
	if len(plan.Movement.Path) > 6 {
		t.Errorf("path length %d exceeds max cells 6", len(plan.Movement.Path))
	}
}

func TestPlanMovement_MeleeApproachObstacle(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	// Wall between NPC and target.
	blockCells(grid, [2]int{2, 0}, [2]int{2, 1}, [2]int{2, 2})
	// NPC at (0,0), target at (4,0).
	input := buildTestInput(grid, cell(0, 0), cell(4, 0), "npc-1", "pc-1", 30,
		[]models.StructuredAction{meleeAction("slash")})
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan — should navigate around obstacle")
	}
	if plan.Movement == nil {
		t.Fatal("expected non-nil Movement")
	}
	// Path should be valid (all cells walkable, connected).
	for _, c := range plan.Movement.Path {
		if !grid[c.CellsY][c.CellsX] {
			t.Errorf("path passes through unwalkable cell (%d,%d)", c.CellsX, c.CellsY)
		}
	}
}

func TestPlanMovement_MeleeApproachOccupied(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	// NPC at (0,0), target at (3,0). Cell (2,0) is occupied by another NPC.
	npcPos := cell(0, 0)
	targetPos := cell(3, 0)
	blockerPos := cell(2, 0)

	input := &TurnInput{
		ActiveNPC: models.ParticipantFull{
			InstanceID:  "npc-1",
			CellsCoords: &npcPos,
		},
		CreatureTemplate: models.Creature{
			Movement:          models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{meleeAction("slash")},
		},
		Participants: []models.ParticipantFull{
			{InstanceID: "npc-1", CellsCoords: &npcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}},
			{InstanceID: "pc-1", CellsCoords: &targetPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}, IsPlayerCharacter: true},
			{InstanceID: "npc-2", CellsCoords: &blockerPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}},
		},
		Intelligence:    0.8,
		MapWidth:        10,
		MapHeight:       10,
		WalkabilityGrid: grid,
	}
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan — should navigate around occupied cell")
	}
	// Path should not pass through occupied cell (2,0).
	for _, c := range plan.Movement.Path {
		if c.CellsX == 2 && c.CellsY == 0 {
			t.Error("path passes through occupied cell (2,0)")
		}
	}
}

func TestPlanMovement_DashApproach(t *testing.T) {
	t.Parallel()

	grid := makeGrid(20, 20)
	// NPC at (0,0), target at (10,0) — distance 50ft, speed 30ft (6 cells).
	// With high intelligence, should Dash.
	input := buildTestInput(grid, cell(0, 0), cell(10, 0), "npc-1", "pc-1", 30,
		[]models.StructuredAction{meleeAction("slash")})
	input.Intelligence = 1.0 // guarantee intelligence gate passes
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan for dash approach")
	}
	if !plan.UseDash {
		t.Error("expected UseDash=true for distant target with high intelligence")
	}
	// Path should be longer than normal movement (>6 cells).
	if len(plan.Movement.Path) <= 6 {
		t.Errorf("expected path longer than 6 cells with Dash, got %d", len(plan.Movement.Path))
	}
}

func TestPlanMovement_NoDashLowIntelligence(t *testing.T) {
	t.Parallel()

	grid := makeGrid(20, 20)
	// NPC at (0,0), target at (10,0) — too far for normal movement.
	input := buildTestInput(grid, cell(0, 0), cell(10, 0), "npc-1", "pc-1", 30,
		[]models.StructuredAction{meleeAction("slash")})
	input.Intelligence = 0.0 // guarantee intelligence gate fails
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan — should partially approach")
	}
	if plan.UseDash {
		t.Error("expected UseDash=false for low intelligence")
	}
	// Path should be truncated to 6 cells (30ft speed).
	if len(plan.Movement.Path) > 6 {
		t.Errorf("path length %d exceeds max cells 6 without Dash", len(plan.Movement.Path))
	}
}

func TestPlanMovement_RangedRetreat(t *testing.T) {
	t.Parallel()

	grid := makeGrid(20, 20)
	// Ranged NPC at (5,5), PC adjacent at (5,6), weapon range 80ft.
	// NPC is in melee threat and in range → should retreat.
	npcPos := cell(5, 5)
	pcPos := cell(5, 6)
	input := &TurnInput{
		ActiveNPC: models.ParticipantFull{
			InstanceID:  "npc-1",
			CellsCoords: &npcPos,
		},
		CreatureTemplate: models.Creature{
			Movement:          models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{rangedAction("bow", 80)},
		},
		Participants: []models.ParticipantFull{
			{InstanceID: "npc-1", CellsCoords: &npcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}},
			{InstanceID: "pc-1", CellsCoords: &pcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}, IsPlayerCharacter: true},
		},
		Intelligence:    0.8,
		MapWidth:        20,
		MapHeight:       20,
		WalkabilityGrid: grid,
	}
	action := &ActionDecision{ActionID: "bow", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleRanged, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan for ranged retreat")
	}
	if plan.Movement == nil {
		t.Fatal("expected non-nil Movement")
	}
	// Should move away from PC — destination should be farther from PC than origin.
	dest := plan.Movement.Path[len(plan.Movement.Path)-1]
	origDist := chebyshevCells(5, 5, 5, 6)
	newDist := chebyshevCells(dest.CellsX, dest.CellsY, 5, 6)
	if newDist <= origDist {
		t.Errorf("retreat should increase distance from threat: orig=%d, new=%d", origDist, newDist)
	}
}

func TestPlanMovement_DisengageRetreat(t *testing.T) {
	t.Parallel()

	grid := makeGrid(20, 20)
	// Ranged NPC at (5,5), PC adjacent at (5,6), intelligence=1.0.
	npcPos := cell(5, 5)
	pcPos := cell(5, 6)
	input := &TurnInput{
		ActiveNPC: models.ParticipantFull{
			InstanceID:  "npc-1",
			CellsCoords: &npcPos,
		},
		CreatureTemplate: models.Creature{
			Movement:          models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{rangedAction("bow", 80)},
		},
		Participants: []models.ParticipantFull{
			{InstanceID: "npc-1", CellsCoords: &npcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}},
			{InstanceID: "pc-1", CellsCoords: &pcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}, IsPlayerCharacter: true},
		},
		Intelligence:    1.0, // guarantee Disengage
		MapWidth:        20,
		MapHeight:       20,
		WalkabilityGrid: grid,
	}
	action := &ActionDecision{ActionID: "bow", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleRanged, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan for disengage retreat")
	}
	if !plan.UseDisengage {
		t.Error("expected UseDisengage=true for ranged NPC with high intelligence")
	}
}

func TestPlanMovement_CasterRetreat(t *testing.T) {
	t.Parallel()

	grid := makeGrid(20, 20)
	npcPos := cell(5, 5)
	pcPos := cell(5, 6)
	input := &TurnInput{
		ActiveNPC: models.ParticipantFull{
			InstanceID:  "npc-1",
			CellsCoords: &npcPos,
		},
		CreatureTemplate: models.Creature{
			Movement:          models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{rangedAction("firebolt", 120)},
		},
		Participants: []models.ParticipantFull{
			{InstanceID: "npc-1", CellsCoords: &npcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}},
			{InstanceID: "pc-1", CellsCoords: &pcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}, IsPlayerCharacter: true},
		},
		Intelligence:    1.0,
		MapWidth:        20,
		MapHeight:       20,
		WalkabilityGrid: grid,
	}
	action := &ActionDecision{ActionID: "firebolt", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleCaster, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan for caster retreat")
	}
	if !plan.UseDisengage {
		t.Error("expected UseDisengage=true for caster")
	}
}

func TestPlanMovement_NoPath(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	// Surround the target with walls — no adjacent walkable cells.
	blockCells(grid,
		[2]int{4, 4}, [2]int{5, 4}, [2]int{6, 4},
		[2]int{4, 5}, [2]int{6, 5},
		[2]int{4, 6}, [2]int{5, 6}, [2]int{6, 6},
	)
	// Target at (5,5), NPC at (0,0).
	input := buildTestInput(grid, cell(0, 0), cell(5, 5), "npc-1", "pc-1", 30,
		[]models.StructuredAction{meleeAction("slash")})
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan != nil {
		t.Errorf("expected nil plan when no path exists, got %+v", plan)
	}
}

func TestPlanMovement_GoalCellSelection(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	// NPC at (0,5), target at (5,5). NPC should pick the closest adjacent cell.
	input := buildTestInput(grid, cell(0, 5), cell(5, 5), "npc-1", "pc-1", 30,
		[]models.StructuredAction{meleeAction("slash")})
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	// Destination should be adjacent to target (5,5).
	dest := plan.Movement.Path[len(plan.Movement.Path)-1]
	dist := chebyshevCells(dest.CellsX, dest.CellsY, 5, 5)
	if dist != 1 {
		t.Errorf("destination (%d,%d) should be exactly adjacent to target (5,5), dist=%d",
			dest.CellsX, dest.CellsY, dist)
	}
	// Should be the closest adjacent cell — (4,5).
	if dest.CellsX != 4 || dest.CellsY != 5 {
		t.Logf("Note: expected destination (4,5), got (%d,%d) — alternate valid cell", dest.CellsX, dest.CellsY)
	}
}

func TestPlanMovement_LargeGrid(t *testing.T) {
	t.Parallel()

	grid := makeGrid(30, 30)
	// Add some obstacles.
	for i := 5; i < 25; i++ {
		blockCells(grid, [2]int{15, i})
	}
	// Opening at row 4 and row 25.
	input := buildTestInput(grid, cell(0, 15), cell(29, 15), "npc-1", "pc-1", 30,
		[]models.StructuredAction{meleeAction("slash")})
	input.Intelligence = 1.0
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	// Should find a path (may Dash).
	if plan == nil {
		t.Fatal("expected non-nil plan on large grid with obstacles")
	}
	if plan.Movement == nil {
		t.Fatal("expected non-nil Movement")
	}
}

func TestPlanMovement_BruteDoesNotDisengage(t *testing.T) {
	t.Parallel()

	grid := makeGrid(20, 20)
	// Brute NPC at (5,5), target at (10,5), PC adjacent at (5,6).
	// Brute should NOT Disengage (melee role stays in combat).
	npcPos := cell(5, 5)
	targetPos := cell(10, 5)
	adjacentPCPos := cell(5, 6)
	input := &TurnInput{
		ActiveNPC: models.ParticipantFull{
			InstanceID:  "npc-1",
			CellsCoords: &npcPos,
		},
		CreatureTemplate: models.Creature{
			Movement:          models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{meleeAction("slash")},
		},
		Participants: []models.ParticipantFull{
			{InstanceID: "npc-1", CellsCoords: &npcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}},
			{InstanceID: "pc-1", CellsCoords: &targetPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}, IsPlayerCharacter: true},
			{InstanceID: "pc-2", CellsCoords: &adjacentPCPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}, IsPlayerCharacter: true},
		},
		Intelligence:    1.0,
		MapWidth:        20,
		MapHeight:       20,
		WalkabilityGrid: grid,
	}
	action := &ActionDecision{ActionID: "slash", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleBrute, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.UseDisengage {
		t.Error("Brute should NOT UseDisengage")
	}
}

func TestPlanMovement_RangedDoesNotDashToApproach(t *testing.T) {
	t.Parallel()

	grid := makeGrid(30, 30)
	// Ranged NPC at (0,0), target at (25,0) — out of range (80ft = 16 cells).
	// Ranged role should NOT Dash to approach (only melee roles Dash to approach).
	input := buildTestInput(grid, cell(0, 0), cell(25, 0), "npc-1", "pc-1", 30,
		[]models.StructuredAction{rangedAction("bow", 80)})
	input.Intelligence = 1.0
	action := &ActionDecision{ActionID: "bow", TargetIDs: []string{"pc-1"}}
	rng := rand.New(rand.NewSource(42))

	plan := PlanMovement(input, RoleRanged, action, rng)
	if plan == nil {
		t.Fatal("expected non-nil plan for partial approach")
	}
	if plan.UseDash {
		t.Error("Ranged role should NOT Dash to approach")
	}
}

// Test helper functions.

func TestBuildOccupied(t *testing.T) {
	t.Parallel()

	pos1 := models.CellsCoordinates{CellsX: 1, CellsY: 2}
	pos2 := models.CellsCoordinates{CellsX: 3, CellsY: 4}

	participants := []models.ParticipantFull{
		{InstanceID: "self", CellsCoords: &pos1, RuntimeState: models.CreatureRuntimeState{CurrentHP: 10}},
		{InstanceID: "other", CellsCoords: &pos2, RuntimeState: models.CreatureRuntimeState{CurrentHP: 10}},
		{InstanceID: "dead", CellsCoords: &models.CellsCoordinates{CellsX: 5, CellsY: 5}, RuntimeState: models.CreatureRuntimeState{CurrentHP: 0}},
		{InstanceID: "nopos", RuntimeState: models.CreatureRuntimeState{CurrentHP: 10}},
	}

	occupied := buildOccupied(participants, "self")

	// Self should NOT be in the map.
	if occupied[[2]int{1, 2}] {
		t.Error("self should not be in occupied map")
	}
	// Other alive participant should be in the map.
	if !occupied[[2]int{3, 4}] {
		t.Error("alive participant should be in occupied map")
	}
	// Dead participant should NOT be in the map.
	if occupied[[2]int{5, 5}] {
		t.Error("dead participant should not be in occupied map")
	}
}

func TestInMeleeThreat(t *testing.T) {
	t.Parallel()

	npcPos := models.CellsCoordinates{CellsX: 5, CellsY: 5}
	adjacentPC := models.CellsCoordinates{CellsX: 5, CellsY: 6}
	farPC := models.CellsCoordinates{CellsX: 10, CellsY: 10}

	tests := []struct {
		name string
		pcs  []models.CellsCoordinates
		want bool
	}{
		{"adjacent PC", []models.CellsCoordinates{adjacentPC}, true},
		{"far PC", []models.CellsCoordinates{farPC}, false},
		{"mixed", []models.CellsCoordinates{farPC, adjacentPC}, true},
		{"no PCs", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			participants := []models.ParticipantFull{
				{InstanceID: "npc-1", CellsCoords: &npcPos, RuntimeState: models.CreatureRuntimeState{CurrentHP: 20}},
			}
			for i, pos := range tt.pcs {
				p := pos // copy
				participants = append(participants, models.ParticipantFull{
					InstanceID:        fmt.Sprintf("pc-%d", i),
					CellsCoords:       &p,
					RuntimeState:      models.CreatureRuntimeState{CurrentHP: 20},
					IsPlayerCharacter: true,
				})
			}

			input := &TurnInput{
				ActiveNPC: models.ParticipantFull{
					InstanceID:  "npc-1",
					CellsCoords: &npcPos,
				},
				Participants: participants,
			}

			got := inMeleeThreat(input)
			if got != tt.want {
				t.Errorf("inMeleeThreat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectAdjacentGoal(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	npcPos := cell(0, 5)
	targetPos := cell(5, 5)

	goal := selectAdjacentGoal(npcPos, targetPos, grid, nil, 10, 10)
	if goal == nil {
		t.Fatal("expected non-nil goal")
	}
	// Goal should be adjacent to target.
	dist := chebyshevCells(goal.CellsX, goal.CellsY, 5, 5)
	if dist != 1 {
		t.Errorf("goal (%d,%d) not adjacent to target (5,5), dist=%d", goal.CellsX, goal.CellsY, dist)
	}
	// Should be the closest to NPC — (4,5).
	if goal.CellsX != 4 || goal.CellsY != 5 {
		t.Logf("expected (4,5), got (%d,%d)", goal.CellsX, goal.CellsY)
	}
}

func TestSelectAdjacentGoal_AllBlocked(t *testing.T) {
	t.Parallel()

	grid := makeGrid(10, 10)
	// Block all cells around target.
	blockCells(grid,
		[2]int{4, 4}, [2]int{5, 4}, [2]int{6, 4},
		[2]int{4, 5}, [2]int{6, 5},
		[2]int{4, 6}, [2]int{5, 6}, [2]int{6, 6},
	)

	goal := selectAdjacentGoal(cell(0, 0), cell(5, 5), grid, nil, 10, 10)
	if goal != nil {
		t.Errorf("expected nil goal when all adjacent cells blocked, got (%d,%d)", goal.CellsX, goal.CellsY)
	}
}

func TestSelectRetreatGoal(t *testing.T) {
	t.Parallel()

	grid := makeGrid(20, 20)
	npcPos := cell(10, 10)
	threats := []models.CellsCoordinates{{CellsX: 10, CellsY: 11}} // threat to the south

	goal := selectRetreatGoal(npcPos, threats, 6, grid, nil, 20, 20)
	if goal == nil {
		t.Fatal("expected non-nil retreat goal")
	}
	// Goal should be north of NPC (away from threat).
	if goal.CellsY >= npcPos.CellsY {
		t.Errorf("retreat goal (%d,%d) should be north of NPC (10,10)", goal.CellsX, goal.CellsY)
	}
}

func TestFindStructuredAction(t *testing.T) {
	t.Parallel()

	creature := models.Creature{
		StructuredActions: []models.StructuredAction{
			{ID: "slash", Name: "Slash"},
			{ID: "bite", Name: "Bite"},
		},
	}

	sa := findStructuredAction(creature, "bite")
	if sa == nil {
		t.Fatal("expected to find action 'bite'")
	}
	if sa.Name != "Bite" {
		t.Errorf("expected Name='Bite', got %q", sa.Name)
	}

	missing := findStructuredAction(creature, "nonexistent")
	if missing != nil {
		t.Error("expected nil for nonexistent action")
	}
}
