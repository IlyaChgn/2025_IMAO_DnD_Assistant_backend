package combatai

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// --- ShouldSpendSpellSlot tests ---

func TestShouldSpendSpellSlot_CantripsAlwaysAllowed(t *testing.T) {
	t.Parallel()

	// Cantrips (level 0) should always be allowed, regardless of conditions.
	if !ShouldSpendSpellSlot(0, 10, 1.0, 0.90, 5) {
		t.Error("cantrips should always be allowed")
	}
}

func TestShouldSpendSpellSlot_HPEmergencyOverridesAll(t *testing.T) {
	t.Parallel()

	// HP < 30% should override all economy restrictions.
	// High intelligence, late round, top-level slot — should still allow.
	if !ShouldSpendSpellSlot(5, 10, 0.20, 0.90, 5) {
		t.Error("HP emergency (20%) should allow any slot")
	}
	if !ShouldSpendSpellSlot(5, 10, 0.29, 0.90, 5) {
		t.Error("HP emergency (29%) should allow any slot")
	}
}

func TestShouldSpendSpellSlot_LowINT_AlwaysAllow(t *testing.T) {
	t.Parallel()

	// Intelligence < 0.55: no resource management.
	if !ShouldSpendSpellSlot(5, 10, 1.0, 0.30, 5) {
		t.Error("low INT should always allow — round 10, level 5")
	}
	if !ShouldSpendSpellSlot(5, 1, 1.0, 0.50, 5) {
		t.Error("low INT should always allow — round 1, level 5")
	}
}

func TestShouldSpendSpellSlot_MedINT_BlocksTopOnRound1(t *testing.T) {
	t.Parallel()

	// Intelligence 0.55–0.75: block top-level slot on round 1.
	if ShouldSpendSpellSlot(5, 1, 1.0, 0.60, 5) {
		t.Error("medium INT should block top-level (5) on round 1")
	}
}

func TestShouldSpendSpellSlot_MedINT_AllowsTopOnRound2(t *testing.T) {
	t.Parallel()

	if !ShouldSpendSpellSlot(5, 2, 1.0, 0.60, 5) {
		t.Error("medium INT should allow top-level on round 2")
	}
}

func TestShouldSpendSpellSlot_MedINT_AllowsNonTopOnRound1(t *testing.T) {
	t.Parallel()

	// Non-top-level (3 < 5) should be allowed even on round 1.
	if !ShouldSpendSpellSlot(3, 1, 1.0, 0.60, 5) {
		t.Error("medium INT should allow non-top-level (3) on round 1")
	}
}

func TestShouldSpendSpellSlot_HighINT_AllowAllRound1(t *testing.T) {
	t.Parallel()

	// High intelligence, early round: all levels allowed.
	if !ShouldSpendSpellSlot(5, 1, 1.0, 0.80, 5) {
		t.Error("high INT should allow top-level on round 1 (opening salvo)")
	}
}

func TestShouldSpendSpellSlot_HighINT_AllowAllRound2(t *testing.T) {
	t.Parallel()

	if !ShouldSpendSpellSlot(5, 2, 1.0, 0.80, 5) {
		t.Error("high INT should allow top-level on round 2")
	}
}

func TestShouldSpendSpellSlot_HighINT_MediumOnlyRound4(t *testing.T) {
	t.Parallel()

	// Round 3-5: medium levels only. For maxLevel=5: midCutoff = (5+1)/2 = 3.
	if ShouldSpendSpellSlot(5, 4, 1.0, 0.80, 5) {
		t.Error("high INT round 4: level 5 should be blocked (mid-cutoff = 3)")
	}
	if ShouldSpendSpellSlot(4, 4, 1.0, 0.80, 5) {
		t.Error("high INT round 4: level 4 should be blocked (mid-cutoff = 3)")
	}
	if !ShouldSpendSpellSlot(3, 4, 1.0, 0.80, 5) {
		t.Error("high INT round 4: level 3 should be allowed (== mid-cutoff)")
	}
	if !ShouldSpendSpellSlot(2, 4, 1.0, 0.80, 5) {
		t.Error("high INT round 4: level 2 should be allowed (< mid-cutoff)")
	}
}

