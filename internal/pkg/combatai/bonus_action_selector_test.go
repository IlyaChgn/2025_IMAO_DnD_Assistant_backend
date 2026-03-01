package combatai

import (
	"math/rand"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// --- Bonus Action Tests ---

func TestSelectBonusAction_Selected(t *testing.T) {
	t.Parallel()

	bonusClaw := models.StructuredAction{
		ID: "bonus_claw", Name: "Bonus Claw", Category: models.ActionCategoryBonus,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 3, DamageType: "slashing"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 100, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{bonusClaw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectBonusAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectBonusAction returned nil, expected bonus claw")
	}
	if decision.ActionID != "bonus_claw" {
		t.Errorf("got ActionID=%q, want bonus_claw", decision.ActionID)
	}
	if decision.ExpectedDamage <= 0 {
		t.Errorf("got EV=%.1f, want > 0", decision.ExpectedDamage)
	}
}

func TestSelectBonusAction_AlreadyUsed(t *testing.T) {
	t.Parallel()

	bonusClaw := models.StructuredAction{
		ID: "bonus_claw", Name: "Bonus Claw", Category: models.ActionCategoryBonus,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 3, DamageType: "slashing"}},
		},
	}

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.BonusActionUsed = true

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{bonusClaw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectBonusAction(input, RoleBrute, rng)
	if decision != nil {
		t.Errorf("BonusActionUsed=true: got %+v, want nil", decision)
	}
}

func TestSelectBonusAction_NoBonusActions(t *testing.T) {
	t.Parallel()

	// Only main action, no bonus actions.
	mainAction := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 3, DamageType: "slashing"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 100, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{mainAction}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectBonusAction(input, RoleBrute, rng)
	if decision != nil {
		t.Errorf("no bonus actions: got %+v, want nil", decision)
	}
}

func TestSelectBonusAction_RechargeReady(t *testing.T) {
	t.Parallel()

	bonusBreath := models.StructuredAction{
		ID: "bonus_breath", Name: "Bonus Breath", Category: models.ActionCategoryBonus,
		Recharge: &models.RechargeData{MinRoll: 5},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 14, OnFail: "full damage", OnSuccess: "half damage",
			Damage: []models.DamageRoll{{DiceCount: 3, DiceType: "d6", DamageType: "fire"}},
		},
	}
	bonusClaw := models.StructuredAction{
		ID: "bonus_claw", Name: "Bonus Claw", Category: models.ActionCategoryBonus,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 4, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 2, DamageType: "slashing"}},
		},
	}

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.RechargeReady = map[string]bool{"bonus_breath": true}

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{bonusBreath, bonusClaw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectBonusAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectBonusAction returned nil")
	}
	if decision.ActionID != "bonus_breath" {
		t.Errorf("recharge priority: got ActionID=%q, want bonus_breath", decision.ActionID)
	}
}

func TestSelectBonusAction_BestByEV(t *testing.T) {
	t.Parallel()

	weak := models.StructuredAction{
		ID: "bonus_slap", Name: "Bonus Slap", Category: models.ActionCategoryBonus,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 1, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 0, DamageType: "bludgeoning"}},
		},
	}
	strong := models.StructuredAction{
		ID: "bonus_bite", Name: "Bonus Bite", Category: models.ActionCategoryBonus,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 7, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d6", Bonus: 4, DamageType: "piercing"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 100, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{weak, strong}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectBonusAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectBonusAction returned nil")
	}
	if decision.ActionID != "bonus_bite" {
		t.Errorf("best by EV: got ActionID=%q, want bonus_bite", decision.ActionID)
	}
}

// --- Legendary Action Tests ---

