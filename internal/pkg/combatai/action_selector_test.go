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

// --- Resource Manager integration tests ---

// makeCasterInput builds a TurnInput for a caster NPC with a staff, cantrip,
// and a leveled spell. Used by resource economy integration tests.
func makeCasterInput(hp int, round int, intelligence float64, slots map[int]int) *TurnInput {
	staff := models.StructuredAction{
		ID: "staff", Name: "Staff", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 2, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 0, DamageType: "bludgeoning"}},
		},
	}
	qr := &models.SpellQuickRef{Range: "120 feet"}

	npc := makeParticipant("npc1", false, hp, 0, 0)
	npc.RuntimeState.Resources.SpellSlots = slots

	return &TurnInput{
		ActiveNPC: npc,
		CreatureTemplate: models.Creature{
			StructuredActions: []models.StructuredAction{staff},
			Spellcasting: &models.Spellcasting{
				SpellSaveDC: 15,
				CasterLevel: 9,
				SpellsByLevel: map[int][]models.SpellKnown{
					0: {{Name: "Fire Bolt", Level: 0, QuickRef: qr}},
					5: {{Name: "Cone of Cold", Level: 5, QuickRef: qr}},
				},
			},
		},
		Participants:   []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats: map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		CurrentRound:   round,
		Intelligence:   intelligence,
	}
}

func TestSelectAction_SmartCaster_ConservesRound7(t *testing.T) {
	t.Parallel()

	// High INT caster in round 7 should conserve: use cantrip, not level 5.
	input := makeCasterInput(100, 7, 0.80, map[int]int{5: 2})

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleCaster, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// Level 5 should be blocked in round 7 (conserve mode: only 1-2 allowed).
	// Should pick cantrip or staff.
	if decision.SlotLevel == 5 {
		t.Errorf("smart caster round 7: should NOT use level 5 slot, got SlotLevel=%d ActionID=%q",
			decision.SlotLevel, decision.ActionID)
	}
}

func TestSelectAction_SmartCaster_SpendsAtLowHP(t *testing.T) {
	t.Parallel()

	// High INT caster at 20% HP in round 7: emergency override, use level 5.
	// NPC MaxHP = 100 (from makeParticipant), HP = 20 → 20%.
	// Use Intelligence=1.0 to guarantee optimal pick (removes randomness).
	input := makeCasterInput(20, 7, 1.0, map[int]int{5: 2})

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleCaster, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// At 20% HP, emergency override should allow level 5.
	// Cone of Cold (L5) has higher EV than cantrip → must be picked at Intelligence=1.0.
	if decision.SlotLevel != 5 {
		t.Errorf("smart caster low HP: got SlotLevel=%d ActionID=%q, want SlotLevel=5 (emergency override)",
			decision.SlotLevel, decision.ActionID)
	}
}

