package usecases

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// --- resolveTargetIDs ---

func TestResolveTargetIDs_MultiTarget(t *testing.T) {
	cmd := &models.ActionCommand{
		TargetIDs: []string{"goblin-1", "goblin-2", "goblin-3"},
		TargetID:  "old-single", // should be ignored when TargetIDs is set
	}
	ids := resolveTargetIDs(cmd)
	if len(ids) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(ids))
	}
	if ids[0] != "goblin-1" || ids[1] != "goblin-2" || ids[2] != "goblin-3" {
		t.Errorf("unexpected targets: %v", ids)
	}
}

func TestResolveTargetIDs_LegacySingleTarget(t *testing.T) {
	cmd := &models.ActionCommand{TargetID: "ogre-1"}
	ids := resolveTargetIDs(cmd)
	if len(ids) != 1 || ids[0] != "ogre-1" {
		t.Errorf("expected [ogre-1], got %v", ids)
	}
}

func TestResolveTargetIDs_NoTargets(t *testing.T) {
	cmd := &models.ActionCommand{}
	ids := resolveTargetIDs(cmd)
	if ids != nil {
		t.Errorf("expected nil, got %v", ids)
	}
}

func TestResolveTargetIDs_DeduplicatePreservesOrder(t *testing.T) {
	cmd := &models.ActionCommand{
		TargetIDs: []string{"goblin-1", "goblin-2", "goblin-1", "goblin-3", "goblin-2"},
	}
	ids := resolveTargetIDs(cmd)
	if len(ids) != 3 {
		t.Fatalf("expected 3 unique targets, got %d: %v", len(ids), ids)
	}
	if ids[0] != "goblin-1" || ids[1] != "goblin-2" || ids[2] != "goblin-3" {
		t.Errorf("expected [goblin-1 goblin-2 goblin-3], got %v", ids)
	}
}

// --- casterLevel ---

func TestCasterLevel_SingleClass(t *testing.T) {
	char := &models.CharacterBase{
		Classes: []models.ClassEntry{{ClassName: "wizard", Level: 10}},
	}
	if got := casterLevel(char); got != 10 {
		t.Errorf("expected 10, got %d", got)
	}
}

func TestCasterLevel_Multiclass(t *testing.T) {
	char := &models.CharacterBase{
		Classes: []models.ClassEntry{
			{ClassName: "fighter", Level: 5},
			{ClassName: "wizard", Level: 3},
		},
	}
	if got := casterLevel(char); got != 8 {
		t.Errorf("expected 8, got %d", got)
	}
}

