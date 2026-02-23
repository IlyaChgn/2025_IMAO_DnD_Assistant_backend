package combatai

import (
	"sort"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// --- Helper ---

func coords(x, y int) *models.CellsCoordinates {
	return &models.CellsCoordinates{CellsX: x, CellsY: y}
}

func enemy(id string, x, y int) *models.ParticipantFull {
	p := makeParticipant(id, true, 50, x, y)
	return &p
}

func sortedIDs(ids []string) []string {
	s := make([]string, len(ids))
	copy(s, ids)
	sort.Strings(s)
	return s
}

func assertIDs(t *testing.T, got []string, wantIDs ...string) {
	t.Helper()
	if len(got) != len(wantIDs) {
		t.Errorf("got %d IDs %v, want %d IDs %v", len(got), got, len(wantIDs), wantIDs)
		return
	}
	gotSorted := sortedIDs(got)
	wantSorted := sortedIDs(wantIDs)
	for i := range gotSorted {
		if gotSorted[i] != wantSorted[i] {
			t.Errorf("got IDs %v, want %v", gotSorted, wantSorted)
			return
		}
	}
}

// --- Sphere / Cylinder ---

func TestFindAoETargets_Sphere_3InRadius(t *testing.T) {
	t.Parallel()

	// NPC at (0,0), 3 enemies within 20ft sphere centered optimally.
	// Enemies at (2,0), (3,0), (4,0) — cluster at x=2..4.
	// Sphere radius 20ft = 4 cells. Centered at (3,0): all within 1 cell.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeSphere, Size: 20}
	enemies := []*models.ParticipantFull{
		enemy("pc1", 2, 0),
		enemy("pc2", 3, 0),
		enemy("pc3", 4, 0),
	}

	got := FindAoETargets(npc, area, enemies)
	assertIDs(t, got, "pc1", "pc2", "pc3")
}

func TestFindAoETargets_Sphere_1OutOfRange(t *testing.T) {
	t.Parallel()

	// Sphere 10ft (2 cells). Enemies at (2,0) and (10,0).
	// Best center = (2,0): captures only pc1. Center at (10,0): only pc2.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeSphere, Size: 10}
	enemies := []*models.ParticipantFull{
		enemy("pc1", 2, 0),
		enemy("pc2", 10, 0),
	}

	got := FindAoETargets(npc, area, enemies)
	if len(got) != 1 {
		t.Errorf("sphere 1 out of range: got %d targets, want 1", len(got))
	}
}

func TestFindAoETargets_Sphere_OptimalPlacement(t *testing.T) {
	t.Parallel()

	// 3 enemies: (3,0), (5,0), (7,0). Sphere 20ft (4 cells).
	// Centered at (5,0): captures all (dist 2, 0, 2 cells → 10, 0, 10 ft).
	// Centered at (3,0): captures (3,0) and (5,0) but not (7,0) (dist 4 cells = 20ft — yes!).
	// Actually dist(3,7) = 4 cells = 20ft → within 20ft radius. So (3,0) also captures all 3.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeSphere, Size: 20}
	enemies := []*models.ParticipantFull{
		enemy("pc1", 3, 0),
		enemy("pc2", 5, 0),
		enemy("pc3", 7, 0),
	}

	got := FindAoETargets(npc, area, enemies)
	assertIDs(t, got, "pc1", "pc2", "pc3")
}

func TestFindAoETargets_Cylinder_SameAsSphere(t *testing.T) {
	t.Parallel()

	// Cylinder uses same 2D logic as sphere.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeCylinder, Size: 15}
	enemies := []*models.ParticipantFull{
		enemy("pc1", 2, 0),
		enemy("pc2", 3, 0),
	}

	got := FindAoETargets(npc, area, enemies)
	assertIDs(t, got, "pc1", "pc2")
}

// --- Cone ---

func TestFindAoETargets_Cone_FireBreath15ft(t *testing.T) {
	t.Parallel()

	// NPC at (0,0). Cone 15ft (3 cells) pointed right.
	// Enemies at (1,0), (2,0), (3,0) — all along the axis.
	// At distance 1: half-width = 0.5 cells. perp = 0 → in.
	// At distance 2: half-width = 1 cell. perp = 0 → in.
	// At distance 3: half-width = 1.5 cells. perp = 0 → in.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeCone, Size: 15}
	enemies := []*models.ParticipantFull{
		enemy("pc1", 1, 0),
		enemy("pc2", 2, 0),
		enemy("pc3", 3, 0),
	}

	got := FindAoETargets(npc, area, enemies)
	assertIDs(t, got, "pc1", "pc2", "pc3")
}

func TestFindAoETargets_Cone_EnemyBehind(t *testing.T) {
	t.Parallel()

	// NPC at (5,5). Reference enemy at (7,5) → cone points right.
	// Enemy at (3,5) → behind NPC (along < 0) → excluded.
	npc := coords(5, 5)
	area := &models.AreaOfEffect{Shape: models.AreaShapeCone, Size: 15}
	enemies := []*models.ParticipantFull{
		enemy("front", 7, 5),
		enemy("behind", 3, 5),
	}

	got := FindAoETargets(npc, area, enemies)
	// Best direction toward "front" captures only "front" (behind is excluded).
	// Best direction toward "behind" captures only "behind" and "front" is behind for that direction.
	// Either way: max 1 target.
	if len(got) != 1 {
		t.Errorf("cone enemy behind: got %d targets %v, want 1", len(got), got)
	}
}

func TestFindAoETargets_Cone_WidthCheck(t *testing.T) {
	t.Parallel()

	// NPC at (0,0). Cone 30ft (6 cells) pointed toward (6,0).
	// Enemy at (6,2): along = 6, perp = 2. half-width = 6/2 = 3. 2 <= 3 → in.
	// Enemy at (6,4): along = ~6, perp = ~4. half-width = 3. 4 > 3 → out.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeCone, Size: 30}
	enemies := []*models.ParticipantFull{
		enemy("ref", 6, 0),
		enemy("edge_in", 6, 2),
		enemy("edge_out", 6, 4),
	}

	got := FindAoETargets(npc, area, enemies)
	// Direction toward "ref": captures ref (perp=0), edge_in (perp≈2 <= 3), not edge_out (perp≈4 > 3).
	// Direction toward "edge_in": might capture differently, but max should be 2.
	if len(got) < 2 {
		t.Errorf("cone width check: got %d targets %v, want at least 2 (ref + edge_in)", len(got), got)
	}
	// Verify edge_out is NOT in the set.
	for _, id := range got {
		if id == "edge_out" {
			t.Error("cone width check: edge_out should not be in AoE")
		}
	}
}

func TestFindAoETargets_Cone_OptimalDirection(t *testing.T) {
	t.Parallel()

	// NPC at (0,0). Cone 15ft.
	// Group A: (2,0), (3,0) — 2 enemies in a line to the right.
	// Group B: (0,2) — 1 enemy straight up.
	// Optimal: point cone toward group A → 2 targets.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeCone, Size: 15}
	enemies := []*models.ParticipantFull{
		enemy("a1", 2, 0),
		enemy("a2", 3, 0),
		enemy("b1", 0, 2),
	}

	got := FindAoETargets(npc, area, enemies)
	if len(got) != 2 {
		t.Errorf("cone optimal direction: got %d targets %v, want 2 (a1, a2)", len(got), got)
	}
}

// --- Cube ---

func TestFindAoETargets_Cube_AllInside(t *testing.T) {
	t.Parallel()

	// Cube 20ft (4 cells side). Centered at (5,5): half = 2 cells.
	// Enemies at (4,5), (5,5), (6,5) — all within 1 cell of center.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeCube, Size: 20}
	enemies := []*models.ParticipantFull{
		enemy("pc1", 4, 5),
		enemy("pc2", 5, 5),
		enemy("pc3", 6, 5),
	}

	got := FindAoETargets(npc, area, enemies)
	assertIDs(t, got, "pc1", "pc2", "pc3")
}

func TestFindAoETargets_Cube_OutsideExcluded(t *testing.T) {
	t.Parallel()

	// Cube 10ft (2 cells side → half = 1 cell).
	// Center at (5,5): captures within ±1 cell.
	// Enemy at (5,5) → in. Enemy at (5,7) → dy=2 > 1 → out.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeCube, Size: 10}
	enemies := []*models.ParticipantFull{
		enemy("in", 5, 5),
		enemy("out", 5, 7),
	}

	got := FindAoETargets(npc, area, enemies)
	if len(got) != 1 || got[0] != "in" {
		t.Errorf("cube outside excluded: got %v, want [in]", got)
	}
}

// --- Line ---

func TestFindAoETargets_Line_TwoAligned(t *testing.T) {
	t.Parallel()

	// NPC at (0,0). Line 30ft long, 5ft wide.
	// Enemies at (2,0) and (4,0) — both on the axis.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeLine, Size: 30, Width: 5}
	enemies := []*models.ParticipantFull{
		enemy("pc1", 2, 0),
		enemy("pc2", 4, 0),
	}

	got := FindAoETargets(npc, area, enemies)
	assertIDs(t, got, "pc1", "pc2")
}

func TestFindAoETargets_Line_WidthCheck(t *testing.T) {
	t.Parallel()

	// NPC at (0,0). Line 60ft long, 10ft wide (half-width = 1 cell).
	// Direction toward (6,0).
	// Enemy at (6,1): along=6, perp=1. half-width=1 cell. 1 <= 1 → in.
	// Enemy at (6,3): along≈6, perp≈3. 3 > 1 → out.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeLine, Size: 60, Width: 10}
	enemies := []*models.ParticipantFull{
		enemy("ref", 6, 0),
		enemy("edge_in", 6, 1),
		enemy("edge_out", 6, 3),
	}

	got := FindAoETargets(npc, area, enemies)
	if len(got) < 2 {
		t.Errorf("line width check: got %d targets %v, want at least 2", len(got), got)
	}
	for _, id := range got {
		if id == "edge_out" {
			t.Error("line width check: edge_out should not be in line")
		}
	}
}

func TestFindAoETargets_Line_DefaultWidth(t *testing.T) {
	t.Parallel()

	// Line with Width=0 → defaults to 5ft (0.5 cells half-width).
	// Enemy at (3,0) on axis → in.
	// Enemy at (3,1) perp=1 > 0.5 → out.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeLine, Size: 30}
	enemies := []*models.ParticipantFull{
		enemy("on_axis", 3, 0),
		enemy("off_axis", 3, 1),
	}

	got := FindAoETargets(npc, area, enemies)
	if len(got) != 1 || got[0] != "on_axis" {
		t.Errorf("line default width: got %v, want [on_axis]", got)
	}
}

// --- Edge cases ---

func TestFindAoETargets_NilNPCCoords(t *testing.T) {
	t.Parallel()

	area := &models.AreaOfEffect{Shape: models.AreaShapeSphere, Size: 20}
	enemies := []*models.ParticipantFull{enemy("pc1", 1, 0)}

	got := FindAoETargets(nil, area, enemies)
	if got != nil {
		t.Errorf("nil NPC coords: got %v, want nil", got)
	}
}

func TestFindAoETargets_NilArea(t *testing.T) {
	t.Parallel()

	npc := coords(0, 0)
	enemies := []*models.ParticipantFull{enemy("pc1", 1, 0)}

	got := FindAoETargets(npc, nil, enemies)
	if got != nil {
		t.Errorf("nil area: got %v, want nil", got)
	}
}

func TestFindAoETargets_NoEnemies(t *testing.T) {
	t.Parallel()

	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeSphere, Size: 20}

	got := FindAoETargets(npc, area, nil)
	if got != nil {
		t.Errorf("no enemies: got %v, want nil", got)
	}
}

