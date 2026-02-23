package combatai

import (
	"strings"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestDecideTurn_DeadNPC(t *testing.T) {
	t.Parallel()

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:      makeParticipant("npc1", false, 0, 0, 0), // dead
		CombatantStats: map[string]CombatantStats{},
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision != nil {
		t.Errorf("dead NPC: got %+v, want nil", decision)
	}
}

func TestDecideTurn_IncapacitatedStunned(t *testing.T) {
	t.Parallel()

	npc := makeParticipant("npc1", false, 50, 0, 0)
	npc.RuntimeState.Conditions = []models.ActiveCondition{
		{Condition: models.ConditionStunned},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:      npc,
		CombatantStats: map[string]CombatantStats{},
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("stunned NPC: got nil, want TurnDecision")
	}
	if decision.Action != nil {
		t.Errorf("stunned NPC should have nil Action, got %+v", decision.Action)
	}
	if !strings.Contains(decision.Reasoning, "Incapacitated") {
		t.Errorf("reasoning %q should contain 'Incapacitated'", decision.Reasoning)
	}
}

func TestDecideTurn_IncapacitatedParalyzed(t *testing.T) {
	t.Parallel()

	npc := makeParticipant("npc1", false, 50, 0, 0)
	npc.RuntimeState.Conditions = []models.ActiveCondition{
		{Condition: models.ConditionParalyzed},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:      npc,
		CombatantStats: map[string]CombatantStats{},
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil || decision.Action != nil {
		t.Errorf("paralyzed NPC: expected TurnDecision with nil Action")
	}
}

func TestDecideTurn_IncapacitatedUnconscious(t *testing.T) {
	t.Parallel()

	npc := makeParticipant("npc1", false, 50, 0, 0)
	npc.RuntimeState.Conditions = []models.ActiveCondition{
		{Condition: models.ConditionUnconscious},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:      npc,
		CombatantStats: map[string]CombatantStats{},
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil || decision.Action != nil {
		t.Errorf("unconscious NPC: expected TurnDecision with nil Action")
	}
}

func TestDecideTurn_ZombieMeleeAttack(t *testing.T) {
	t.Parallel()

	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 4, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 2, DamageType: "slashing"}},
		},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 30, 0, 0),
		CreatureTemplate: models.Creature{Ability: models.Ability{Int: 3}, StructuredActions: []models.StructuredAction{claw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 40, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"npc1": {MaxHP: 30, AC: 8}, "pc1": {MaxHP: 40, AC: 15}},
		Intelligence:     0.10,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("zombie attack: got nil decision")
	}
	if decision.Action == nil {
		t.Fatal("zombie attack: Action is nil, want attack")
	}
	if decision.Action.ActionID != "claw" {
		t.Errorf("got ActionID=%q, want 'claw'", decision.Action.ActionID)
	}
}

func TestDecideTurn_DragonBreathReady(t *testing.T) {
	t.Parallel()

	breath := models.StructuredAction{
		ID: "breath", Name: "Fire Breath", Category: models.ActionCategoryAction,
		Recharge: &models.RechargeData{MinRoll: 5},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 18, OnFail: "full damage", OnSuccess: "half damage",
			Damage: []models.DamageRoll{{DiceCount: 12, DiceType: "d6", DamageType: "fire"}},
		},
	}
	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 10, Reach: 10,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d6", Bonus: 6, DamageType: "slashing"}},
		},
	}

	npc := makeParticipant("npc1", false, 200, 0, 0)
	npc.RuntimeState.Resources.RechargeReady = map[string]bool{"breath": true}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{Ability: models.Ability{Int: 16}, StructuredActions: []models.StructuredAction{breath, claw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"npc1": {MaxHP: 200, AC: 19}, "pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 5}}},
		Intelligence:     0.80,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil || decision.Action == nil {
		t.Fatal("dragon breath: got nil decision/action")
	}
	if decision.Action.ActionID != "breath" {
		t.Errorf("got ActionID=%q, want 'breath' (recharge-ready)", decision.Action.ActionID)
	}
}

func TestDecideTurn_LichCastsSpell(t *testing.T) {
	t.Parallel()

	staff := models.StructuredAction{
		ID: "staff", Name: "Staff", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 3, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 0, DamageType: "bludgeoning"}},
		},
	}

	qr := &models.SpellQuickRef{Range: "150 feet"}
	npc := makeParticipant("npc1", false, 135, 0, 0)
	npc.RuntimeState.Resources.SpellSlots = map[int]int{5: 2}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC: npc,
		CreatureTemplate: models.Creature{
			Ability:           models.Ability{Int: 20},
			StructuredActions: []models.StructuredAction{staff},
			Spellcasting: &models.Spellcasting{
				SpellSaveDC:      20,
				SpellAttackBonus: 12,
				CasterLevel:      18,
				SpellsByLevel: map[int][]models.SpellKnown{
					5: {{Name: "Cone of Cold", Level: 5, QuickRef: qr}},
				},
			},
		},
		Participants:   []models.ParticipantFull{makeParticipant("pc1", true, 60, 1, 0)},
		CombatantStats: map[string]CombatantStats{"npc1": {MaxHP: 135, AC: 17}, "pc1": {MaxHP: 60, AC: 15, SaveBonuses: map[string]int{"DEX": 3}}},
		Intelligence:   1.0,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil || decision.Action == nil {
		t.Fatal("lich spell: got nil decision/action")
	}
	if decision.Action.ActionType != models.ActionSpellCast {
		t.Errorf("got ActionType=%q, want %q", decision.Action.ActionType, models.ActionSpellCast)
	}
}

