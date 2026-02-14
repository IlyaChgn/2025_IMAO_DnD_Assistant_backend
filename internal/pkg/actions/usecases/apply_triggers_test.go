package usecases

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/triggers"
)

// --- helpers ---

func alwaysSucceed() float32 { return 0.0 }
func alwaysFail() float32   { return 1.0 }

func makeTarget(hp, tempHP int) *models.ParticipantFull {
	return &models.ParticipantFull{
		InstanceID: "target-1",
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: hp,
			MaxHP:     hp,
			TempHP:    tempHP,
		},
	}
}

func makeAttackerPC(hp, tempHP int) *models.ParticipantFull {
	return &models.ParticipantFull{
		InstanceID:        "attacker-1",
		IsPlayerCharacter: true,
		CharacterRuntime: &models.CharacterRuntime{
			CharacterID: "char-abc",
			CurrentHP:   hp,
			TemporaryHP: tempHP,
		},
	}
}

func flameTongueTrigger() models.TriggerEffect {
	return models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.0,
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "2d6",
				"damageType": "fire",
			},
		},
	}
}

func makeOwner() *models.ParticipantFull {
	return &models.ParticipantFull{
		InstanceID: "owner-1",
	}
}

func testOpts() triggerOpts {
	return triggerOpts{
		SourceID:   "weapon-1",
		SourceName: "Flame Tongue",
		RandFloat:  alwaysSucceed,
	}
}

func testOptsWithStats(ts *TargetStats) triggerOpts {
	o := testOpts()
	o.TargetStats = ts
	return o
}

// --- tests ---

func TestApplyTriggerResults_DealDamage(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{flameTongueTrigger()}

	results, changes := applyTriggerResults(triggers, testOpts(), models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Skipped {
		t.Fatal("expected result not to be skipped")
	}
	if r.DamageResult == nil {
		t.Fatal("expected DamageResult to be set")
	}
	if r.DamageResult.Total < 2 || r.DamageResult.Total > 12 {
		t.Errorf("2d6 total out of range: %d", r.DamageResult.Total)
	}

	// Target HP should be reduced
	expectedHP := 50 - r.DamageResult.Total
	if target.RuntimeState.CurrentHP != expectedHP {
		t.Errorf("expected target HP %d, got %d", expectedHP, target.RuntimeState.CurrentHP)
	}

	// StateChange should exist
	if len(changes) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(changes))
	}
	if changes[0].HPDelta != -r.DamageResult.Total {
		t.Errorf("expected HPDelta %d, got %d", -r.DamageResult.Total, changes[0].HPDelta)
	}
	if changes[0].TargetID != "target-1" {
		t.Errorf("expected TargetID 'target-1', got %q", changes[0].TargetID)
	}
}

func TestApplyTriggerResults_Heal(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(20, 0)
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.0,
		Effect: models.Effect{
			Type:   models.EffectHeal,
			Params: map[string]interface{}{"amount": float64(5)},
		},
	}}

	opts := testOpts()
	opts.SourceName = "Vampiric Sword"
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 || results[0].Skipped {
		t.Fatalf("expected 1 non-skipped result, got %d", len(results))
	}
	if attacker.CharacterRuntime.CurrentHP != 25 {
		t.Errorf("expected attacker HP 25, got %d", attacker.CharacterRuntime.CurrentHP)
	}
	if len(changes) != 1 || changes[0].HPDelta != 5 {
		t.Errorf("expected HPDelta +5, got %+v", changes)
	}
	if changes[0].TargetID != "attacker-1" {
		t.Errorf("expected TargetID 'attacker-1', got %q", changes[0].TargetID)
	}
}

func TestApplyTriggerResults_GrantTempHP(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.0,
		Effect: models.Effect{
			Type:   models.EffectGrantTempHP,
			Params: map[string]interface{}{"amount": float64(8)},
		},
	}}

	opts := testOpts()
	opts.SourceName = "Fiendish Blade"
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 || results[0].Skipped {
		t.Fatalf("expected 1 non-skipped result, got %d", len(results))
	}
	if attacker.CharacterRuntime.TemporaryHP != 8 {
		t.Errorf("expected attacker temp HP 8, got %d", attacker.CharacterRuntime.TemporaryHP)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(changes))
	}
}

