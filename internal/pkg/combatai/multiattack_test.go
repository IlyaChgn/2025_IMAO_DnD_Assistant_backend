package combatai

import (
	"math"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestEvaluateMultiattack_SingleGroup(t *testing.T) {
	t.Parallel()

	bite := models.StructuredAction{
		ID:       "bite",
		Name:     "Bite",
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollMeleeWeapon,
			Bonus: 6,
			Reach: 5,
			Damage: []models.DamageRoll{
				{DiceCount: 2, DiceType: "d6", Bonus: 4, DamageType: "piercing"},
			},
		},
	}

	claw := models.StructuredAction{
		ID:       "claw",
		Name:     "Claw",
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollMeleeWeapon,
			Bonus: 6,
			Reach: 5,
			Damage: []models.DamageRoll{
				{DiceCount: 1, DiceType: "d8", Bonus: 4, DamageType: "slashing"},
			},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 80, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{bite, claw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     0.50,
	}

	groups := []models.MultiattackGroup{
		{
			ID:   "multi1",
			Name: "Bite + 2 Claws",
			Actions: []models.MultiattackEntry{
				{ActionID: "bite", Count: 1},
				{ActionID: "claw", Count: 2},
			},
		},
	}

	decision := EvaluateMultiattack(input, groups)
	if decision == nil {
		t.Fatal("EvaluateMultiattack returned nil, want non-nil")
	}
	if decision.MultiattackGroupID != "multi1" {
		t.Errorf("GroupID = %q, want %q", decision.MultiattackGroupID, "multi1")
	}
	if len(decision.MultiattackSteps) != 3 {
		t.Errorf("steps = %d, want 3", len(decision.MultiattackSteps))
	}
	if decision.ExpectedDamage <= 0 {
		t.Errorf("EV = %.2f, want > 0", decision.ExpectedDamage)
	}
}

func TestEvaluateMultiattack_BestGroup(t *testing.T) {
	t.Parallel()

	weakAttack := models.StructuredAction{
		ID: "weak", Name: "Weak", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 4, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 1, DamageType: "bludgeoning"}},
		},
	}
	strongAttack := models.StructuredAction{
		ID: "strong", Name: "Strong", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 8, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 3, DiceType: "d10", Bonus: 5, DamageType: "slashing"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 80, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{weakAttack, strongAttack}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     0.50,
	}

	groups := []models.MultiattackGroup{
		{
			ID: "weak_group", Name: "Two Weak",
			Actions: []models.MultiattackEntry{{ActionID: "weak", Count: 2}},
		},
		{
			ID: "strong_group", Name: "Two Strong",
			Actions: []models.MultiattackEntry{{ActionID: "strong", Count: 2}},
		},
	}

	decision := EvaluateMultiattack(input, groups)
	if decision == nil {
		t.Fatal("EvaluateMultiattack returned nil")
	}
	if decision.MultiattackGroupID != "strong_group" {
		t.Errorf("picked %q, want %q", decision.MultiattackGroupID, "strong_group")
	}
}

func TestEvaluateMultiattack_Empty(t *testing.T) {
	t.Parallel()

	decision := EvaluateMultiattack(&TurnInput{}, nil)
	if decision != nil {
		t.Errorf("empty groups: got %+v, want nil", decision)
	}
}

func TestEvaluateMultiattack_MissingActionID(t *testing.T) {
	t.Parallel()

	realAction := models.StructuredAction{
		ID: "real", Name: "Real", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 80, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{realAction}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     0.50,
	}

	groups := []models.MultiattackGroup{
		{
			ID: "group1", Name: "Mixed",
			Actions: []models.MultiattackEntry{
				{ActionID: "real", Count: 1},
				{ActionID: "nonexistent", Count: 2}, // should be skipped
			},
		},
	}

	decision := EvaluateMultiattack(input, groups)
	if decision == nil {
		t.Fatal("EvaluateMultiattack returned nil, want non-nil (real action should work)")
	}
	if len(decision.MultiattackSteps) != 1 {
		t.Errorf("steps = %d, want 1 (only 'real' should be included)", len(decision.MultiattackSteps))
	}

	// EV should be > 0 for the real action.
	if decision.ExpectedDamage <= 0 {
		t.Errorf("EV = %.2f, want > 0", decision.ExpectedDamage)
	}
}

func TestEvaluateMultiattack_EVSumsCorrectly(t *testing.T) {
	t.Parallel()

	action := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d6", Bonus: 3, DamageType: "slashing"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 80, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{action}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     0.50,
	}

	singleEV := ComputeExpectedDamage(action, CombatantStats{AC: 15})

	groups := []models.MultiattackGroup{
		{
			ID: "three_claws", Name: "Three Claws",
			Actions: []models.MultiattackEntry{{ActionID: "claw", Count: 3}},
		},
	}

	decision := EvaluateMultiattack(input, groups)
	if decision == nil {
		t.Fatal("EvaluateMultiattack returned nil")
	}

	expectedEV := singleEV * 3
	if math.Abs(decision.ExpectedDamage-expectedEV) > 0.01 {
		t.Errorf("EV = %.4f, want %.4f (3 × %.4f)", decision.ExpectedDamage, expectedEV, singleEV)
	}
}