func TestSelectAction_MedCaster_BlocksTopRound1(t *testing.T) {
	t.Parallel()

	// Medium INT caster on round 1 should block top-level slot.
	// Give both L2 and L5 slots so it has options.
	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.SpellSlots = map[int]int{2: 2, 5: 2}

	qr := &models.SpellQuickRef{Range: "120 feet"}

	input := &TurnInput{
		ActiveNPC: npc,
		CreatureTemplate: models.Creature{
			StructuredActions: []models.StructuredAction{
				{
					ID: "staff", Name: "Staff", Category: models.ActionCategoryAction,
					Attack: &models.AttackRollData{
						Type: models.AttackRollMeleeWeapon, Bonus: 2, Reach: 5,
						Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d6", Bonus: 0, DamageType: "bludgeoning"}},
					},
				},
			},
			Spellcasting: &models.Spellcasting{
				SpellSaveDC: 15,
				CasterLevel: 9,
				SpellsByLevel: map[int][]models.SpellKnown{
					2: {{Name: "Scorching Ray", Level: 2, QuickRef: qr}},
					5: {{Name: "Cone of Cold", Level: 5, QuickRef: qr}},
				},
			},
		},
		Participants:   []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats: map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		CurrentRound:   1,
		Intelligence:   0.60, // medium — blocks top-level on round 1
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleCaster, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// Should NOT pick level 5 (top-level blocked on round 1 at medium INT).
	if decision.SlotLevel == 5 {
		t.Errorf("medium INT round 1: should NOT use top-level (5), got SlotLevel=%d", decision.SlotLevel)
	}
}

func TestSelectAction_DumbCaster_UsesEverything(t *testing.T) {
	t.Parallel()

	// Low INT caster in round 7 should still use level 5 — no economy.
	input := makeCasterInput(100, 7, 0.30, map[int]int{5: 2})

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleCaster, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// Low INT: no resource management → L5 should be available.
	// With intelligence=0.30 and random selection, it may or may not pick L5,
	// but we verify L5 is at least a candidate by running multiple seeds.
	foundL5 := false
	for seed := int64(0); seed < 50; seed++ {
		r := rand.New(rand.NewSource(seed))
		d := SelectAction(input, RoleCaster, r)
		if d != nil && d.SlotLevel == 5 {
			foundL5 = true
			break
		}
	}
	if !foundL5 {
		t.Error("low INT round 7: level 5 should be available (no economy), but never selected in 50 tries")
	}
}

func TestSelectAction_SmartNPC_SkipsWeakAbility(t *testing.T) {
	t.Parallel()

	// Strong baseline weapon + weak limited-use ability.
	// Smart NPC should skip the ability (EV < 150% baseline).
	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 7, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d6", Bonus: 4, DamageType: "slashing"}},
		},
	}
	// Weak limited-use ability: low damage save-based.
	weakAbility := models.StructuredAction{
		ID: "web", Name: "Web Spray", Category: models.ActionCategoryAction,
		Uses: &models.UsesData{Max: 1},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 12, OnSuccess: "no effect",
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", DamageType: "bludgeoning"}},
		},
	}

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.AbilityUses = map[string]int{"web": 1}

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{claw, weakAbility}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:     1.0, // always optimal
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// Claw is baseline. Weak ability EV << 150% of claw EV → skipped.
	if decision.ActionID != "claw" {
		t.Errorf("smart NPC: got %q, want %q (weak ability should be skipped)", decision.ActionID, "claw")
	}
}

func TestSelectAction_SmartNPC_UsesStrongAbility(t *testing.T) {
	t.Parallel()

	// Weak baseline weapon + strong limited-use ability.
	// Smart NPC should use the ability (EV >= 150% baseline).
	dagger := models.StructuredAction{
		ID: "dagger", Name: "Dagger", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 2, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d4", Bonus: 0, DamageType: "piercing"}},
		},
	}
	// Strong limited-use ability: high damage.
	strongAbility := models.StructuredAction{
		ID: "acid_spray", Name: "Acid Spray", Category: models.ActionCategoryAction,
		Uses: &models.UsesData{Max: 1},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 15, OnSuccess: "half damage",
			Damage: []models.DamageRoll{{DiceCount: 6, DiceType: "d8", DamageType: "acid"}},
		},
	}

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.AbilityUses = map[string]int{"acid_spray": 1}

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{dagger, strongAbility}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:     1.0, // always optimal
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// Acid Spray EV >> 150% of dagger EV → should be used.
	if decision.ActionID != "acid_spray" {
		t.Errorf("smart NPC: got %q, want %q (strong ability should be used)", decision.ActionID, "acid_spray")
	}
}

// --- AoE integration tests ---

// makeAoEBreathWeapon creates a recharge breath weapon with a 15ft cone AoE.
func makeAoEBreathWeapon() models.StructuredAction {
	return models.StructuredAction{
		ID: "fire_breath", Name: "Fire Breath", Category: models.ActionCategoryAction,
		Recharge: &models.RechargeData{MinRoll: 5},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 15, OnFail: "full damage", OnSuccess: "half damage",
			Damage: []models.DamageRoll{{DiceCount: 8, DiceType: "d6", DamageType: "fire"}},
			Area:   &models.AreaOfEffect{Shape: models.AreaShapeCone, Size: 15},
		},
	}
}

