package combatai

import (
	"math"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// makeFireBreathCreature returns a creature with a fire breath weapon and a melee bite.
func makeFireBreathCreature() models.Creature {
	return models.Creature{
		Movement: models.CreatureMovement{Walk: 30},
		StructuredActions: []models.StructuredAction{
			{
				ID:       "bite",
				Name:     "Bite",
				Category: models.ActionCategoryAction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 7,
					Reach: 10,
					Damage: []models.DamageRoll{
						{DiceCount: 2, DiceType: "d10", Bonus: 4, DamageType: "piercing"},
					},
				},
			},
			{
				ID:       "fire_breath",
				Name:     "Fire Breath",
				Category: models.ActionCategoryAction,
				SavingThrow: &models.SavingThrowData{
					Ability:   models.AbilityDEX,
					DC:        17,
					OnSuccess: "half damage",
					Damage: []models.DamageRoll{
						{DiceCount: 12, DiceType: "d6", Bonus: 0, DamageType: "fire"},
					},
				},
			},
		},
	}
}

// makeSlashingCreature returns a creature with only a melee slashing attack.
func makeSlashingCreature() models.Creature {
	return models.Creature{
		Movement: models.CreatureMovement{Walk: 30},
		StructuredActions: []models.StructuredAction{
			{
				ID:       "longsword",
				Name:     "Longsword",
				Category: models.ActionCategoryAction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 5,
					Reach: 5,
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"},
					},
				},
			},
		},
	}
}

func TestAssessSingleThreat_ConcentrationBonus(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)
	enemy.RuntimeState.Concentration = &models.ConcentrationState{EffectName: "Bless"}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	assert.True(t, score.IsConcentrating)
	// Base 10 + 30 (concentration) = 40
	assert.InDelta(t, 40.0, score.Score, 0.01)
}

func TestAssessSingleThreat_LowHPBonus(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 10, 1, 0) // 10/100 = 10% HP

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 100, AC: 15}},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	assert.InDelta(t, 0.10, score.HPPercent, 0.01)
	// Base 10 + 25 (< 25% HP) = 35
	assert.InDelta(t, 35.0, score.Score, 0.01)
}

func TestAssessSingleThreat_MediumHPBonus(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 40, 1, 0) // 40/100 = 40% HP

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 100, AC: 15}},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 + 10 (< 50% HP) = 20
	assert.InDelta(t, 20.0, score.Score, 0.01)
}

func TestAssessSingleThreat_DistancePenalty(t *testing.T) {
	t.Parallel()

	// NPC at (0,0), enemy at (20,0) = 100ft. Movement 30 + reach 5 = 35ft. 100 > 35 → penalty.
	enemy := makeParticipant("pc1", true, 50, 20, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(), // Walk 30, reach 5
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 - 15 (distance) = -5
	assert.InDelta(t, -5.0, score.Score, 0.01)
	assert.Equal(t, 100, score.Distance)
}

func TestAssessSingleThreat_NoDistancePenalty_InRange(t *testing.T) {
	t.Parallel()

	// NPC at (0,0), enemy at (1,0) = 5ft. Within movement+reach.
	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10, no penalty
	assert.InDelta(t, 10.0, score.Score, 0.01)
}

func TestAssessSingleThreat_ImmunityPenalty(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeFireBreathCreature(),
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, Immunities: []string{"fire"}},
		},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 - 50 (immune to fire) = -40
	assert.InDelta(t, -40.0, score.Score, 0.01)
}

func TestAssessSingleThreat_ResistancePenalty(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeFireBreathCreature(),
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, Resistances: []string{"fire"}},
		},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 - 20 (resistant to fire) = -10
	assert.InDelta(t, -10.0, score.Score, 0.01)
}

func TestAssessSingleThreat_VulnerabilityBonus(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeFireBreathCreature(),
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, Vulnerabilities: []string{"fire"}},
		},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 + 20 (vulnerable to fire) = 30
	assert.InDelta(t, 30.0, score.Score, 0.01)
}

