package triggers

import (
	"strings"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func alwaysSucceed() float32 { return 0.0 }
func alwaysFail() float32    { return 1.0 }

func flameTongue() models.TriggerEffect {
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

func TestEvaluate_DealDamage(t *testing.T) {
	results := Evaluate(
		[]TriggerInput{{Trigger: flameTongue(), CooldownKey: "ft:0"}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Flame Tongue"},
		nil,
		alwaysSucceed,
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Skipped {
		t.Fatal("expected result not to be skipped")
	}
	if r.Event != models.ItemTriggerOnHit {
		t.Errorf("expected on_hit, got %q", r.Event)
	}
	if r.EffectType != models.EffectDealDamage {
		t.Errorf("expected deal_damage, got %q", r.EffectType)
	}
	if r.DamageResult == nil {
		t.Fatal("expected DamageResult to be set")
	}
	if r.DamageResult.DamageType != "fire" {
		t.Errorf("expected fire, got %q", r.DamageResult.DamageType)
	}
	if r.DamageResult.Total < 2 || r.DamageResult.Total > 12 {
		t.Errorf("2d6 total out of range: %d", r.DamageResult.Total)
	}
	if len(r.DamageResult.Rolls) != 2 {
		t.Errorf("expected 2 rolls, got %d", len(r.DamageResult.Rolls))
	}
}

func TestEvaluate_EventFiltering(t *testing.T) {
	results := Evaluate(
		[]TriggerInput{{Trigger: flameTongue(), CooldownKey: "ft:0"}},
		EventContext{Event: models.ItemTriggerOnTurnStart, SourceName: "Flame Tongue"},
		nil,
		alwaysSucceed,
	)
	if len(results) != 0 {
		t.Fatalf("expected 0 results for non-matching event, got %d", len(results))
	}
}

func TestEvaluate_ChanceSuccess(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Chance:  0.5,
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "1d6",
				"damageType": "necrotic",
			},
		},
	}

	// randFloat returns 0.3, which is < 0.5 → fires
	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Sword of Wounding"},
		nil,
		func() float32 { return 0.3 },
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Skipped {
		t.Fatal("expected result not to be skipped (chance 0.3 < 0.5)")
	}
}

func TestEvaluate_ChanceFailure(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Chance:  0.5,
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "1d6",
				"damageType": "necrotic",
			},
		},
	}

	// randFloat returns 0.7, which is >= 0.5 → skipped
	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Sword of Wounding"},
		nil,
		func() float32 { return 0.7 },
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Skipped {
		t.Fatal("expected result to be skipped (chance 0.7 >= 0.5)")
	}
	if results[0].SkipReason != "chance" {
		t.Errorf("expected SkipReason 'chance', got %q", results[0].SkipReason)
	}
}

func TestEvaluate_ChanceZeroAlwaysFires(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Chance:  0, // 0 means always fires
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "1d6",
				"damageType": "fire",
			},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Test"},
		nil,
		alwaysFail, // would fail if chance were checked
	)
	if len(results) != 1 || results[0].Skipped {
		t.Fatal("chance=0 should always fire")
	}
}

func TestEvaluate_ChanceClampedAboveOne(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Chance:  1.5, // invalid, clamped to 1.0 → always fires
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "1d6",
				"damageType": "fire",
			},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Test"},
		nil,
		alwaysFail,
	)
	if len(results) != 1 || results[0].Skipped {
		t.Fatal("chance>1.0 should be clamped to 1.0 and always fire")
	}
}

func TestEvaluate_ChanceClampedNegative(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Chance:  -0.5, // invalid, clamped to 0 → treated as always fires
		Effect: models.Effect{
			Type: models.EffectDealDamage,
			Params: map[string]interface{}{
				"dice":       "1d6",
				"damageType": "fire",
			},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Test"},
		nil,
		alwaysFail,
	)
	if len(results) != 1 || results[0].Skipped {
		t.Fatal("negative chance should be clamped to 0 (always fires)")
	}
}

