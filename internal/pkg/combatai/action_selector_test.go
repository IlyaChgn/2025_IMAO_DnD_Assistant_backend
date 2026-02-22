package combatai

import (
	"math/rand"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestSelectAction_RechargeReady(t *testing.T) {
	t.Parallel()

	breath := models.StructuredAction{
		ID: "breath", Name: "Fire Breath", Category: models.ActionCategoryAction,
		Recharge: &models.RechargeData{MinRoll: 5},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 15, OnFail: "full damage", OnSuccess: "half damage",
			Damage: []models.DamageRoll{{DiceCount: 8, DiceType: "d6", DamageType: "fire"}},
		},
	}
	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"}},
		},
	}

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.RechargeReady = map[string]bool{"breath": true}

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{breath, claw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:     1.0, // always optimal
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	if decision.ActionID != "breath" {
		t.Errorf("got ActionID=%q, want %q (recharge-ready should be top priority)", decision.ActionID, "breath")
	}
}

func TestSelectAction_MultiattackVsSingle(t *testing.T) {
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

	input := &TurnInput{
		ActiveNPC: makeParticipant("npc1", false, 100, 0, 0),
		CreatureTemplate: models.Creature{
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
		CombatantStats: map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:   1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// Multiattack (3 attacks) should beat any single action.
	if decision.MultiattackGroupID != "multi" {
		t.Errorf("expected multiattack, got ActionID=%q, MultiattackGroupID=%q", decision.ActionID, decision.MultiattackGroupID)
	}
}

func TestSelectAction_BestSingleAction(t *testing.T) {
	t.Parallel()

	weak := models.StructuredAction{
		ID: "dagger", Name: "Dagger", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 3, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 1, DamageType: "piercing"}},
		},
	}
	strong := models.StructuredAction{
		ID: "greatsword", Name: "Greatsword", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 7, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d6", Bonus: 5, DamageType: "slashing"}},
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
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	if decision.ActionID != "greatsword" {
		t.Errorf("got ActionID=%q, want %q (higher EV)", decision.ActionID, "greatsword")
	}
}

func TestSelectAction_UsesExhausted(t *testing.T) {
	t.Parallel()

	limited := models.StructuredAction{
		ID: "web", Name: "Web", Category: models.ActionCategoryAction,
		Uses: &models.UsesData{Max: 1},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 13, OnFail: "restrained", OnSuccess: "no effect",
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d8", DamageType: "bludgeoning"}},
		},
	}
	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 3, DamageType: "slashing"}},
		},
	}

	npc := makeParticipant("npc1", false, 50, 0, 0)
	npc.RuntimeState.Resources.AbilityUses = map[string]int{"web": 0} // exhausted

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{limited, claw}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:     1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	if decision.ActionID != "claw" {
		t.Errorf("got ActionID=%q, want %q (web should be filtered out)", decision.ActionID, "claw")
	}
}

func TestSelectAction_SpellVsWeapon(t *testing.T) {
	t.Parallel()

	weakMelee := models.StructuredAction{
		ID: "staff", Name: "Staff", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 2, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 0, DamageType: "bludgeoning"}},
		},
	}

	qr := &models.SpellQuickRef{Range: "120 feet"}

	npc := makeParticipant("npc1", false, 50, 0, 0)
	npc.RuntimeState.Resources.SpellSlots = map[int]int{3: 2}

	input := &TurnInput{
		ActiveNPC: npc,
		CreatureTemplate: models.Creature{
			StructuredActions: []models.StructuredAction{weakMelee},
			Spellcasting: &models.Spellcasting{
				SpellSaveDC: 15,
				CasterLevel: 5,
				SpellsByLevel: map[int][]models.SpellKnown{
					3: {{Name: "Fireball", Level: 3, QuickRef: qr}},
				},
			},
		},
		Participants:   []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats: map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:   1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleCaster, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// Fireball EV should beat a weak staff.
	if decision.ActionType != models.ActionSpellCast {
		t.Errorf("expected spell_cast, got %q", decision.ActionType)
	}
}

func TestSelectAction_NoSlotsSkipsSpell(t *testing.T) {
	t.Parallel()

	melee := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"}},
		},
	}

	qr := &models.SpellQuickRef{Range: "120 feet"}

	npc := makeParticipant("npc1", false, 50, 0, 0)
	npc.RuntimeState.Resources.SpellSlots = map[int]int{3: 0} // no slots left

	input := &TurnInput{
		ActiveNPC: npc,
		CreatureTemplate: models.Creature{
			StructuredActions: []models.StructuredAction{melee},
			Spellcasting: &models.Spellcasting{
				SpellSaveDC: 15,
				SpellsByLevel: map[int][]models.SpellKnown{
					3: {{Name: "Fireball", Level: 3, QuickRef: qr}},
				},
			},
		},
		Participants:   []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats: map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:   1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleCaster, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	if decision.ActionID != "claw" {
		t.Errorf("got ActionID=%q, want %q (spell should be skipped, no slots)", decision.ActionID, "claw")
	}
}

func TestSelectAction_IntelligenceAlwaysOptimal(t *testing.T) {
	t.Parallel()

	weak := models.StructuredAction{
		ID: "slap", Name: "Slap", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 1, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 0, DamageType: "bludgeoning"}},
		},
	}
	strong := models.StructuredAction{
		ID: "smash", Name: "Smash", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 8, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 3, DiceType: "d10", Bonus: 5, DamageType: "bludgeoning"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 100, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{weak, strong}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     1.0, // always optimal
	}

	// Run 20 times — at Intelligence 1.0, rng.Float64() is always < 1.0, so always optimal.
	for i := 0; i < 20; i++ {
		rng := rand.New(rand.NewSource(int64(i)))
		decision := SelectAction(input, RoleBrute, rng)
		if decision == nil {
			t.Fatal("SelectAction returned nil")
		}
		if decision.ActionID != "smash" {
			t.Errorf("iteration %d: got %q, want %q (Intelligence=1.0 should always pick optimal)", i, decision.ActionID, "smash")
		}
	}
}

func TestSelectAction_NoCandidates(t *testing.T) {
	t.Parallel()

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{}, // no actions at all
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		Intelligence:     0.50,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision != nil {
		t.Errorf("no candidates: got %+v, want nil", decision)
	}
}