func TestApplyTriggerResults_GrantTempHP_NoStackLower(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 10) // already has 10 temp HP
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.0,
		Effect: models.Effect{
			Type:   models.EffectGrantTempHP,
			Params: map[string]interface{}{"amount": float64(5)}, // lower than current
		},
	}}

	opts := testOpts()
	opts.SourceName = "Fiendish Blade"
	_, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	// Temp HP should NOT be replaced (5 < 10)
	if attacker.CharacterRuntime.TemporaryHP != 10 {
		t.Errorf("expected temp HP unchanged at 10, got %d", attacker.CharacterRuntime.TemporaryHP)
	}
	// No state change since temp HP was not applied
	if len(changes) != 0 {
		t.Errorf("expected 0 state changes (temp HP not replaced), got %d", len(changes))
	}
}

func TestApplyTriggerResults_NoTriggers(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)

	results, changes := applyTriggerResults(nil, testOpts(), models.ItemTriggerOnHit, false, target, attacker)

	if results != nil {
		t.Errorf("expected nil results, got %d", len(results))
	}
	if changes != nil {
		t.Errorf("expected nil changes, got %d", len(changes))
	}
}

func TestApplyTriggerResults_OnCriticalOnly(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnCritical,
		Chance:  1.0,
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "3d6",
				"damageType": "radiant",
			},
		},
	}}

	opts := testOpts()
	opts.SourceName = "Holy Avenger"

	// Non-crit: on_critical trigger should not fire
	results, _ := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)
	if len(results) != 0 {
		t.Fatalf("expected 0 results on non-crit, got %d", len(results))
	}

	// Crit: on_critical trigger should fire
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, true, target, attacker)
	if len(results) != 1 {
		t.Fatalf("expected 1 result on crit, got %d", len(results))
	}
	if results[0].Skipped {
		t.Fatal("expected result not to be skipped")
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(changes))
	}
}

func TestApplyTriggerResults_CritDispatchesBothEvents(t *testing.T) {
	target := makeTarget(100, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{
		// on_hit trigger (always fires)
		{
			Trigger: models.ItemTriggerOnHit,
			Chance:  1.0,
			Effect: models.Effect{
				Type: models.EffectDealDamage,
				Params: map[string]interface{}{
					"dice":       "2d6",
					"damageType": "fire",
				},
			},
		},
		// on_critical trigger (fires only on crit)
		{
			Trigger: models.ItemTriggerOnCritical,
			Chance:  1.0,
			Effect: models.Effect{
				Type: models.EffectDealDamage,
				Params: map[string]interface{}{
					"dice":       "3d6",
					"damageType": "radiant",
				},
			},
		},
	}

	opts := testOpts()
	opts.SourceName = "Holy Flame Blade"
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, true, target, attacker)

	// Both triggers should fire on crit
	if len(results) != 2 {
		t.Fatalf("expected 2 results (on_hit + on_critical), got %d", len(results))
	}
	if results[0].Event != models.ItemTriggerOnHit {
		t.Errorf("first result: expected on_hit event, got %q", results[0].Event)
	}
	if results[1].Event != models.ItemTriggerOnCritical {
		t.Errorf("second result: expected on_critical event, got %q", results[1].Event)
	}

	// Both should produce state changes
	if len(changes) != 2 {
		t.Fatalf("expected 2 state changes, got %d", len(changes))
	}
}

func TestApplyTriggerResults_ChanceSkipped_NoSideEffects(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnHit,
		Chance:  0.5,
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "2d6",
				"damageType": "fire",
			},
		},
	}}

	opts := testOpts()
	opts.RandFloat = alwaysFail // returns 1.0, which is >= 0.5 → skipped
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Skipped {
		t.Fatal("expected result to be skipped (chance failed)")
	}
	if results[0].SkipReason != "chance" {
		t.Errorf("expected SkipReason 'chance', got %q", results[0].SkipReason)
	}

	// No side effects
	if len(changes) != 0 {
		t.Errorf("expected 0 state changes, got %d", len(changes))
	}
	if target.RuntimeState.CurrentHP != 50 {
		t.Errorf("target HP should be unchanged at 50, got %d", target.RuntimeState.CurrentHP)
	}
}