func TestEvaluate_CooldownSkip(t *testing.T) {
	cooldowns := models.CooldownState{"staff:0": true}

	results := Evaluate(
		[]TriggerInput{{Trigger: flameTongue(), CooldownKey: "staff:0"}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Staff"},
		cooldowns,
		alwaysSucceed,
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Skipped {
		t.Fatal("expected result to be skipped due to cooldown")
	}
	if results[0].SkipReason != "cooldown" {
		t.Errorf("expected SkipReason 'cooldown', got %q", results[0].SkipReason)
	}
}

func TestEvaluate_NilCooldowns(t *testing.T) {
	results := Evaluate(
		[]TriggerInput{{Trigger: flameTongue(), CooldownKey: "ft:0"}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Flame Tongue"},
		nil,
		alwaysSucceed,
	)
	if len(results) != 1 || results[0].Skipped {
		t.Fatal("nil cooldowns should not skip anything")
	}
}

func TestEvaluate_Heal(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnUse,
		Chance:  1.0,
		Effect: models.Effect{
			Type:   models.EffectHeal,
			Params: map[string]interface{}{"dice": "1d8+4"},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnUse, SourceName: "Staff of Healing"},
		nil,
		alwaysSucceed,
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.HealResult == nil {
		t.Fatal("expected HealResult to be set")
	}
	if r.HealResult.Total < 5 || r.HealResult.Total > 12 {
		t.Errorf("1d8+4 total out of range: %d", r.HealResult.Total)
	}
}

func TestEvaluate_HealFlatAmount(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnTurnStart,
		Effect: models.Effect{
			Type:   models.EffectHeal,
			Params: map[string]interface{}{"amount": float64(1)},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnTurnStart, SourceName: "Ring of Regeneration"},
		nil,
		alwaysSucceed,
	)
	r := results[0]
	if r.HealResult == nil || r.HealResult.Total != 1 {
		t.Fatalf("expected flat heal of 1, got %+v", r.HealResult)
	}
}

func TestEvaluate_ApplyCondition(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Chance:  0.5,
		Effect: models.Effect{
			Type: models.EffectApplyCondition,
			Params: map[string]interface{}{
				"condition": "frightened",
				"duration":  "1 minute",
			},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Mace of Terror"},
		nil,
		alwaysSucceed,
	)
	r := results[0]
	if r.ConditionResult == nil {
		t.Fatal("expected ConditionResult to be set")
	}
	if r.ConditionResult.Action != "apply" {
		t.Errorf("expected action 'apply', got %q", r.ConditionResult.Action)
	}
	if r.ConditionResult.Condition != "frightened" {
		t.Errorf("expected condition 'frightened', got %q", r.ConditionResult.Condition)
	}
	if r.ConditionResult.Duration != "1 minute" {
		t.Errorf("expected duration '1 minute', got %q", r.ConditionResult.Duration)
	}
}

func TestEvaluate_RemoveCondition(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnUse,
		Effect: models.Effect{
			Type: models.EffectRemoveCondition,
			Params: map[string]interface{}{
				"conditions": []interface{}{"poisoned", "diseased"},
			},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnUse, SourceName: "Antitoxin"},
		nil,
		alwaysSucceed,
	)
	r := results[0]
	if r.ConditionResult == nil {
		t.Fatal("expected ConditionResult to be set")
	}
	if r.ConditionResult.Action != "remove" {
		t.Errorf("expected action 'remove', got %q", r.ConditionResult.Action)
	}
	if len(r.ConditionResult.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(r.ConditionResult.Conditions))
	}
}

func TestEvaluate_GrantTempHP_Dice(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnEquip,
		Effect: models.Effect{
			Type:   models.EffectGrantTempHP,
			Params: map[string]interface{}{"dice": "1d6"},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnEquip, SourceName: "Amulet of Health"},
		nil,
		alwaysSucceed,
	)
	r := results[0]
	if r.TempHPResult == nil {
		t.Fatal("expected TempHPResult to be set")
	}
	if r.TempHPResult.Amount < 1 || r.TempHPResult.Amount > 6 {
		t.Errorf("1d6 temp HP out of range: %d", r.TempHPResult.Amount)
	}
	if r.TempHPResult.Dice != "1d6" {
		t.Errorf("expected dice '1d6', got %q", r.TempHPResult.Dice)
	}
}

func TestEvaluate_GrantTempHP_FlatAmount(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnEquip,
		Effect: models.Effect{
			Type:   models.EffectGrantTempHP,
			Params: map[string]interface{}{"amount": float64(10)},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnEquip, SourceName: "Armor of Resistance"},
		nil,
		alwaysSucceed,
	)
	r := results[0]
	if r.TempHPResult == nil || r.TempHPResult.Amount != 10 {
		t.Fatalf("expected flat temp HP of 10, got %+v", r.TempHPResult)
	}
}

