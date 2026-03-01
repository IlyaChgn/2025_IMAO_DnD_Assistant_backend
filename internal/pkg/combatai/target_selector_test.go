package combatai

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// makeParticipant creates a minimal alive participant for target selection tests.
func makeParticipant(id string, isPC bool, hp int, x, y int) models.ParticipantFull {
	return models.ParticipantFull{
		InstanceID:        id,
		IsPlayerCharacter: isPC,
		CellsCoords:       &models.CellsCoordinates{CellsX: x, CellsY: y},
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: hp,
			MaxHP:     100,
		},
	}
}

// makeMeleeAction creates a melee attack action with reach 5.
func makeMeleeAction() *models.StructuredAction {
	return &models.StructuredAction{
		ID:       "slash",
		Name:     "Slash",
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollMeleeWeapon,
			Bonus: 5,
			Reach: 5,
			Damage: []models.DamageRoll{
				{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"},
			},
		},
	}
}

// makeRangedAction creates a ranged attack action.
func makeRangedAction() *models.StructuredAction {
	return &models.StructuredAction{
		ID:       "bow",
		Name:     "Bow",
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollRangedWeapon,
			Bonus: 4,
			Range: &models.RangeData{Normal: 80, Long: 320},
			Damage: []models.DamageRoll{
				{DiceCount: 1, DiceType: "d6", Bonus: 2, DamageType: "piercing"},
			},
		},
	}
}

func TestSelectTarget_ZombieStickyAlive(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 30, 1, 0), makeParticipant("pc2", true, 50, 2, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 30, AC: 15}, "pc2": {MaxHP: 50, AC: 14}},
		Intelligence:     0.10,
		PreviousTargetID: "pc2",
	}

	targets := SelectTarget(input, makeMeleeAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("zombie sticky: got %v, want [pc2]", targets)
	}
}

func TestSelectTarget_ZombyStickyDead(t *testing.T) {
	t.Parallel()

	deadPC := makeParticipant("pc2", true, 0, 2, 0)
	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 30, 1, 0), deadPC},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 30, AC: 15}, "pc2": {MaxHP: 50, AC: 14}},
		Intelligence:     0.10,
		PreviousTargetID: "pc2",
	}

	targets := SelectTarget(input, makeMeleeAction())
	if len(targets) != 1 || targets[0] != "pc1" {
		t.Errorf("zombie sticky dead → nearest: got %v, want [pc1]", targets)
	}
}

func TestSelectTarget_GoblinFinishesWounded(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC:    makeParticipant("npc1", false, 50, 0, 0),
		Participants: []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0), makeParticipant("pc2", true, 5, 1, 0)},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 100, AC: 12}, // 5/100 = 5% HP — wounded
		},
		Intelligence: 0.25,
	}

	targets := SelectTarget(input, makeMeleeAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("goblin finish wounded: got %v, want [pc2]", targets)
	}
}

func TestSelectTarget_NormalMeleeLowestHP(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		Participants: []models.ParticipantFull{
			makeParticipant("pc1", true, 30, 1, 0),
			makeParticipant("pc2", true, 10, 1, 0),
			makeParticipant("pc3", true, 50, 1, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 18},
			"pc3": {MaxHP: 50, AC: 12},
		},
		Intelligence: 0.45,
	}

	targets := SelectTarget(input, makeMeleeAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("normal melee lowest HP: got %v, want [pc2] (HP=10)", targets)
	}
}

func TestSelectTarget_NormalRangedLowestAC(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		Participants: []models.ParticipantFull{
			makeParticipant("pc1", true, 30, 3, 0),
			makeParticipant("pc2", true, 30, 4, 0),
			makeParticipant("pc3", true, 30, 5, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 18},
			"pc2": {MaxHP: 50, AC: 12},
			"pc3": {MaxHP: 50, AC: 15},
		},
		Intelligence: 0.45,
	}

	targets := SelectTarget(input, makeRangedAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("normal ranged lowest AC: got %v, want [pc2] (AC=12)", targets)
	}
}