func TestCasterLevel_NoClasses(t *testing.T) {
	char := &models.CharacterBase{}
	if got := casterLevel(char); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

// --- resolveCantripDice ---

func firebolCantripDef() *models.SpellDefinition {
	return &models.SpellDefinition{
		Level: 0,
		CantripScaling: &models.CantripScaling{
			DamageDice: []models.CantripScalingTier{
				{MinLevel: 1, DiceCount: 1, DiceType: "d10"},
				{MinLevel: 5, DiceCount: 2, DiceType: "d10"},
				{MinLevel: 11, DiceCount: 3, DiceType: "d10"},
				{MinLevel: 17, DiceCount: 4, DiceType: "d10"},
			},
		},
	}
}

func TestResolveCantripDice_Level1(t *testing.T) {
	count, diceType := resolveCantripDice(firebolCantripDef(), 1)
	if count != 1 || diceType != "d10" {
		t.Errorf("expected 1d10 at level 1, got %dd%s", count, diceType)
	}
}

func TestResolveCantripDice_Level5(t *testing.T) {
	count, diceType := resolveCantripDice(firebolCantripDef(), 5)
	if count != 2 || diceType != "d10" {
		t.Errorf("expected 2d10 at level 5, got %dd%s", count, diceType)
	}
}

func TestResolveCantripDice_Level11(t *testing.T) {
	count, diceType := resolveCantripDice(firebolCantripDef(), 11)
	if count != 3 || diceType != "d10" {
		t.Errorf("expected 3d10 at level 11, got %dd%s", count, diceType)
	}
}

func TestResolveCantripDice_Level17(t *testing.T) {
	count, diceType := resolveCantripDice(firebolCantripDef(), 17)
	if count != 4 || diceType != "d10" {
		t.Errorf("expected 4d10 at level 17, got %dd%s", count, diceType)
	}
}

func TestResolveCantripDice_Level8_UsesLevel5Tier(t *testing.T) {
	count, diceType := resolveCantripDice(firebolCantripDef(), 8)
	if count != 2 || diceType != "d10" {
		t.Errorf("expected 2d10 at level 8 (tier 5 applies), got %dd%s", count, diceType)
	}
}

func TestResolveCantripDice_NoScaling(t *testing.T) {
	def := &models.SpellDefinition{Level: 0}
	count, diceType := resolveCantripDice(def, 10)
	if count != 0 || diceType != "" {
		t.Errorf("expected (0, \"\") without scaling, got (%d, %q)", count, diceType)
	}
}

// --- resolveUpcastDamage ---

func fireballDef() *models.SpellDefinition {
	return &models.SpellDefinition{
		Level: 3,
		Upcast: &models.UpcastData{
			Scaling: []models.UpcastScaling{
				{Level: 4, Damage: &models.DamageRoll{DiceCount: 1, DiceType: "d6", DamageType: "fire"}},
			},
		},
	}
}

func TestResolveUpcastDamage_NoUpcast(t *testing.T) {
	result := resolveUpcastDamage(fireballDef(), 3)
	if result != nil {
		t.Errorf("expected nil at base level, got %+v", result)
	}
}

func TestResolveUpcastDamage_Level4(t *testing.T) {
	result := resolveUpcastDamage(fireballDef(), 4)
	if result == nil {
		t.Fatal("expected upcast damage at level 4")
	}
	// 1 level above base → 1 * 1d6
	if result.DiceCount != 1 || result.DiceType != "d6" {
		t.Errorf("expected 1d6, got %dd%s", result.DiceCount, result.DiceType)
	}
}

func TestResolveUpcastDamage_Level5(t *testing.T) {
	result := resolveUpcastDamage(fireballDef(), 5)
	if result == nil {
		t.Fatal("expected upcast damage at level 5")
	}
	// 2 levels above base → 2 * 1d6
	if result.DiceCount != 2 || result.DiceType != "d6" {
		t.Errorf("expected 2d6, got %dd%s", result.DiceCount, result.DiceType)
	}
}

func TestResolveUpcastDamage_Level9(t *testing.T) {
	result := resolveUpcastDamage(fireballDef(), 9)
	if result == nil {
		t.Fatal("expected upcast damage at level 9")
	}
	// 6 levels above base → 6 * 1d6
	if result.DiceCount != 6 || result.DiceType != "d6" {
		t.Errorf("expected 6d6, got %dd%s", result.DiceCount, result.DiceType)
	}
}

func TestResolveUpcastDamage_NilUpcast(t *testing.T) {
	def := &models.SpellDefinition{Level: 1}
	result := resolveUpcastDamage(def, 3)
	if result != nil {
		t.Errorf("expected nil without upcast data, got %+v", result)
	}
}

// --- resolveUpcastHealing ---

func cureWoundsDef() *models.SpellDefinition {
	return &models.SpellDefinition{
		Level: 1,
		Upcast: &models.UpcastData{
			Scaling: []models.UpcastScaling{
				{Level: 2, HealingAdd: 1}, // +1d8 per level above 1st
			},
		},
	}
}

func healDef() *models.SpellDefinition {
	return &models.SpellDefinition{
		Level: 6,
		Upcast: &models.UpcastData{
			Scaling: []models.UpcastScaling{
				{Level: 7, HealingAddFlat: 10}, // +10 HP per level above 6th
			},
		},
	}
}

func TestResolveUpcastHealing_NilUpcast(t *testing.T) {
	def := &models.SpellDefinition{Level: 1}
	dice, flat := resolveUpcastHealing(def, 3)
	if dice != 0 || flat != 0 {
		t.Errorf("expected (0, 0), got (%d, %d)", dice, flat)
	}
}

func TestResolveUpcastHealing_AtBaseLevel(t *testing.T) {
	dice, flat := resolveUpcastHealing(cureWoundsDef(), 1)
	if dice != 0 || flat != 0 {
		t.Errorf("expected (0, 0) at base level, got (%d, %d)", dice, flat)
	}
}

func TestResolveUpcastHealing_CureWoundsLevel3(t *testing.T) {
	dice, flat := resolveUpcastHealing(cureWoundsDef(), 3)
	// 2 levels above base → 2 extra dice
	if dice != 2 || flat != 0 {
		t.Errorf("expected (2, 0), got (%d, %d)", dice, flat)
	}
}

func TestResolveUpcastHealing_HealLevel8(t *testing.T) {
	dice, flat := resolveUpcastHealing(healDef(), 8)
	// 2 levels above base → +20 flat HP
	if dice != 0 || flat != 20 {
		t.Errorf("expected (0, 20), got (%d, %d)", dice, flat)
	}
}

// --- buildActiveCondition ---

func TestBuildActiveCondition_UntilSaved(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition:   models.ConditionStunned,
		Duration:    "until saved",
		SaveEnds:    true,
		SaveAbility: "CON",
	}
	ac := buildActiveCondition(cond, "wizard-1", "Power Word Stun", 17, "target-1")
	if ac.Duration != models.DurationUntilSave {
		t.Errorf("expected DurationUntilSave, got %q", ac.Duration)
	}
	if ac.SaveToEnd == nil {
		t.Fatal("expected SaveToEnd to be set")
	}
	if ac.SaveToEnd.DC != 17 {
		t.Errorf("expected DC 17, got %d", ac.SaveToEnd.DC)
	}
	if ac.SaveToEnd.Ability != "CON" {
		t.Errorf("expected ability CON, got %q", ac.SaveToEnd.Ability)
	}
	if ac.SourceID != "wizard-1" {
		t.Errorf("expected sourceID wizard-1, got %q", ac.SourceID)
	}
}