func TestApplyTriggerResults_ApplyCondition_NoHPMutation(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.0,
		Effect: models.Effect{
			Type: models.EffectApplyCondition,
			Params: map[string]interface{}{
				"condition": "frightened",
				"duration":  "1 minute",
			},
		},
	}}

	opts := testOpts()
	opts.SourceName = "Mace of Terror"
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 || results[0].Skipped {
		t.Fatalf("expected 1 non-skipped result, got %d", len(results))
	}
	if results[0].ConditionResult == nil {
		t.Fatal("expected ConditionResult to be set")
	}
	if results[0].ConditionResult.Condition != "frightened" {
		t.Errorf("expected condition 'frightened', got %q", results[0].ConditionResult.Condition)
	}

	// No HP mutation — no state changes
	if len(changes) != 0 {
		t.Errorf("expected 0 state changes for apply_condition, got %d", len(changes))
	}

	// HP unchanged
	if target.RuntimeState.CurrentHP != 50 {
		t.Errorf("target HP should be unchanged at 50, got %d", target.RuntimeState.CurrentHP)
	}
}

func TestApplyTriggerResults_MultipleTriggers(t *testing.T) {
	target := makeTarget(100, 0)
	attacker := makeAttackerPC(20, 0)
	triggers := []models.TriggerEffect{
		flameTongueTrigger(), // deal_damage 2d6 fire
		{
			Trigger: models.ItemTriggerOnHit,
			Chance:  1.0,
			Effect: models.Effect{
				Type:   models.EffectHeal,
				Params: map[string]interface{}{"amount": float64(3)},
			},
		},
		{
			Trigger: models.ItemTriggerOnHit,
			Chance:  1.0,
			Effect: models.Effect{
				Type: models.EffectApplyCondition,
				Params: map[string]interface{}{
					"condition": "poisoned",
				},
			},
		},
	}

	opts := testOpts()
	opts.SourceName = "Venomous Flame Blade"
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	// All 3 triggers should fire
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// First: deal_damage — target damaged, 1 state change
	// Second: heal — attacker healed, 1 state change
	// Third: apply_condition — no state change
	if len(changes) != 2 {
		t.Fatalf("expected 2 state changes (damage + heal), got %d", len(changes))
	}

	// Verify attacker was healed
	if attacker.CharacterRuntime.CurrentHP != 23 {
		t.Errorf("expected attacker HP 23, got %d", attacker.CharacterRuntime.CurrentHP)
	}
}

func TestApplyTriggerResults_CreatureTarget(t *testing.T) {
	target := &models.ParticipantFull{
		InstanceID:  "goblin-1",
		DisplayName: "Goblin Chief",
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: 30,
			MaxHP:     30,
		},
	}
	attacker := makeAttackerPC(25, 0)
	triggers := []models.TriggerEffect{flameTongueTrigger()}

	results, changes := applyTriggerResults(triggers, testOpts(), models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 || results[0].Skipped {
		t.Fatalf("expected 1 non-skipped result, got %d", len(results))
	}

	dmg := results[0].DamageResult.Total
	expectedHP := 30 - dmg
	if target.RuntimeState.CurrentHP != expectedHP {
		t.Errorf("expected creature HP %d, got %d", expectedHP, target.RuntimeState.CurrentHP)
	}
	if len(changes) != 1 || changes[0].TargetID != "goblin-1" {
		t.Errorf("expected state change for goblin-1, got %+v", changes)
	}
}

func TestApplyTriggerResults_HealCreatureCapped(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := &models.ParticipantFull{
		InstanceID: "vampire-1",
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: 28,
			MaxHP:     30,
		},
	}
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.0,
		Effect: models.Effect{
			Type:   models.EffectHeal,
			Params: map[string]interface{}{"amount": float64(10)},
		},
	}}

	opts := testOpts()
	opts.SourceName = "Vampiric Blade"
	applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	// Creature heal capped at maxHP
	if attacker.RuntimeState.CurrentHP != 30 {
		t.Errorf("expected creature HP capped at 30, got %d", attacker.RuntimeState.CurrentHP)
	}
}