func TestEvaluate_UnknownEffectType_SkippedWithError(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Effect: models.Effect{
			Type:   "teleport",
			Params: map[string]interface{}{},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Test"},
		nil,
		alwaysSucceed,
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Skipped || results[0].SkipReason != "error" {
		t.Fatalf("expected skipped with reason 'error', got skipped=%v reason=%q",
			results[0].Skipped, results[0].SkipReason)
	}
}

func TestEvaluate_MissingRequiredParam_SkippedWithError(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Effect: models.Effect{
			Type:   models.EffectDealDamage,
			Params: map[string]interface{}{"dice": "2d6"}, // missing damageType
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Test"},
		nil,
		alwaysSucceed,
	)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Skipped || results[0].SkipReason != "error" {
		t.Fatalf("expected skipped with reason 'error', got skipped=%v reason=%q",
			results[0].Skipped, results[0].SkipReason)
	}
}

func TestEvaluate_ErrorDoesNotAbortSiblings(t *testing.T) {
	good := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Effect: models.Effect{
			Type:   models.EffectDealDamage,
			Params: map[string]interface{}{"dice": "1d6", "damageType": "fire"},
		},
	}
	bad := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Effect: models.Effect{
			Type:   "nonexistent",
			Params: map[string]interface{}{},
		},
	}
	good2 := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Effect: models.Effect{
			Type:   models.EffectApplyCondition,
			Params: map[string]interface{}{"condition": "poisoned"},
		},
	}

	results := Evaluate(
		[]TriggerInput{
			{Trigger: good, CooldownKey: "a:0"},
			{Trigger: bad, CooldownKey: "a:1"},
			{Trigger: good2, CooldownKey: "a:2"},
		},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Cursed Blade"},
		nil,
		alwaysSucceed,
	)
	if len(results) != 3 {
		t.Fatalf("expected 3 results (including error), got %d", len(results))
	}
	// First: valid deal_damage
	if results[0].Skipped || results[0].EffectType != models.EffectDealDamage {
		t.Errorf("first result should be valid deal_damage, got skipped=%v type=%q",
			results[0].Skipped, results[0].EffectType)
	}
	// Second: error skip
	if !results[1].Skipped || results[1].SkipReason != "error" {
		t.Errorf("second result should be error skip, got skipped=%v reason=%q",
			results[1].Skipped, results[1].SkipReason)
	}
	// Third: valid apply_condition (not aborted by sibling error)
	if results[2].Skipped || results[2].EffectType != models.EffectApplyCondition {
		t.Errorf("third result should be valid apply_condition, got skipped=%v type=%q",
			results[2].Skipped, results[2].EffectType)
	}
}

func TestEvaluate_MultipleTriggersSameEvent(t *testing.T) {
	t1 := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Effect: models.Effect{
			Type:   models.EffectDealDamage,
			Params: map[string]interface{}{"dice": "1d6", "damageType": "fire"},
		},
	}
	t2 := models.TriggerEffect{
		Trigger: models.ItemTriggerOnHit,
		Effect: models.Effect{
			Type:   models.EffectApplyCondition,
			Params: map[string]interface{}{"condition": "poisoned"},
		},
	}
	t3 := models.TriggerEffect{
		Trigger: models.ItemTriggerOnTurnStart, // different event — should not fire
		Effect: models.Effect{
			Type:   models.EffectHeal,
			Params: map[string]interface{}{"amount": float64(5)},
		},
	}

	results := Evaluate(
		[]TriggerInput{
			{Trigger: t1, CooldownKey: "a:0"},
			{Trigger: t2, CooldownKey: "a:1"},
			{Trigger: t3, CooldownKey: "a:2"},
		},
		EventContext{Event: models.ItemTriggerOnHit, SourceName: "Venomous Flame Blade"},
		nil,
		alwaysSucceed,
	)
	if len(results) != 2 {
		t.Fatalf("expected 2 results (only on_hit triggers), got %d", len(results))
	}
	if results[0].EffectType != models.EffectDealDamage {
		t.Errorf("first result: expected deal_damage, got %q", results[0].EffectType)
	}
	if results[1].EffectType != models.EffectApplyCondition {
		t.Errorf("second result: expected apply_condition, got %q", results[1].EffectType)
	}
}

func TestEvaluate_RemoveConditionDescription(t *testing.T) {
	trigger := models.TriggerEffect{
		Trigger: models.ItemTriggerOnUse,
		Effect: models.Effect{
			Type: models.EffectRemoveCondition,
			Params: map[string]interface{}{
				"conditions": []interface{}{"poisoned", "diseased"},
			},
		},
	}

	results := Evaluate(
		[]TriggerInput{{Trigger: trigger}},
		EventContext{Event: models.ItemTriggerOnUse, SourceName: "Antitoxin"},
		nil,
		alwaysSucceed,
	)
	r := results[0]
	if !strings.Contains(r.Description, "poisoned, diseased") {
		t.Errorf("expected comma-separated conditions in description, got %q", r.Description)
	}
}