func TestShouldSpendSpellSlot_HighINT_ConserveRound7(t *testing.T) {
	t.Parallel()

	// Round 6+: cantrips + level 1-2 only.
	if ShouldSpendSpellSlot(3, 7, 1.0, 0.80, 5) {
		t.Error("high INT round 7: level 3 should be blocked (conserve mode)")
	}
	if !ShouldSpendSpellSlot(2, 7, 1.0, 0.80, 5) {
		t.Error("high INT round 7: level 2 should be allowed")
	}
	if !ShouldSpendSpellSlot(1, 7, 1.0, 0.80, 5) {
		t.Error("high INT round 7: level 1 should be allowed")
	}
}

func TestShouldSpendSpellSlot_HighINT_MedCutoff_MaxLevel9(t *testing.T) {
	t.Parallel()

	// For maxLevel=9: midCutoff = (9+1)/2 = 5.
	if !ShouldSpendSpellSlot(5, 4, 1.0, 0.80, 9) {
		t.Error("high INT round 4, maxLevel 9: level 5 should be allowed (mid-cutoff = 5)")
	}
	if ShouldSpendSpellSlot(6, 4, 1.0, 0.80, 9) {
		t.Error("high INT round 4, maxLevel 9: level 6 should be blocked (> mid-cutoff)")
	}
}

func TestShouldSpendSpellSlot_HighINT_MedCutoff_MaxLevel3(t *testing.T) {
	t.Parallel()

	// For maxLevel=3: midCutoff = (3+1)/2 = 2.
	if !ShouldSpendSpellSlot(2, 4, 1.0, 0.80, 3) {
		t.Error("high INT round 4, maxLevel 3: level 2 should be allowed (mid-cutoff = 2)")
	}
	if ShouldSpendSpellSlot(3, 4, 1.0, 0.80, 3) {
		t.Error("high INT round 4, maxLevel 3: level 3 should be blocked (> mid-cutoff)")
	}
}

// --- maxAvailableSlotLevel tests ---

func TestMaxAvailableSlotLevel_Mixed(t *testing.T) {
	t.Parallel()

	slots := map[int]int{1: 2, 3: 0, 5: 1}
	if got := maxAvailableSlotLevel(slots); got != 5 {
		t.Errorf("maxAvailableSlotLevel: got %d, want 5", got)
	}
}

func TestMaxAvailableSlotLevel_Empty(t *testing.T) {
	t.Parallel()

	if got := maxAvailableSlotLevel(map[int]int{}); got != 0 {
		t.Errorf("maxAvailableSlotLevel empty: got %d, want 0", got)
	}
}

func TestMaxAvailableSlotLevel_AllExhausted(t *testing.T) {
	t.Parallel()

	slots := map[int]int{1: 0, 2: 0, 3: 0}
	if got := maxAvailableSlotLevel(slots); got != 0 {
		t.Errorf("maxAvailableSlotLevel all exhausted: got %d, want 0", got)
	}
}

func TestMaxAvailableSlotLevel_Nil(t *testing.T) {
	t.Parallel()

	if got := maxAvailableSlotLevel(nil); got != 0 {
		t.Errorf("maxAvailableSlotLevel nil: got %d, want 0", got)
	}
}

// --- ShouldUseAbility tests ---

func TestShouldUseAbility_LowINT_AlwaysTrue(t *testing.T) {
	t.Parallel()

	if !ShouldUseAbility(5.0, 10.0, 0.30) {
		t.Error("low INT should always allow ability use")
	}
}