func TestDecideTurn_SkeletonArcherRanged(t *testing.T) {
	t.Parallel()

	shortbow := models.StructuredAction{
		ID: "shortbow", Name: "Shortbow", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollRangedWeapon, Bonus: 4,
			Range: &models.RangeData{Normal: 80, Long: 320},
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 2, DamageType: "piercing"}},
		},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 13, 0, 0),
		CreatureTemplate: models.Creature{Ability: models.Ability{Int: 6}, StructuredActions: []models.StructuredAction{shortbow}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 30, 5, 0)},
		CombatantStats:   map[string]CombatantStats{"npc1": {MaxHP: 13, AC: 13}, "pc1": {MaxHP: 30, AC: 14}},
		Intelligence:     0.20,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil || decision.Action == nil {
		t.Fatal("skeleton archer: got nil decision/action")
	}
	if decision.Action.ActionID != "shortbow" {
		t.Errorf("got ActionID=%q, want 'shortbow'", decision.Action.ActionID)
	}
}

func TestDecideTurn_LowHPNoAttacks_Dodge(t *testing.T) {
	t.Parallel()

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 5, 0, 0),
		CreatureTemplate: models.Creature{Ability: models.Ability{Int: 10}}, // no actions
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 40, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"npc1": {MaxHP: 100, AC: 12}, "pc1": {MaxHP: 40, AC: 15}},
		Intelligence:     0.50,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil || decision.Action == nil {
		t.Fatal("low HP dodge: got nil decision/action")
	}
	if decision.Action.ActionID != "dodge" {
		t.Errorf("got ActionID=%q, want 'dodge'", decision.Action.ActionID)
	}
}

func TestDecideTurn_HealthyNoAttacks_Skip(t *testing.T) {
	t.Parallel()

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 100, 0, 0),
		CreatureTemplate: models.Creature{Ability: models.Ability{Int: 10}}, // no actions
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 40, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"npc1": {MaxHP: 100, AC: 12}, "pc1": {MaxHP: 40, AC: 15}},
		Intelligence:     0.50,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("healthy skip: got nil decision")
	}
	if decision.Action != nil {
		t.Errorf("healthy NPC with no attacks should skip, got %+v", decision.Action)
	}
}

func TestDecideTurn_Multiattack(t *testing.T) {
	t.Parallel()

	bite := models.StructuredAction{
		ID: "bite", Name: "Bite", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 6, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d6", Bonus: 4, DamageType: "piercing"}},
		},
	}
	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 6, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 4, DamageType: "slashing"}},
		},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 100, 0, 0),
		CreatureTemplate: models.Creature{
			Ability:           models.Ability{Int: 10},
			StructuredActions: []models.StructuredAction{bite, claw},
			Multiattacks: []models.MultiattackGroup{
				{
					ID: "multi", Name: "Bite + 2 Claws",
					Actions: []models.MultiattackEntry{
						{ActionID: "bite", Count: 1},
						{ActionID: "claw", Count: 2},
					},
				},
			},
		},
		Participants:   []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats: map[string]CombatantStats{"npc1": {MaxHP: 100, AC: 15}, "pc1": {MaxHP: 50, AC: 15}},
		Intelligence:   1.0,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil || decision.Action == nil {
		t.Fatal("multiattack: got nil decision/action")
	}
	if decision.Action.MultiattackSteps == nil {
		t.Fatal("expected MultiattackSteps, got nil")
	}
	if len(decision.Action.MultiattackSteps) != 3 {
		t.Errorf("MultiattackSteps len = %d, want 3", len(decision.Action.MultiattackSteps))
	}
}

func TestDecideTurn_ReasoningContainsRole(t *testing.T) {
	t.Parallel()

	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 6, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 4, DamageType: "slashing"}},
		},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{Ability: models.Ability{Str: 18, Int: 5}, StructuredActions: []models.StructuredAction{claw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 40, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"npc1": {MaxHP: 50, AC: 12}, "pc1": {MaxHP: 40, AC: 15}},
		Intelligence:     0.50,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("got nil decision")
	}
	// Ogre-like creature (high STR) should be classified as Brute.
	if !strings.Contains(decision.Reasoning, "Brute") {
		t.Errorf("reasoning %q should contain role name 'Brute'", decision.Reasoning)
	}
}