func TestSelectAction_AoE_BreathWeapon3Targets(t *testing.T) {
	t.Parallel()

	// Dragon at (0,0) with Fire Breath (15ft cone). 3 PCs in a line ahead.
	// All 3 should be in the cone → EV tripled, all 3 in TargetIDs.
	breath := makeAoEBreathWeapon()
	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 6, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 4, DamageType: "slashing"}},
		},
	}

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.RechargeReady = map[string]bool{"fire_breath": true}

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{breath, claw}},
		Participants: []models.ParticipantFull{
			makeParticipant("pc1", true, 50, 1, 0),
			makeParticipant("pc2", true, 50, 2, 0),
			makeParticipant("pc3", true, 50, 3, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
			"pc2": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
			"pc3": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
		},
		Intelligence: 1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	if decision.ActionID != "fire_breath" {
		t.Errorf("AoE 3 targets: got ActionID=%q, want fire_breath", decision.ActionID)
	}
	// Should have 3 targets (all PCs in the cone).
	if len(decision.TargetIDs) != 3 {
		t.Errorf("AoE 3 targets: got %d TargetIDs, want 3", len(decision.TargetIDs))
	}
	// EV should be ~3x single-target.
	// Single-target EV for 8d6 DC15 vs DEX+2, half on success:
	// fail = 0.60, avg = 28, EV = 0.60*28 + 0.40*14 = 16.8 + 5.6 = 22.4
	// 3-target: 22.4 * 3 = 67.2
	if decision.ExpectedDamage < 60.0 || decision.ExpectedDamage > 75.0 {
		t.Errorf("AoE 3 targets: got EV=%.1f, want ~67.2 (22.4 * 3)", decision.ExpectedDamage)
	}
}

func TestSelectAction_AoE_BreathWeapon1Target(t *testing.T) {
	t.Parallel()

	// Dragon at (0,0). Only 1 PC in cone direction, 1 behind NPC.
	// AoE should not inflate EV (only 1 target in cone).
	breath := makeAoEBreathWeapon()
	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 6, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 4, DamageType: "slashing"}},
		},
	}

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.RechargeReady = map[string]bool{"fire_breath": true}

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{breath, claw}},
		Participants: []models.ParticipantFull{
			makeParticipant("pc1", true, 50, 2, 0),   // in front
			makeParticipant("pc2", true, 50, -10, 0), // far behind
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
			"pc2": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
		},
		Intelligence: 1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	if decision.ActionID != "fire_breath" {
		t.Errorf("AoE 1 target: got ActionID=%q, want fire_breath (recharge priority)", decision.ActionID)
	}
	// Only 1 target in cone → EV should be single-target (~22.4).
	if decision.ExpectedDamage > 30.0 {
		t.Errorf("AoE 1 target: got EV=%.1f, want ~22.4 (no AoE boost)", decision.ExpectedDamage)
	}
}

func TestSelectAction_AoE_NilCoords_SingleTarget(t *testing.T) {
	t.Parallel()

	// NPC with nil coordinates → AoE fallback to single-target EV.
	breath := makeAoEBreathWeapon()

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.CellsCoords = nil // no grid coordinates
	npc.RuntimeState.Resources.RechargeReady = map[string]bool{"fire_breath": true}

	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{breath}},
		Participants: []models.ParticipantFull{
			makeParticipant("pc1", true, 50, 1, 0),
			makeParticipant("pc2", true, 50, 2, 0),
			makeParticipant("pc3", true, 50, 3, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
			"pc2": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
			"pc3": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
		},
		Intelligence: 1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// With nil NPC coords, FindAoETargets returns nil → no AoE boost.
	// EV should be single-target (~22.4).
	if decision.ExpectedDamage > 30.0 {
		t.Errorf("nil coords AoE: got EV=%.1f, want ~22.4 (no boost)", decision.ExpectedDamage)
	}
}

func TestSelectAction_AoE_BoostsRechargeEV(t *testing.T) {
	t.Parallel()

	// Verify that a recharge AoE breath weapon has its EV boosted
	// in the returned ActionDecision when multiple targets are in the cone.
	breath := makeAoEBreathWeapon()

	npc := makeParticipant("npc1", false, 100, 0, 0)
	npc.RuntimeState.Resources.RechargeReady = map[string]bool{"fire_breath": true}

	// 2 PCs in the cone.
	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{breath}},
		Participants: []models.ParticipantFull{
			makeParticipant("pc1", true, 50, 1, 0),
			makeParticipant("pc2", true, 50, 2, 0),
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
			"pc2": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
		},
		Intelligence: 1.0,
	}

	rng := rand.New(rand.NewSource(42))
	decision := SelectAction(input, RoleBrute, rng)
	if decision == nil {
		t.Fatal("SelectAction returned nil")
	}
	// EV should be ~2x single-target: 22.4 * 2 = 44.8
	if decision.ExpectedDamage < 40.0 || decision.ExpectedDamage > 50.0 {
		t.Errorf("AoE recharge boost: got EV=%.1f, want ~44.8 (22.4 * 2)", decision.ExpectedDamage)
	}
	if len(decision.TargetIDs) != 2 {
		t.Errorf("AoE recharge boost: got %d TargetIDs, want 2", len(decision.TargetIDs))
	}
}