func TestAssessSingleThreat_CombinedConcentrationAndLowHP(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 5, 1, 0) // 5/100 = 5%
	enemy.RuntimeState.Concentration = &models.ConcentrationState{EffectName: "Hold Person"}

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 100, AC: 15}},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 + 30 (concentration) + 25 (< 25% HP) = 65
	assert.InDelta(t, 65.0, score.Score, 0.01)
}

func TestAssessThreats_Ranking(t *testing.T) {
	t.Parallel()

	// pc1: concentrating, full HP → high priority
	pc1 := makeParticipant("pc1", true, 50, 1, 0)
	pc1.RuntimeState.Concentration = &models.ConcentrationState{EffectName: "Bless"}

	// pc2: no concentration, full HP → low priority
	pc2 := makeParticipant("pc2", true, 50, 2, 0)

	// pc3: no concentration, very low HP → medium-high priority (finish off)
	pc3 := makeParticipant("pc3", true, 10, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		Participants: []models.ParticipantFull{
			makeParticipant("npc1", false, 50, 0, 0),
			pc1, pc2, pc3,
		},
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15},
			"pc2": {MaxHP: 50, AC: 15},
			"pc3": {MaxHP: 100, AC: 15},
		},
	}

	scores := AssessThreats(input)
	assert.Len(t, scores, 3)

	// Find scores by ID.
	scoreMap := make(map[string]float64)
	for _, s := range scores {
		scoreMap[s.TargetID] = s.Score
	}

	// pc1 (concentration +30) > pc3 (low HP +25) > pc2 (no bonuses)
	assert.Greater(t, scoreMap["pc1"], scoreMap["pc3"])
	assert.Greater(t, scoreMap["pc3"], scoreMap["pc2"])
}

func TestMainDamageType_MeleeAttack(t *testing.T) {
	t.Parallel()

	creature := makeSlashingCreature()
	assert.Equal(t, "slashing", mainDamageType(&creature))
}

func TestMainDamageType_SavingThrow(t *testing.T) {
	t.Parallel()

	creature := makeFireBreathCreature()
	// Fire Breath (12d6 fire, avg=42) > Bite (2d10+4 piercing, avg=15)
	assert.Equal(t, "fire", mainDamageType(&creature))
}

func TestMainDamageType_NoActions(t *testing.T) {
	t.Parallel()

	creature := models.Creature{}
	assert.Equal(t, "", mainDamageType(&creature))
}

func TestMaxMeleeReach_Default(t *testing.T) {
	t.Parallel()

	creature := models.Creature{} // no actions
	assert.Equal(t, 5, maxMeleeReach(creature))
}

func TestMaxMeleeReach_HasReach10(t *testing.T) {
	t.Parallel()

	creature := models.Creature{
		StructuredActions: []models.StructuredAction{
			{
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Reach: 10,
				},
			},
			{
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Reach: 5,
				},
			},
		},
	}
	assert.Equal(t, 10, maxMeleeReach(creature))
}

func TestMaxMeleeReach_OnlyRanged(t *testing.T) {
	t.Parallel()

	creature := models.Creature{
		StructuredActions: []models.StructuredAction{
			{
				Attack: &models.AttackRollData{
					Type:  models.AttackRollRangedWeapon,
					Range: &models.RangeData{Normal: 80},
				},
			},
		},
	}
	assert.Equal(t, 5, maxMeleeReach(creature)) // no melee → default 5
}

func TestContainsDamageType_CaseInsensitive(t *testing.T) {
	t.Parallel()

	assert.True(t, containsDamageType([]string{"Fire", "Cold"}, "fire"))
	assert.True(t, containsDamageType([]string{"fire"}, "FIRE"))
	assert.False(t, containsDamageType([]string{"fire"}, "lightning"))
	assert.False(t, containsDamageType(nil, "fire"))
}