func TestSelectLegendaryAction_Selected(t *testing.T) {
	t.Parallel()

	legendaryTail := models.StructuredAction{
		ID: "tail_attack", Name: "Tail Attack", Category: models.ActionCategoryLegendary,
		LegendaryCost: 1,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 8, Reach: 10,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d8", Bonus: 5, DamageType: "bludgeoning"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.LegendaryActions = 3

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{legendaryTail}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectLegendaryAction(input, rng)
	if decision == nil {
		t.Fatal("SelectLegendaryAction returned nil")
	}
	if decision.ActionID != "tail_attack" {
		t.Errorf("got ActionID=%q, want tail_attack", decision.ActionID)
	}
	if decision.LegendaryCost != 1 {
		t.Errorf("got LegendaryCost=%d, want 1", decision.LegendaryCost)
	}
}

func TestSelectLegendaryAction_CostCheck(t *testing.T) {
	t.Parallel()

	// LegendaryCost=2, only 1 remaining → skipped.
	expensiveAction := models.StructuredAction{
		ID: "wing_attack", Name: "Wing Attack", Category: models.ActionCategoryLegendary,
		LegendaryCost: 2,
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 16, OnFail: "full damage", OnSuccess: "half damage",
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d6", Bonus: 5, DamageType: "bludgeoning"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.LegendaryActions = 1 // not enough for cost=2

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{expensiveAction}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectLegendaryAction(input, rng)
	if decision != nil {
		t.Errorf("cost=2 with 1 remaining: got %+v, want nil", decision)
	}
}

func TestSelectLegendaryAction_CostCheckPass(t *testing.T) {
	t.Parallel()

	// LegendaryCost=2, 2 remaining → should work.
	expensiveAction := models.StructuredAction{
		ID: "wing_attack", Name: "Wing Attack", Category: models.ActionCategoryLegendary,
		LegendaryCost: 2,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 8, Reach: 10,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d6", Bonus: 5, DamageType: "bludgeoning"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.LegendaryActions = 2

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{expensiveAction}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectLegendaryAction(input, rng)
	if decision == nil {
		t.Fatal("cost=2 with 2 remaining: got nil, want wing_attack")
	}
	if decision.ActionID != "wing_attack" {
		t.Errorf("got ActionID=%q, want wing_attack", decision.ActionID)
	}
	if decision.LegendaryCost != 2 {
		t.Errorf("got LegendaryCost=%d, want 2", decision.LegendaryCost)
	}
}

func TestSelectLegendaryAction_NoActionsRemaining(t *testing.T) {
	t.Parallel()

	legendaryTail := models.StructuredAction{
		ID: "tail_attack", Name: "Tail Attack", Category: models.ActionCategoryLegendary,
		LegendaryCost: 1,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 8, Reach: 10,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d8", Bonus: 5, DamageType: "bludgeoning"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.LegendaryActions = 0 // exhausted

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{legendaryTail}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectLegendaryAction(input, rng)
	if decision != nil {
		t.Errorf("no actions remaining: got %+v, want nil", decision)
	}
}

func TestSelectLegendaryAction_DefaultCost(t *testing.T) {
	t.Parallel()

	// LegendaryCost=0 → defaults to 1.
	legendaryTail := models.StructuredAction{
		ID: "tail_attack", Name: "Tail Attack", Category: models.ActionCategoryLegendary,
		// LegendaryCost not set (0)
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 8, Reach: 10,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d8", Bonus: 5, DamageType: "bludgeoning"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.LegendaryActions = 1

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{legendaryTail}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectLegendaryAction(input, rng)
	if decision == nil {
		t.Fatal("default cost: got nil, want tail_attack")
	}
	if decision.LegendaryCost != 1 {
		t.Errorf("got LegendaryCost=%d, want 1 (default)", decision.LegendaryCost)
	}
}

func TestSelectLegendaryAction_BestByEV(t *testing.T) {
	t.Parallel()

	weakTail := models.StructuredAction{
		ID: "tail_swipe", Name: "Tail Swipe", Category: models.ActionCategoryLegendary,
		LegendaryCost: 1,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 3, Reach: 10,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 1, DamageType: "bludgeoning"}},
		},
	}
	strongDetect := models.StructuredAction{
		ID: "tail_slam", Name: "Tail Slam", Category: models.ActionCategoryLegendary,
		LegendaryCost: 1,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 8, Reach: 10,
			Damage: []models.DamageRoll{{DiceCount: 3, DiceType: "d10", Bonus: 5, DamageType: "bludgeoning"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.LegendaryActions = 3

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{weakTail, strongDetect}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectLegendaryAction(input, rng)
	if decision == nil {
		t.Fatal("best by EV: got nil")
	}
	if decision.ActionID != "tail_slam" {
		t.Errorf("best by EV: got ActionID=%q, want tail_slam", decision.ActionID)
	}
}

func TestSelectLegendaryAction_IntelligenceGate(t *testing.T) {
	t.Parallel()

	weak := models.StructuredAction{
		ID: "detect", Name: "Detect", Category: models.ActionCategoryLegendary,
		LegendaryCost: 1,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 1, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 0, DamageType: "bludgeoning"}},
		},
	}
	strong := models.StructuredAction{
		ID: "tail_attack", Name: "Tail Attack", Category: models.ActionCategoryLegendary,
		LegendaryCost: 1,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 8, Reach: 10,
			Damage: []models.DamageRoll{{DiceCount: 3, DiceType: "d10", Bonus: 5, DamageType: "bludgeoning"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.LegendaryActions = 3

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{weak, strong}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     0.10, // very low — mostly random
	}

	// Low intelligence: should sometimes pick the weaker option.
	foundWeak := false
	for seed := int64(0); seed < 100; seed++ {
		rng := rand.New(rand.NewSource(seed))
		d := SelectLegendaryAction(input, rng)
		if d != nil && d.ActionID == "detect" {
			foundWeak = true
			break
		}
	}
	if !foundWeak {
		t.Error("intelligence gate: low INT should sometimes pick suboptimal, but never picked 'detect' in 100 tries")
	}
}

func TestSelectLegendaryAction_NoLegendaryCategory(t *testing.T) {
	t.Parallel()

	// Only main actions, no legendary category.
	mainAction := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 3, DamageType: "slashing"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.LegendaryActions = 3

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{mainAction}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectLegendaryAction(input, rng)
	if decision != nil {
		t.Errorf("no legendary category: got %+v, want nil", decision)
	}
}