func TestApplyTriggerResults_DealDamage_FireImmunity(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{flameTongueTrigger()} // 2d6 fire

	ts := &TargetStats{
		Name:       "Fire Elemental",
		AC:         13,
		Immunities: []string{"fire"},
	}
	opts := testOptsWithStats(ts)
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 || results[0].Skipped {
		t.Fatalf("expected 1 non-skipped result, got %d", len(results))
	}

	// Fire immune — 0 damage applied
	if target.RuntimeState.CurrentHP != 50 {
		t.Errorf("expected target HP unchanged at 50 (fire immune), got %d", target.RuntimeState.CurrentHP)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(changes))
	}
	if changes[0].HPDelta != 0 {
		t.Errorf("expected HPDelta 0 (immunity), got %d", changes[0].HPDelta)
	}
}

func TestApplyTriggerResults_DealDamage_Resistance(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.0,
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "2d6",
				"damageType": "fire",
			},
		},
	}}

	ts := &TargetStats{
		Name:        "Red Dragon Wyrmling",
		AC:          17,
		Resistances: []string{"fire"},
	}
	opts := testOptsWithStats(ts)
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 || results[0].Skipped {
		t.Fatalf("expected 1 non-skipped result, got %d", len(results))
	}

	rawDmg := results[0].DamageResult.Total
	expectedDmg := rawDmg / 2 // resistance halves, round down
	expectedHP := 50 - expectedDmg
	if target.RuntimeState.CurrentHP != expectedHP {
		t.Errorf("expected target HP %d (resistance: %d raw → %d), got %d",
			expectedHP, rawDmg, expectedDmg, target.RuntimeState.CurrentHP)
	}
	if len(changes) != 1 || changes[0].HPDelta != -expectedDmg {
		t.Errorf("expected HPDelta %d, got %d", -expectedDmg, changes[0].HPDelta)
	}
}

func TestApplyTriggerResults_DealDamage_Vulnerability(t *testing.T) {
	target := makeTarget(100, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.0,
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "2d6",
				"damageType": "fire",
			},
		},
	}}

	ts := &TargetStats{
		Name:            "Ice Mephit",
		AC:              11,
		Vulnerabilities: []string{"fire"},
	}
	opts := testOptsWithStats(ts)
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	rawDmg := results[0].DamageResult.Total
	expectedDmg := rawDmg * 2 // vulnerability doubles
	expectedHP := 100 - expectedDmg
	if target.RuntimeState.CurrentHP != expectedHP {
		t.Errorf("expected target HP %d (vulnerability: %d raw → %d), got %d",
			expectedHP, rawDmg, expectedDmg, target.RuntimeState.CurrentHP)
	}
	if len(changes) != 1 || changes[0].HPDelta != -expectedDmg {
		t.Errorf("expected HPDelta %d, got %d", -expectedDmg, changes[0].HPDelta)
	}
}

