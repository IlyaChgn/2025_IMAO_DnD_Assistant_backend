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