func TestBuildActiveCondition_1Minute(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition: models.ConditionFrightened,
		Duration:  "1 minute",
	}
	ac := buildActiveCondition(cond, "bard-1", "Fear", 15, "target-1")
	if ac.Duration != models.DurationRounds {
		t.Errorf("expected DurationRounds, got %q", ac.Duration)
	}
	if ac.RoundsLeft != 10 {
		t.Errorf("expected 10 rounds, got %d", ac.RoundsLeft)
	}
}

func TestBuildActiveCondition_Concentration(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition: models.ConditionCharmed,
		Duration:  "concentration, up to 1 minute",
	}
	ac := buildActiveCondition(cond, "sorcerer-1", "Hold Person", 14, "target-1")
	if ac.Duration != models.DurationConcentration {
		t.Errorf("expected DurationConcentration, got %q", ac.Duration)
	}
}

func TestBuildActiveCondition_EndOfNextTurn(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition: models.ConditionBlinded,
		Duration:  "until end of next turn",
	}
	ac := buildActiveCondition(cond, "cleric-1", "Color Spray", 0, "target-1")
	if ac.Duration != models.DurationUntilTurn {
		t.Errorf("expected DurationUntilTurn, got %q", ac.Duration)
	}
	if ac.EndsOnTurn != "end" {
		t.Errorf("expected endsOnTurn 'end', got %q", ac.EndsOnTurn)
	}
}

func TestBuildActiveCondition_WithEscapeDC(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition:   models.ConditionRestrained,
		Duration:    "until saved",
		SaveEnds:    true,
		SaveAbility: "STR",
		EscapeDC:    15,
		EscapeType:  "STR_or_DEX",
	}
	ac := buildActiveCondition(cond, "druid-1", "Entangle", 13, "target-1")
	if ac.EscapeDC != 15 {
		t.Errorf("expected EscapeDC 15, got %d", ac.EscapeDC)
	}
	if ac.EscapeType != "STR_or_DEX" {
		t.Errorf("expected EscapeType STR_or_DEX, got %q", ac.EscapeType)
	}
	// SaveToEnd should use EscapeDC over spell DC
	if ac.SaveToEnd == nil || ac.SaveToEnd.DC != 15 {
		t.Errorf("expected SaveToEnd DC = EscapeDC 15, got %+v", ac.SaveToEnd)
	}
}

func TestBuildActiveCondition_UniqueIDsPerTarget(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition: models.ConditionStunned,
		Duration:  "1 minute",
	}
	ac1 := buildActiveCondition(cond, "wizard-1", "Hold Monster", 17, "goblin-1")
	ac2 := buildActiveCondition(cond, "wizard-1", "Hold Monster", 17, "goblin-2")
	if ac1.ID == ac2.ID {
		t.Errorf("condition IDs should be unique per target, both got %q", ac1.ID)
	}
}

// --- parseDurationRounds ---

func TestParseDurationRounds_Rounds(t *testing.T) {
	if n := parseDurationRounds("3 rounds"); n != 3 {
		t.Errorf("expected 3, got %d", n)
	}
}

func TestParseDurationRounds_Minutes(t *testing.T) {
	if n := parseDurationRounds("10 minutes"); n != 100 {
		t.Errorf("expected 100, got %d", n)
	}
}

func TestParseDurationRounds_1Hour(t *testing.T) {
	if n := parseDurationRounds("1 hour"); n != 600 {
		t.Errorf("expected 600, got %d", n)
	}
}