func TestApplyTriggerResults_NilTarget_DealDamageSkipped(t *testing.T) {
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{flameTongueTrigger()} // deal_damage 2d6 fire

	// target is nil — deal_damage should be silently skipped (no crash)
	results, changes := applyTriggerResults(triggers, testOpts(), models.ItemTriggerOnHit, false, nil, attacker)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Result still fires (engine doesn't know about target), but no side effects applied
	if results[0].Skipped {
		t.Fatal("engine result should not be skipped (it fired successfully)")
	}
	// No state changes because target is nil
	if len(changes) != 0 {
		t.Errorf("expected 0 state changes (nil target), got %d", len(changes))
	}
	// Attacker HP unchanged
	if attacker.CharacterRuntime.CurrentHP != 30 {
		t.Errorf("attacker HP should be unchanged at 30, got %d", attacker.CharacterRuntime.CurrentHP)
	}
}

func TestApplyTriggerResults_NilTargetStats_NoCrash(t *testing.T) {
	target := makeTarget(50, 0)
	attacker := makeAttackerPC(30, 0)
	triggers := []models.TriggerEffect{flameTongueTrigger()}

	// No TargetStats — raw damage applied (no resistance check)
	opts := testOpts()
	opts.TargetStats = nil
	results, changes := applyTriggerResults(triggers, opts, models.ItemTriggerOnHit, false, target, attacker)

	if len(results) != 1 || results[0].Skipped {
		t.Fatalf("expected 1 non-skipped result, got %d", len(results))
	}

	rawDmg := results[0].DamageResult.Total
	expectedHP := 50 - rawDmg
	if target.RuntimeState.CurrentHP != expectedHP {
		t.Errorf("expected target HP %d (no resistance check), got %d", expectedHP, target.RuntimeState.CurrentHP)
	}
	if len(changes) != 1 || changes[0].HPDelta != -rawDmg {
		t.Errorf("expected raw HPDelta %d, got %d", -rawDmg, changes[0].HPDelta)
	}
}

// --- cooldown integration tests ---

func TestApplyTriggerResults_Cooldown_1PerTurn_FiresThenSkips(t *testing.T) {
	owner := makeOwner()
	target := makeTarget(100, 0)
	attacker := makeAttackerPC(30, 0)
	trigs := []models.TriggerEffect{{
		Trigger:  models.ItemTriggerOnHit,
		Chance:   1.0,
		Cooldown: "1/turn",
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "2d6",
				"damageType": "fire",
			},
		},
	}}

	opts := testOpts()
	opts.Owner = owner

	// First hit: trigger fires
	results, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target, attacker)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Skipped {
		t.Fatal("expected trigger to fire on first hit")
	}

	// Owner charges should be consumed
	if owner.TriggerCharges == nil || owner.TriggerCharges["weapon-1:0"] != 1 {
		t.Fatalf("expected charges[weapon-1:0]=1, got %v", owner.TriggerCharges)
	}

	// Second hit with same owner: trigger should be skipped (cooldown)
	target2 := makeTarget(100, 0)
	results2, changes2 := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target2, attacker)
	if len(results2) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results2))
	}
	if !results2[0].Skipped {
		t.Fatal("expected trigger to be skipped on second hit (cooldown)")
	}
	if results2[0].SkipReason != "cooldown" {
		t.Errorf("expected SkipReason 'cooldown', got %q", results2[0].SkipReason)
	}
	// No side effects
	if len(changes2) != 0 {
		t.Errorf("expected 0 state changes on cooldown, got %d", len(changes2))
	}
}

func TestApplyTriggerResults_Cooldown_MixedWithNoCooldown(t *testing.T) {
	owner := makeOwner()
	target := makeTarget(200, 0)
	attacker := makeAttackerPC(30, 0)
	trigs := []models.TriggerEffect{
		// Trigger 0: no cooldown (always fires)
		flameTongueTrigger(),
		// Trigger 1: 1/turn cooldown
		{
			Trigger:  models.ItemTriggerOnHit,
			Chance:   1.0,
			Cooldown: "1/turn",
			Effect: models.Effect{
				Type: models.EffectDealDamage,
				Params: map[string]interface{}{
					"dice":       "1d8",
					"damageType": "radiant",
				},
			},
		},
	}

	opts := testOpts()
	opts.Owner = owner

	// First hit: both fire
	results, changes := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target, attacker)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Skipped || results[1].Skipped {
		t.Fatal("both triggers should fire on first hit")
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 state changes, got %d", len(changes))
	}

	// Second hit: only non-cooldown fires
	target2 := makeTarget(200, 0)
	results2, changes2 := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target2, attacker)
	if len(results2) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results2))
	}
	if results2[0].Skipped {
		t.Fatal("non-cooldown trigger should still fire")
	}
	if !results2[1].Skipped {
		t.Fatal("cooldown trigger should be skipped on second hit")
	}
	if len(changes2) != 1 {
		t.Fatalf("expected 1 state change (only non-cooldown), got %d", len(changes2))
	}
}

func TestApplyTriggerResults_Cooldown_NilOwner_NoCooldowns(t *testing.T) {
	target := makeTarget(100, 0)
	attacker := makeAttackerPC(30, 0)
	trigs := []models.TriggerEffect{{
		Trigger:  models.ItemTriggerOnHit,
		Chance:   1.0,
		Cooldown: "1/turn",
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "2d6",
				"damageType": "fire",
			},
		},
	}}

	opts := testOpts()
	// Owner is nil — no cooldown tracking

	// First hit
	results, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target, attacker)
	if len(results) != 1 || results[0].Skipped {
		t.Fatal("expected trigger to fire without owner")
	}

	// Second hit — still fires (no cooldown tracking)
	target2 := makeTarget(100, 0)
	results2, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target2, attacker)
	if len(results2) != 1 || results2[0].Skipped {
		t.Fatal("expected trigger to fire again without owner (no cooldowns)")
	}
}