func TestSelectTarget_SmartPrefersConcentrating(t *testing.T) {
	t.Parallel()

	conc := makeParticipant("pc2", true, 40, 1, 0)
	conc.RuntimeState.Concentration = &models.ConcentrationState{EffectName: "Bless"}

	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		Participants: []models.ParticipantFull{
			makeParticipant("pc1", true, 20, 1, 0),
			conc,
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 15},
		},
		Intelligence: 0.70,
	}

	targets := SelectTarget(input, makeMeleeAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("smart → concentration: got %v, want [pc2]", targets)
	}
}

func TestSelectTarget_NoEnemies(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		Participants: []models.ParticipantFull{
			makeParticipant("npc2", false, 50, 1, 0), // another NPC, not an enemy
		},
		CombatantStats: map[string]CombatantStats{},
		Intelligence:   0.50,
	}

	targets := SelectTarget(input, makeMeleeAction())
	if targets != nil {
		t.Errorf("no enemies: got %v, want nil", targets)
	}
}

func TestSelectTarget_AllDead(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		Participants: []models.ParticipantFull{
			makeParticipant("pc1", true, 0, 1, 0), // dead
			makeParticipant("pc2", true, 0, 2, 0), // dead
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 14},
		},
		Intelligence: 0.50,
	}

	targets := SelectTarget(input, makeMeleeAction())
	if targets != nil {
		t.Errorf("all dead: got %v, want nil", targets)
	}
}

// makeFireMeleeAction creates a melee attack that deals fire damage.
func makeFireMeleeAction() *models.StructuredAction {
	return &models.StructuredAction{
		ID:       "flame_tongue",
		Name:     "Flame Tongue",
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollMeleeWeapon,
			Bonus: 5,
			Reach: 5,
			Damage: []models.DamageRoll{
				{DiceCount: 2, DiceType: "d6", Bonus: 3, DamageType: "fire"},
			},
		},
	}
}

func TestSelectTarget_SmartAvoidsImmune(t *testing.T) {
	t.Parallel()

	// NPC deals fire damage. pc1 is immune to fire, pc2 is not.
	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{
			Movement: models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{
				*makeFireMeleeAction(),
			},
		},
		Participants: []models.ParticipantFull{
			makeParticipant("npc1", false, 50, 0, 0),
			makeParticipant("pc1", true, 50, 1, 0),
			makeParticipant("pc2", true, 50, 1, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, Immunities: []string{"fire"}},
			"pc2": {MaxHP: 50, AC: 15},
		},
		Intelligence: 0.70,
	}

	targets := SelectTarget(input, makeFireMeleeAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("smart avoids immune: got %v, want [pc2]", targets)
	}
}

func TestSelectTarget_SmartPrefersVulnerable(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{
			Movement: models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{
				*makeFireMeleeAction(),
			},
		},
		Participants: []models.ParticipantFull{
			makeParticipant("npc1", false, 50, 0, 0),
			makeParticipant("pc1", true, 50, 1, 0),
			makeParticipant("pc2", true, 50, 1, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 15, Vulnerabilities: []string{"fire"}},
		},
		Intelligence: 0.70,
	}

	targets := SelectTarget(input, makeFireMeleeAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("smart prefers vulnerable: got %v, want [pc2]", targets)
	}
}

func TestSelectTarget_SmartDistancePenalty(t *testing.T) {
	t.Parallel()

	// pc1 nearby, pc2 far away. Both same stats otherwise.
	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{
			Movement: models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{
				*makeMeleeAction(),
			},
		},
		Participants: []models.ParticipantFull{
			makeParticipant("npc1", false, 50, 0, 0),
			makeParticipant("pc1", true, 50, 1, 0),  // 5ft away
			makeParticipant("pc2", true, 50, 20, 0), // 100ft away
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 15},
		},
		Intelligence: 0.70,
	}

	// Using a ranged action so both targets are in range.
	targets := SelectTarget(input, makeRangedAction())
	if len(targets) != 1 || targets[0] != "pc1" {
		t.Errorf("smart distance penalty: got %v, want [pc1] (nearby)", targets)
	}
}