func TestShouldUseAbility_SmartWorthIt(t *testing.T) {
	t.Parallel()

	// abilityEV=15, baseline=8 → 1.5 * 8 = 12. 15 >= 12 → true.
	if !ShouldUseAbility(15.0, 8.0, 0.60) {
		t.Error("smart NPC: ability EV 15 >= 150%% of baseline 8 → should allow")
	}
}

func TestShouldUseAbility_SmartNotWorthIt(t *testing.T) {
	t.Parallel()

	// abilityEV=10, baseline=8 → 1.5 * 8 = 12. 10 < 12 → false.
	if ShouldUseAbility(10.0, 8.0, 0.60) {
		t.Error("smart NPC: ability EV 10 < 150%% of baseline 8 → should block")
	}
}

func TestShouldUseAbility_SmartExactThreshold(t *testing.T) {
	t.Parallel()

	// abilityEV=12, baseline=8 → 1.5 * 8 = 12.0. 12 >= 12 → true.
	if !ShouldUseAbility(12.0, 8.0, 0.60) {
		t.Error("smart NPC: ability EV exactly at 150%% threshold → should allow")
	}
}

func TestShouldUseAbility_ZeroBaseline(t *testing.T) {
	t.Parallel()

	// No baseline weapons → always allow.
	if !ShouldUseAbility(5.0, 0.0, 0.80) {
		t.Error("zero baseline should always allow ability use")
	}
}

// --- bestBaselineEV tests ---

func TestBestBaselineEV_SkipsLimited(t *testing.T) {
	t.Parallel()

	claw := models.StructuredAction{
		ID: "claw", Name: "Claw", Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type: models.AttackRollMeleeWeapon, Bonus: 5, Reach: 5,
			Damage: []models.DamageRoll{{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"}},
		},
	}
	web := models.StructuredAction{
		ID: "web", Name: "Web", Category: models.ActionCategoryAction,
		Uses: &models.UsesData{Max: 1},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 15, OnSuccess: "no effect",
			Damage: []models.DamageRoll{{DiceCount: 4, DiceType: "d10", DamageType: "bludgeoning"}},
		},
	}
	breath := models.StructuredAction{
		ID: "breath", Name: "Fire Breath", Category: models.ActionCategoryAction,
		Recharge: &models.RechargeData{MinRoll: 5},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 17, OnSuccess: "half damage",
			Damage: []models.DamageRoll{{DiceCount: 12, DiceType: "d6", DamageType: "fire"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{claw, web, breath}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:     0.50,
	}

	ev := bestBaselineEV(input)
	// Only claw should be considered (web has Uses, breath has Recharge).
	// Claw: hit_chance = (21 - (15-5))/20 = 11/20 = 0.55, avg_damage = 1*(8+1)/2+3 = 7.5
	// EV = 0.55 * 7.5 + 0.05 * 7.5 = 4.125 + 0.375 = 4.5
	if ev < 4.0 || ev > 5.0 {
		t.Errorf("bestBaselineEV: got %.2f, want ~4.5 (only claw should count)", ev)
	}
}

func TestBestBaselineEV_NoBaseline(t *testing.T) {
	t.Parallel()

	// Only resource-gated actions — baseline should be 0.
	web := models.StructuredAction{
		ID: "web", Name: "Web", Category: models.ActionCategoryAction,
		Uses: &models.UsesData{Max: 1},
		SavingThrow: &models.SavingThrowData{
			Ability: models.AbilityDEX, DC: 13, OnSuccess: "no effect",
			Damage: []models.DamageRoll{{DiceCount: 2, DiceType: "d8", DamageType: "bludgeoning"}},
		},
	}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: models.Creature{StructuredActions: []models.StructuredAction{web}},
		Participants:     []models.ParticipantFull{makeParticipant("pc1", true, 50, 1, 0)},
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15, SaveBonuses: map[string]int{"DEX": 2}}},
		Intelligence:     0.50,
	}

	ev := bestBaselineEV(input)
	if ev != 0 {
		t.Errorf("bestBaselineEV with no baseline weapons: got %.2f, want 0", ev)
	}
}