func TestDecideTurn_PureFunction(t *testing.T) {
	t.Parallel()

	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{Ability: models.Ability{Int: 10}, StructuredActions: []models.StructuredAction{claw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 40, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"npc1": {MaxHP: 50, AC: 12}, "pc1": {MaxHP: 40, AC: 15}},
		Intelligence:     1.0,
	}

	// Same seed → same result.
	ai1 := NewRuleBasedAIWithSeed(42)
	d1, _ := ai1.DecideTurn(input)

	ai2 := NewRuleBasedAIWithSeed(42)
	d2, _ := ai2.DecideTurn(input)

	if d1 == nil || d2 == nil {
		t.Fatal("decisions should not be nil")
	}
	if d1.Action == nil || d2.Action == nil {
		t.Fatal("actions should not be nil")
	}
	if d1.Action.ActionID != d2.Action.ActionID {
		t.Errorf("same seed different result: %q vs %q", d1.Action.ActionID, d2.Action.ActionID)
	}
	if d1.Reasoning != d2.Reasoning {
		t.Errorf("same seed different reasoning: %q vs %q", d1.Reasoning, d2.Reasoning)
	}
}

func TestDecideTurn_BonusActionPopulated(t *testing.T) {
	t.Parallel()

	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"}},
		},
	}
	bonusBite := models.StructuredAction{
		ID: "bonus_bite", Name: "Bonus Bite", Category: models.ActionCategoryBonus,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 3, DamageType: "piercing"}},
		},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{Ability: models.Ability{Int: 10}, StructuredActions: []models.StructuredAction{claw, bonusBite}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 40, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"npc1": {MaxHP: 50, AC: 12}, "pc1": {MaxHP: 40, AC: 15}},
		Intelligence:     1.0,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("got nil decision")
	}
	if decision.Action == nil || decision.Action.ActionID != "claw" {
		t.Errorf("main action: got %v, want claw", decision.Action)
	}
	if decision.BonusAction == nil {
		t.Fatal("BonusAction is nil, want bonus_bite")
	}
	if decision.BonusAction.ActionID != "bonus_bite" {
		t.Errorf("bonus action: got ActionID=%q, want bonus_bite", decision.BonusAction.ActionID)
	}
	if !strings.Contains(decision.Reasoning, "bonus") {
		t.Errorf("reasoning %q should mention bonus action", decision.Reasoning)
	}
}

func TestDecideTurn_IncapacitatedNoBonusAction(t *testing.T) {
	t.Parallel()

	bonusBite := models.StructuredAction{
		ID: "bonus_bite", Name: "Bonus Bite", Category: models.ActionCategoryBonus,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 3, DamageType: "piercing"}},
		},
	}

	npc := makeParticipant("npc1", false, 50, 0, 0)
	npc.RuntimeState.Conditions = []models.ActiveCondition{
		{Condition: models.ConditionStunned},
	}

	ai := NewRuleBasedAIWithSeed(42)
	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{bonusBite}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 40, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 40, AC: 15}},
		Intelligence:     1.0,
	}

	decision, err := ai.DecideTurn(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision == nil {
		t.Fatal("got nil decision for incapacitated NPC")
	}
	if decision.Action != nil {
		t.Errorf("incapacitated: Action should be nil, got %+v", decision.Action)
	}
	if decision.BonusAction != nil {
		t.Errorf("incapacitated: BonusAction should be nil, got %+v", decision.BonusAction)
	}
}

func TestBuildReasoning_WithBonusAction(t *testing.T) {
	t.Parallel()

	action := &ActionDecision{ActionName: "Claw", ExpectedDamage: 5.0, TargetIDs: []string{"pc1"}}
	bonus := &ActionDecision{ActionName: "Bonus Bite", ExpectedDamage: 3.0}

	r := buildReasoning(RoleBrute, nil, action, bonus)
	if !strings.Contains(r, "Brute") {
		t.Errorf("reasoning %q missing role", r)
	}
	if !strings.Contains(r, "Claw") {
		t.Errorf("reasoning %q missing main action", r)
	}
	if !strings.Contains(r, "bonus Bonus Bite") {
		t.Errorf("reasoning %q missing bonus action", r)
	}
}

func TestBuildReasoning_NilActionWithBonusOnly(t *testing.T) {
	t.Parallel()

	bonus := &ActionDecision{ActionName: "Healing Word", ExpectedDamage: 0}

	r := buildReasoning(RoleCaster, nil, nil, bonus)
	if !strings.Contains(r, "no action") {
		t.Errorf("reasoning %q should say 'no action'", r)
	}
	if !strings.Contains(r, "bonus Healing Word") {
		t.Errorf("reasoning %q missing bonus action", r)
	}
}