func TestApplyTriggerResults_Cooldown_3PerTurn(t *testing.T) {
	owner := makeOwner()
	attacker := makeAttackerPC(30, 0)
	trigs := []models.TriggerEffect{{
		Trigger:  models.ItemTriggerOnHit,
		Chance:   1.0,
		Cooldown: "3/turn",
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "1d6",
				"damageType": "fire",
			},
		},
	}}

	opts := testOpts()
	opts.Owner = owner

	// Hits 1-3: trigger fires each time
	for i := 1; i <= 3; i++ {
		target := makeTarget(100, 0)
		results, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target, attacker)
		if len(results) != 1 {
			t.Fatalf("hit %d: expected 1 result, got %d", i, len(results))
		}
		if results[0].Skipped {
			t.Fatalf("hit %d: expected trigger to fire", i)
		}
	}

	if owner.TriggerCharges["weapon-1:0"] != 3 {
		t.Fatalf("expected 3 charges consumed, got %d", owner.TriggerCharges["weapon-1:0"])
	}

	// Hit 4: trigger should be skipped
	target := makeTarget(100, 0)
	results, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target, attacker)
	if len(results) != 1 {
		t.Fatalf("hit 4: expected 1 result, got %d", len(results))
	}
	if !results[0].Skipped {
		t.Fatal("hit 4: expected trigger to be skipped (3/turn exhausted)")
	}
	if results[0].SkipReason != "cooldown" {
		t.Errorf("hit 4: expected SkipReason 'cooldown', got %q", results[0].SkipReason)
	}
}

func TestApplyTriggerResults_Cooldown_ResetRestoresFiring(t *testing.T) {
	owner := makeOwner()
	attacker := makeAttackerPC(30, 0)
	trigs := []models.TriggerEffect{{
		Trigger:  models.ItemTriggerOnHit,
		Chance:   1.0,
		Cooldown: "1/turn",
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "2d6",
				"damageType": "fire",
			},
		},
	}}

	opts := testOpts()
	opts.Owner = owner

	// First hit: fires and consumes charge
	target := makeTarget(100, 0)
	results, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target, attacker)
	if results[0].Skipped {
		t.Fatal("expected trigger to fire on first hit")
	}

	// Second hit: skipped (cooldown)
	target2 := makeTarget(100, 0)
	results2, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target2, attacker)
	if !results2[0].Skipped {
		t.Fatal("expected trigger to be skipped on second hit")
	}

	// Reset turn cooldowns
	owner.TriggerCharges = triggers.ResetCooldowns(
		owner.TriggerCharges, trigs, "weapon-1", triggers.PeriodTurn,
	)

	// Third hit after reset: fires again
	target3 := makeTarget(100, 0)
	results3, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target3, attacker)
	if len(results3) != 1 {
		t.Fatalf("expected 1 result after reset, got %d", len(results3))
	}
	if results3[0].Skipped {
		t.Fatal("expected trigger to fire again after turn reset")
	}
}

func TestApplyTriggerResults_Cooldown_CritBothEvents(t *testing.T) {
	owner := makeOwner()
	target := makeTarget(200, 0)
	attacker := makeAttackerPC(30, 0)
	trigs := []models.TriggerEffect{
		// idx 0: on_hit 1/turn
		{
			Trigger:  models.ItemTriggerOnHit,
			Chance:   1.0,
			Cooldown: "1/turn",
			Effect: models.Effect{
				Type: models.EffectDealDamage,
				Params: map[string]interface{}{
					"dice":       "2d6",
					"damageType": "fire",
				},
			},
		},
		// idx 1: on_critical 1/turn
		{
			Trigger:  models.ItemTriggerOnCritical,
			Chance:   1.0,
			Cooldown: "1/turn",
			Effect: models.Effect{
				Type: models.EffectDealDamage,
				Params: map[string]interface{}{
					"dice":       "3d6",
					"damageType": "radiant",
				},
			},
		},
	}

	opts := testOpts()
	opts.Owner = owner

	// First crit: both on_hit and on_critical fire
	results, changes := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, true, target, attacker)
	if len(results) != 2 {
		t.Fatalf("expected 2 results (on_hit + on_critical), got %d", len(results))
	}
	if results[0].Skipped || results[1].Skipped {
		t.Fatal("both triggers should fire on first crit")
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 state changes, got %d", len(changes))
	}
	if owner.TriggerCharges["weapon-1:0"] != 1 || owner.TriggerCharges["weapon-1:1"] != 1 {
		t.Fatalf("expected both charges consumed, got %v", owner.TriggerCharges)
	}

	// Second crit: both on cooldown
	target2 := makeTarget(200, 0)
	results2, changes2 := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, true, target2, attacker)
	if len(results2) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results2))
	}
	if !results2[0].Skipped || results2[0].SkipReason != "cooldown" {
		t.Fatal("on_hit should be skipped (cooldown)")
	}
	if !results2[1].Skipped || results2[1].SkipReason != "cooldown" {
		t.Fatal("on_critical should be skipped (cooldown)")
	}
	if len(changes2) != 0 {
		t.Errorf("expected 0 state changes, got %d", len(changes2))
	}
}