func TestSelectTarget_SmartConcentrationOverridesDistance(t *testing.T) {
	t.Parallel()

	// pc1 nearby, pc2 far but concentrating. Concentration (+30) > distance penalty (-15).
	// pc2 at 15 cells = 75ft (within 80ft ranged range, but > 30+5=35 movement+reach → penalty).
	pc2 := makeParticipant("pc2", true, 50, 15, 0)
	pc2.RuntimeState.Concentration = &models.ConcentrationState{EffectName: "Haste"}

	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{
			Movement: models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{
				*makeMeleeAction(),
			},
		},
		Participants: []models.ParticipantFull{
			makeParticipant("npc1", false, 50, 0, 0),
			makeParticipant("pc1", true, 50, 1, 0),
			pc2,
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 15},
		},
		Intelligence: 0.70,
	}

	// Ranged action so both are in range (80ft range, pc2 at 75ft).
	targets := SelectTarget(input, makeRangedAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("smart concentration > distance: got %v, want [pc2] (concentrating)", targets)
	}
}

func TestSelectTarget_SmartFocusFire(t *testing.T) {
	t.Parallel()

	// 3 PCs with identical stats — no concentration, same HP, same AC, same distance.
	// Without focus fire, any target is equally valid.
	// With 2 allies targeting pc1, focus-fire bonus (+30) should make pc1 the clear winner.
	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{
			Movement: models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{
				*makeMeleeAction(),
			},
		},
		Participants: []models.ParticipantFull{
			makeParticipant("npc1", false, 50, 0, 0),
			makeParticipant("pc1", true, 50, 1, 0),
			makeParticipant("pc2", true, 50, 1, 0),
			makeParticipant("pc3", true, 50, 1, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 15},
			"pc3": {MaxHP: 50, AC: 15},
		},
		Intelligence:     0.70,
		RecentNPCTargets: map[string]string{"npc2": "pc1", "npc3": "pc1"},
	}

	targets := SelectTarget(input, makeMeleeAction())
	if len(targets) != 1 || targets[0] != "pc1" {
		t.Errorf("smart focus fire: got %v, want [pc1] (2 allies already targeting pc1)", targets)
	}
}

func TestSelectTarget_SmartFocusFire_NilMap(t *testing.T) {
	t.Parallel()

	// With nil RecentNPCTargets, focus fire is disabled. All PCs are equally scored.
	// We just verify no crash and a valid target is returned.
	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{
			Movement: models.CreatureMovement{Walk: 30},
			StructuredActions: []models.StructuredAction{
				*makeMeleeAction(),
			},
		},
		Participants: []models.ParticipantFull{
			makeParticipant("npc1", false, 50, 0, 0),
			makeParticipant("pc1", true, 50, 1, 0),
			makeParticipant("pc2", true, 50, 1, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 15},
		},
		Intelligence:     0.70,
		RecentNPCTargets: nil,
	}

	targets := SelectTarget(input, makeMeleeAction())
	if len(targets) != 1 {
		t.Errorf("smart focus fire nil map: got %v, want exactly 1 target", targets)
	}
}

func TestSelectTarget_GoblinIgnoresFocusFire(t *testing.T) {
	t.Parallel()

	// Goblin tier (intelligence 0.25) should NOT be affected by focus fire.
	// Goblin logic: finish wounded (<25% HP) in reach, else nearest.
	// pc2 is wounded (5 HP), pc1 is focused by allies but full HP.
	// Goblin should pick pc2 (wounded), not pc1 (focus fire).
	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{
			Movement: models.CreatureMovement{Walk: 30},
		},
		Participants: []models.ParticipantFull{
			makeParticipant("npc1", false, 50, 0, 0),
			makeParticipant("pc1", true, 50, 1, 0),
			makeParticipant("pc2", true, 5, 1, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 100, AC: 15},
		},
		Intelligence:     0.25,
		RecentNPCTargets: map[string]string{"npc2": "pc1", "npc3": "pc1"},
	}

	targets := SelectTarget(input, makeMeleeAction())
	if len(targets) != 1 || targets[0] != "pc2" {
		t.Errorf("goblin ignores focus fire: got %v, want [pc2] (wounded, goblin tier)", targets)
	}
}