func TestAssessSingleThreat_NilCoords(t *testing.T) {
	t.Parallel()

	enemy := models.ParticipantFull{
		InstanceID:        "pc1",
		IsPlayerCharacter: true,
		CellsCoords:       nil,
		RuntimeState:      models.CreatureRuntimeState{CurrentHP: 50, MaxHP: 50},
	}

	npc := makeParticipant("npc1", false, 50, 0, 0)
	input := &TurnInput{
		ActiveNPC:        npc,
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Distance = MaxInt32 (nil coords), which is > movement+reach → penalty applied.
	assert.Equal(t, math.MaxInt32, score.Distance)
	// Base 10 - 15 (distance) = -5
	assert.InDelta(t, -5.0, score.Score, 0.01)
}

// --- Focus-fire tests ---

func TestAssessSingleThreat_FocusFire_OneAlly(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		RecentNPCTargets: map[string]string{"npc2": "pc1"},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 + 15 (1 ally) = 25
	assert.InDelta(t, 25.0, score.Score, 0.01)
	assert.Equal(t, 1, score.AllyCount)
}

func TestAssessSingleThreat_FocusFire_TwoAllies(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		RecentNPCTargets: map[string]string{"npc2": "pc1", "npc3": "pc1"},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 + 30 (2 allies) = 40
	assert.InDelta(t, 40.0, score.Score, 0.01)
	assert.Equal(t, 2, score.AllyCount)
}

func TestAssessSingleThreat_FocusFire_CappedAtThree(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		RecentNPCTargets: map[string]string{
			"npc2": "pc1", "npc3": "pc1", "npc4": "pc1", "npc5": "pc1", "npc6": "pc1",
		},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 + 45 (capped at 3 allies × 15) = 55
	assert.InDelta(t, 55.0, score.Score, 0.01)
	assert.Equal(t, 3, score.AllyCount) // capped
}

func TestAssessSingleThreat_FocusFire_NilMap(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		RecentNPCTargets: nil,
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10, no focus-fire bonus
	assert.InDelta(t, 10.0, score.Score, 0.01)
	assert.Equal(t, 0, score.AllyCount)
}

func TestAssessSingleThreat_FocusFire_SelfExcluded(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeSlashingCreature(),
		CombatantStats:   map[string]CombatantStats{"pc1": {MaxHP: 50, AC: 15}},
		RecentNPCTargets: map[string]string{"npc1": "pc1"}, // self targeting pc1
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10, self excluded from ally count
	assert.InDelta(t, 10.0, score.Score, 0.01)
	assert.Equal(t, 0, score.AllyCount)
}

func TestAssessSingleThreat_FocusFire_OverriddenByImmunity(t *testing.T) {
	t.Parallel()

	enemy := makeParticipant("pc1", true, 50, 1, 0)

	input := &TurnInput{
		ActiveNPC:        makeParticipant("npc1", false, 50, 0, 0),
		CreatureTemplate: makeFireBreathCreature(),
		CombatantStats: map[string]CombatantStats{
			"pc1": {MaxHP: 50, AC: 15, Immunities: []string{"fire"}},
		},
		RecentNPCTargets: map[string]string{"npc2": "pc1", "npc3": "pc1"},
	}

	score := assessSingleThreat(input, &enemy, input.CombatantStats["pc1"])

	// Base 10 - 50 (fire immune) + 30 (2 allies) = -10
	assert.InDelta(t, -10.0, score.Score, 0.01)
	assert.Equal(t, 2, score.AllyCount)
}

func TestCountAlliesTargeting(t *testing.T) {
	t.Parallel()

	targets := map[string]string{
		"npc1": "pc1",
		"npc2": "pc1",
		"npc3": "pc2",
		"npc4": "pc1",
	}

	assert.Equal(t, 3, countAlliesTargeting(targets, "pc1", "npc99"))
	assert.Equal(t, 2, countAlliesTargeting(targets, "pc1", "npc1")) // self excluded
	assert.Equal(t, 1, countAlliesTargeting(targets, "pc2", "npc99"))
	assert.Equal(t, 0, countAlliesTargeting(targets, "pc3", "npc99"))
	assert.Equal(t, 0, countAlliesTargeting(nil, "pc1", "npc1"))
}