func TestApplyTriggerResults_Cooldown_NonCritThenCrit(t *testing.T) {
	owner := makeOwner()
	attacker := makeAttackerPC(30, 0)
	trigs := []models.TriggerEffect{
		// idx 0: on_hit 1/turn
		{
			Trigger:  models.ItemTriggerOnHit,
			Chance:   1.0,
			Cooldown: "1/turn",
			Effect: models.Effect{
				Type: models.EffectDealDamage,
				Params: map[string]interface{}{
					"dice":       "2d6",
					"damageType": "fire",
				},
			},
		},
		// idx 1: on_critical 1/turn
		{
			Trigger:  models.ItemTriggerOnCritical,
			Chance:   1.0,
			Cooldown: "1/turn",
			Effect: models.Effect{
				Type: models.EffectDealDamage,
				Params: map[string]interface{}{
					"dice":       "3d6",
					"damageType": "radiant",
				},
			},
		},
	}

	opts := testOpts()
	opts.Owner = owner

	// Non-crit: only on_hit fires
	target := makeTarget(200, 0)
	results, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target, attacker)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (on_hit only), got %d", len(results))
	}
	if results[0].Skipped {
		t.Fatal("on_hit should fire")
	}

	// Crit: on_hit exhausted, on_critical fires
	target2 := makeTarget(200, 0)
	results2, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, true, target2, attacker)
	if len(results2) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results2))
	}
	if !results2[0].Skipped || results2[0].SkipReason != "cooldown" {
		t.Fatal("on_hit should be skipped (already used this turn)")
	}
	if results2[1].Skipped {
		t.Fatal("on_critical should fire (independent cooldown key)")
	}
}

func TestApplyTriggerResults_Cooldown_ChanceFailDoesNotConsume(t *testing.T) {
	owner := makeOwner()
	target := makeTarget(100, 0)
	attacker := makeAttackerPC(30, 0)
	trigs := []models.TriggerEffect{{
		Trigger:  models.ItemTriggerOnHit,
		Chance:   0.5,
		Cooldown: "1/turn",
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "2d6",
				"damageType": "fire",
			},
		},
	}}

	opts := testOpts()
	opts.Owner = owner
	opts.RandFloat = alwaysFail // chance fails

	// Chance fails — trigger skipped, charge NOT consumed
	results, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target, attacker)
	if len(results) != 1 || !results[0].Skipped {
		t.Fatal("expected trigger to be skipped (chance)")
	}
	if results[0].SkipReason != "chance" {
		t.Errorf("expected SkipReason 'chance', got %q", results[0].SkipReason)
	}
	if owner.TriggerCharges != nil && owner.TriggerCharges["weapon-1:0"] != 0 {
		t.Fatalf("charge should NOT be consumed on chance failure, got %v", owner.TriggerCharges)
	}

	// Next hit with chance succeeding — trigger fires (charge was not consumed)
	opts.RandFloat = alwaysSucceed
	target2 := makeTarget(100, 0)
	results2, _ := applyTriggerResults(trigs, opts, models.ItemTriggerOnHit, false, target2, attacker)
	if len(results2) != 1 || results2[0].Skipped {
		t.Fatal("expected trigger to fire (charge was not consumed)")
	}
	if owner.TriggerCharges["weapon-1:0"] != 1 {
		t.Fatalf("expected charge consumed after successful fire, got %v", owner.TriggerCharges)
	}
}