func TestParseDurationRounds_Unparseable(t *testing.T) {
	if n := parseDurationRounds("until dispelled"); n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

// --- sumDamageRolls ---

func TestSumDamageRolls_MixedFinalDamage(t *testing.T) {
	rolls := []models.ActionRollResult{
		{Total: 10, FinalDamage: intPtr(5)}, // resistance halved
		{Total: 8},                          // no FinalDamage → use Total
		{Total: 12, FinalDamage: intPtr(0)}, // immunity
	}
	if got := sumDamageRolls(rolls); got != 13 { // 5 + 8 + 0
		t.Errorf("expected 13, got %d", got)
	}
}

func TestSumDamageRolls_Empty(t *testing.T) {
	if got := sumDamageRolls(nil); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

// --- resolveConditionEffect ---

func TestResolveConditionEffect_AutoApplies(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition: models.ConditionStunned,
		Duration:  "until saved",
		SaveEnds:  true,
	}
	spell := &models.SpellDefinition{
		Resolution: models.SpellResolution{Type: "auto"},
	}
	resp := &models.ActionResponse{}
	resolveConditionEffect(cond, spell, nil, []string{"target-1"}, resp)
	if len(resp.ConditionApplied) != 1 {
		t.Fatalf("expected 1 condition applied, got %d", len(resp.ConditionApplied))
	}
	if resp.ConditionApplied[0].Condition != "stunned" {
		t.Errorf("expected 'stunned', got %q", resp.ConditionApplied[0].Condition)
	}
}

func TestResolveConditionEffect_SaveFailed(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition: models.ConditionParalyzed,
		Duration:  "1 minute",
		SaveEnds:  true,
	}
	spell := &models.SpellDefinition{
		Resolution: models.SpellResolution{Type: "save"},
	}
	saveRes := &spellSaveResult{saved: false}
	resp := &models.ActionResponse{}
	resolveConditionEffect(cond, spell, saveRes, []string{"orc-1", "orc-2"}, resp)
	if len(resp.ConditionApplied) != 2 {
		t.Fatalf("expected 2 conditions (multi-target), got %d", len(resp.ConditionApplied))
	}
}

func TestResolveConditionEffect_SaveSucceeded_NoCondition(t *testing.T) {
	cond := &models.ConditionEffect{
		Condition: models.ConditionParalyzed,
		Duration:  "1 minute",
	}
	spell := &models.SpellDefinition{
		Resolution: models.SpellResolution{Type: "save"},
	}
	saveRes := &spellSaveResult{saved: true, noDamage: true}
	resp := &models.ActionResponse{}
	resolveConditionEffect(cond, spell, saveRes, []string{"orc-1"}, resp)
	if len(resp.ConditionApplied) != 0 {
		t.Errorf("expected no conditions on successful save, got %d", len(resp.ConditionApplied))
	}
}

// --- appendConditionToTarget ---

func TestAppendConditionToTarget_Creature(t *testing.T) {
	target := &models.ParticipantFull{
		InstanceID: "goblin-1",
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: 20,
			MaxHP:     20,
		},
	}
	ac := models.ActiveCondition{
		ID:        "spell_hold_paralyzed",
		Condition: models.ConditionParalyzed,
		Duration:  models.DurationUntilSave,
	}
	appendConditionToTarget(target, ac)
	if len(target.RuntimeState.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(target.RuntimeState.Conditions))
	}
	if target.RuntimeState.Conditions[0].Condition != models.ConditionParalyzed {
		t.Errorf("expected paralyzed, got %q", target.RuntimeState.Conditions[0].Condition)
	}
}

func TestAppendConditionToTarget_PC(t *testing.T) {
	target := &models.ParticipantFull{
		InstanceID:        "player-1",
		IsPlayerCharacter: true,
		CharacterRuntime: &models.CharacterRuntime{
			CharacterID: "char-abc",
			CurrentHP:   30,
		},
	}
	ac := models.ActiveCondition{
		ID:         "spell_fear_frightened",
		Condition:  models.ConditionFrightened,
		Duration:   models.DurationRounds,
		RoundsLeft: 10,
		SourceID:   "caster-1",
		SaveToEnd: &models.SaveToEndCondition{
			Ability: "WIS",
			DC:      15,
			Timing:  "end_of_turn",
		},
	}
	appendConditionToTarget(target, ac)
	if len(target.CharacterRuntime.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(target.CharacterRuntime.Conditions))
	}
	ci := target.CharacterRuntime.Conditions[0]
	if ci.Type != models.ConditionFrightened {
		t.Errorf("expected frightened, got %q", ci.Type)
	}
	if ci.Duration.Remaining != 10 {
		t.Errorf("expected 10 rounds remaining, got %d", ci.Duration.Remaining)
	}
	if ci.SaveRetry == nil || ci.SaveRetry.DC != 15 {
		t.Errorf("expected SaveRetry DC 15, got %+v", ci.SaveRetry)
	}
	if ci.SourceCreatureID != "caster-1" {
		t.Errorf("expected sourceCreatureID caster-1, got %q", ci.SourceCreatureID)
	}
}