func TestFindAoETargets_EnemiesNoCoords(t *testing.T) {
	t.Parallel()

	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeSphere, Size: 20}

	noCoords := &models.ParticipantFull{
		InstanceID:        "pc1",
		IsPlayerCharacter: true,
		RuntimeState:      models.CreatureRuntimeState{CurrentHP: 50, MaxHP: 100},
	}
	enemies := []*models.ParticipantFull{noCoords}

	got := FindAoETargets(npc, area, enemies)
	if got != nil {
		t.Errorf("enemies no coords: got %v, want nil", got)
	}
}

func TestFindAoETargets_Cone_EnemyOnNPC(t *testing.T) {
	t.Parallel()

	// Enemy on same cell as NPC → dirLen = 0, skipped as reference direction.
	npc := coords(0, 0)
	area := &models.AreaOfEffect{Shape: models.AreaShapeCone, Size: 15}
	enemies := []*models.ParticipantFull{
		enemy("same_cell", 0, 0),
		enemy("front", 2, 0),
	}

	got := FindAoETargets(npc, area, enemies)
	// Direction toward "front": captures "front" (along=2, perp=0, ok).
	// "same_cell" at (0,0): along=0 → not captured.
	if len(got) != 1 || got[0] != "front" {
		t.Errorf("cone enemy on NPC: got %v, want [front]", got)
	}
}
