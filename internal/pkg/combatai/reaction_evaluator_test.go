package combatai

import (
	"math/rand"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// --- Test helpers ---

func makeShieldCandidate(attackTotal, currentAC int, intelligence float64, reactionUsed bool) *ShieldCandidate {
	npc := &models.ParticipantFull{
		InstanceID: "npc-1",
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: 50,
			MaxHP:     100,
			Resources: models.ResourceState{
				ReactionUsed: reactionUsed,
			},
		},
	}
	creature := &models.Creature{
		Spellcasting: &models.Spellcasting{
			SpellSlots: map[int]int{1: 4},
			SpellsByLevel: map[int][]models.SpellKnown{
				1: {{SpellID: "shield", Name: "Shield", Level: 1}},
			},
		},
	}
	return &ShieldCandidate{
		NPC:          npc,
		Creature:     creature,
		AttackTotal:  attackTotal,
		CurrentAC:    currentAC,
		Intelligence: intelligence,
	}
}

func deterministicRng(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

// --- ShouldCastShield tests ---

func TestShouldCastShield_HitToMiss(t *testing.T) {
	// Attack 17 hits AC 14, but misses AC 14+5=19 → Shield helps
	c := makeShieldCandidate(17, 14, 1.0, false)
	rng := deterministicRng(42)
	if !ShouldCastShield(c, rng) {
		t.Error("expected Shield to trigger when attack hits but would miss with +5 AC")
	}
}

func TestShouldCastShield_AlreadyMisses(t *testing.T) {
	// Attack 12 < AC 14 → already misses, no need for Shield
	c := makeShieldCandidate(12, 14, 1.0, false)
	rng := deterministicRng(42)
	if ShouldCastShield(c, rng) {
		t.Error("expected Shield NOT to trigger when attack already misses")
	}
}

func TestShouldCastShield_StillHits(t *testing.T) {
	// Attack 25 >= AC 14+5=19 → Shield won't help
	c := makeShieldCandidate(25, 14, 1.0, false)
	rng := deterministicRng(42)
	if ShouldCastShield(c, rng) {
		t.Error("expected Shield NOT to trigger when attack still hits with +5 AC")
	}
}

func TestShouldCastShield_ReactionUsed(t *testing.T) {
	c := makeShieldCandidate(17, 14, 1.0, true)
	rng := deterministicRng(42)
	if ShouldCastShield(c, rng) {
		t.Error("expected Shield NOT to trigger when reaction already used")
	}
}

func TestShouldCastShield_NoSpellSlots(t *testing.T) {
	c := makeShieldCandidate(17, 14, 1.0, false)
	// Template has 4 max slots, but runtime remaining is 0 (all exhausted).
	c.Creature.Spellcasting.SpellSlots = map[int]int{1: 4}
	c.NPC.RuntimeState.Resources.SpellSlots = map[int]int{1: 0}
	rng := deterministicRng(42)
	if ShouldCastShield(c, rng) {
		t.Error("expected Shield NOT to trigger when no spell slots remaining")
	}
}

func TestShouldCastShield_IntelligenceGate(t *testing.T) {
	// Low intelligence NPC with seeded RNG that generates value > intelligence
	c := makeShieldCandidate(17, 14, 0.05, false)
	// Seed 0: first Float64() = 0.9451... > 0.05 → gate fails
	rng := deterministicRng(0)
	if ShouldCastShield(c, rng) {
		t.Error("expected Shield NOT to trigger when intelligence gate fails")
	}
}

func TestShouldCastShield_ExactlyAtACPlusFive(t *testing.T) {
	// Attack 19 = AC 14+5 → attack >= AC+5, so Shield won't help
	c := makeShieldCandidate(19, 14, 1.0, false)
	rng := deterministicRng(42)
	if ShouldCastShield(c, rng) {
		t.Error("expected Shield NOT to trigger when attack total equals AC+5")
	}
}

func TestShouldCastShield_ExactlyAtAC(t *testing.T) {
	// Attack 14 = AC 14 → hits, and +5 would make 19 > 14, so Shield helps
	c := makeShieldCandidate(14, 14, 1.0, false)
	rng := deterministicRng(42)
	if !ShouldCastShield(c, rng) {
		t.Error("expected Shield to trigger when attack exactly matches AC")
	}
}

// --- ShouldCastCounterspell tests ---

func makeCounterspellCandidate(spellLevel, distance int, intelligence float64, reactionUsed bool) *CounterspellCandidate {
	npc := &models.ParticipantFull{
		InstanceID: "npc-2",
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: 50,
			MaxHP:     100,
			Resources: models.ResourceState{
				ReactionUsed: reactionUsed,
			},
		},
	}
	creature := &models.Creature{
		Spellcasting: &models.Spellcasting{
			SpellSlots: map[int]int{3: 3},
			SpellsByLevel: map[int][]models.SpellKnown{
				3: {{SpellID: "counterspell", Name: "Counterspell", Level: 3}},
			},
		},
	}
	return &CounterspellCandidate{
		NPC:          npc,
		Creature:     creature,
		SpellLevel:   spellLevel,
		Distance:     distance,
		Intelligence: intelligence,
	}
}

func TestShouldCastCounterspell_InRange(t *testing.T) {
	c := makeCounterspellCandidate(3, 30, 1.0, false)
	rng := deterministicRng(42)
	if !ShouldCastCounterspell(c, rng) {
		t.Error("expected Counterspell to trigger for level 3 spell within 30ft")
	}
}

func TestShouldCastCounterspell_TooFar(t *testing.T) {
	c := makeCounterspellCandidate(3, 65, 1.0, false)
	rng := deterministicRng(42)
	if ShouldCastCounterspell(c, rng) {
		t.Error("expected Counterspell NOT to trigger when caster is 65ft away")
	}
}

func TestShouldCastCounterspell_Cantrip(t *testing.T) {
	c := makeCounterspellCandidate(0, 30, 1.0, false)
	rng := deterministicRng(42)
	if ShouldCastCounterspell(c, rng) {
		t.Error("expected Counterspell NOT to trigger for cantrips")
	}
}

func TestShouldCastCounterspell_ReactionUsed(t *testing.T) {
	c := makeCounterspellCandidate(3, 30, 1.0, true)
	rng := deterministicRng(42)
	if ShouldCastCounterspell(c, rng) {
		t.Error("expected Counterspell NOT to trigger when reaction already used")
	}
}

func TestShouldCastCounterspell_HighLevelBoost(t *testing.T) {
	// Level 5+ spell gets +0.2 intelligence boost
	c := makeCounterspellCandidate(5, 30, 0.3, false)
	// With boost: threshold = 0.3 + 0.2 = 0.5
	// Seed 42: first Float64() ≈ 0.37... < 0.5 → should pass
	rng := deterministicRng(42)
	if !ShouldCastCounterspell(c, rng) {
		t.Error("expected Counterspell to trigger with high-level spell intelligence boost")
	}
}

func TestShouldCastCounterspell_NoSlots(t *testing.T) {
	c := makeCounterspellCandidate(3, 30, 1.0, false)
	c.NPC.RuntimeState.Resources.SpellSlots = map[int]int{3: 0}
	c.Creature.Spellcasting.SpellSlots = map[int]int{3: 0}
	rng := deterministicRng(42)
	if ShouldCastCounterspell(c, rng) {
		t.Error("expected Counterspell NOT to trigger when no level 3+ slots remain")
	}
}

// --- ShouldParry tests ---

func makeParryCandidate(incomingDmg int, maxHP int, intelligence float64, reactionUsed bool) *ParryCandidate {
	npc := &models.ParticipantFull{
		InstanceID: "npc-3",
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: maxHP,
			MaxHP:     maxHP,
			Resources: models.ResourceState{
				ReactionUsed: reactionUsed,
			},
		},
	}
	creature := &models.Creature{
		StructuredActions: []models.StructuredAction{
			{
				ID:       "parry",
				Name:     "Parry",
				Category: models.ActionCategoryReaction,
			},
		},
	}
	return &ParryCandidate{
		NPC:          npc,
		Creature:     creature,
		IncomingDmg:  incomingDmg,
		Intelligence: intelligence,
	}
}

func TestShouldParry_SignificantDamage(t *testing.T) {
	// 15 damage vs 100 max HP → 15% > 10% threshold
	c := makeParryCandidate(15, 100, 1.0, false)
	rng := deterministicRng(42)
	if !ShouldParry(c, rng) {
		t.Error("expected Parry to trigger for significant damage (>10% HP)")
	}
}

func TestShouldParry_TrivialDamage(t *testing.T) {
	// 2 damage vs 100 max HP → 2% < 10% threshold
	c := makeParryCandidate(2, 100, 1.0, false)
	rng := deterministicRng(42)
	if ShouldParry(c, rng) {
		t.Error("expected Parry NOT to trigger for trivial damage (<10% HP)")
	}
}

func TestShouldParry_ReactionUsed(t *testing.T) {
	c := makeParryCandidate(15, 100, 1.0, true)
	rng := deterministicRng(42)
	if ShouldParry(c, rng) {
		t.Error("expected Parry NOT to trigger when reaction already used")
	}
}

func TestShouldParry_NoParryReaction(t *testing.T) {
	c := makeParryCandidate(15, 100, 1.0, false)
	c.Creature.StructuredActions = nil // no Parry action
	rng := deterministicRng(42)
	if ShouldParry(c, rng) {
		t.Error("expected Parry NOT to trigger when creature has no Parry reaction")
	}
}

// --- ParryReduction tests ---

func TestParryReduction(t *testing.T) {
	tests := []struct {
		name     string
		profStr  string
		dex      int
		expected int
	}{
		{"prof 3 dex 14", "+3", 14, 5}, // prof=3, dexMod=(14-10)/2=2, total=5
		{"prof 2 dex 10", "+2", 10, 2}, // prof=2, dexMod=0, total=2
		{"prof 4 dex 16", "+4", 16, 7}, // prof=4, dexMod=3, total=7
		{"prof 2 dex 8", "+2", 8, 1},   // prof=2, dexMod=-1, total=1
		{"prof 0 dex 6", "", 6, 0},     // prof=0, dexMod=-2, total=0 (clamped)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creature := &models.Creature{
				ProficiencyBonus: tt.profStr,
				Ability:          models.Ability{Dex: tt.dex},
			}
			got := ParryReduction(creature)
			if got != tt.expected {
				t.Errorf("ParryReduction() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// --- FindShieldSpell tests ---

func TestFindShieldSpell_Regular(t *testing.T) {
	creature := &models.Creature{
		Spellcasting: &models.Spellcasting{
			SpellSlots: map[int]int{1: 4},
			SpellsByLevel: map[int][]models.SpellKnown{
				1: {{SpellID: "shield", Name: "Shield", Level: 1}},
			},
		},
	}
	resources := &models.ResourceState{}
	id, level, _ := FindShieldSpell(creature, resources)
	if id == "" {
		t.Fatal("expected to find Shield spell")
	}
	if level != 1 {
		t.Errorf("expected slot level 1, got %d", level)
	}
}

func TestFindShieldSpell_Innate(t *testing.T) {
	creature := &models.Creature{
		InnateSpellcasting: &models.InnateSpellcasting{
			AtWill: []models.SpellKnown{
				{SpellID: "shield", Name: "Shield", Level: 1},
			},
		},
	}
	resources := &models.ResourceState{}
	id, level, _ := FindShieldSpell(creature, resources)
	if id == "" {
		t.Fatal("expected to find innate Shield spell")
	}
	if level != 0 {
		t.Errorf("expected slot level 0 for innate, got %d", level)
	}
}

func TestFindShieldSpell_NotFound(t *testing.T) {
	creature := &models.Creature{
		Spellcasting: &models.Spellcasting{
			SpellSlots: map[int]int{1: 4},
			SpellsByLevel: map[int][]models.SpellKnown{
				1: {{SpellID: "magic-missile", Name: "Magic Missile", Level: 1}},
			},
		},
	}
	resources := &models.ResourceState{}
	id, _, _ := FindShieldSpell(creature, resources)
	if id != "" {
		t.Errorf("expected Shield not found, got %q", id)
	}
}

func TestFindShieldSpell_SlotsExhausted(t *testing.T) {
	creature := &models.Creature{
		Spellcasting: &models.Spellcasting{
			SpellSlots: map[int]int{1: 2},
			SpellsByLevel: map[int][]models.SpellKnown{
				1: {{SpellID: "shield", Name: "Shield", Level: 1}},
			},
		},
	}
	// All level 1 slots exhausted (remaining = 0).
	resources := &models.ResourceState{
		SpellSlots: map[int]int{1: 0},
	}
	id, _, _ := FindShieldSpell(creature, resources)
	if id != "" {
		t.Errorf("expected Shield not found when slots exhausted, got %q", id)
	}
}

// --- FindCounterspellSpell tests ---

func TestFindCounterspellSpell(t *testing.T) {
	creature := &models.Creature{
		Spellcasting: &models.Spellcasting{
			SpellSlots: map[int]int{3: 3},
			Spells:     []models.SpellKnown{{SpellID: "counterspell", Name: "Counterspell", Level: 3}},
		},
	}
	resources := &models.ResourceState{}
	id, level, _ := FindCounterspellSpell(creature, resources)
	if id == "" {
		t.Fatal("expected to find Counterspell")
	}
	if level != 3 {
		t.Errorf("expected slot level 3, got %d", level)
	}
}

// --- FindParryReaction tests ---

func TestFindParryReaction_Found(t *testing.T) {
	creature := &models.Creature{
		StructuredActions: []models.StructuredAction{
			{ID: "claw", Name: "Claw", Category: models.ActionCategoryAction},
			{ID: "parry", Name: "Parry", Category: models.ActionCategoryReaction},
		},
	}
	action := FindParryReaction(creature)
	if action == nil {
		t.Fatal("expected to find Parry reaction")
	}
	if action.ID != "parry" {
		t.Errorf("expected action ID 'parry', got %q", action.ID)
	}
}

func TestFindParryReaction_NotFound(t *testing.T) {
	creature := &models.Creature{
		StructuredActions: []models.StructuredAction{
			{ID: "claw", Name: "Claw", Category: models.ActionCategoryAction},
		},
	}
	action := FindParryReaction(creature)
	if action != nil {
		t.Error("expected Parry reaction not found")
	}
}

func TestFindParryReaction_CaseInsensitive(t *testing.T) {
	creature := &models.Creature{
		StructuredActions: []models.StructuredAction{
			{ID: "reaction-1", Name: "PARRY", Category: models.ActionCategoryReaction},
		},
	}
	action := FindParryReaction(creature)
	if action == nil {
		t.Fatal("expected to find PARRY reaction (case-insensitive)")
	}
}

// --- EffectiveAC tests ---

func TestEffectiveAC_NoModifiers(t *testing.T) {
	p := &models.ParticipantFull{
		RuntimeState: models.CreatureRuntimeState{},
	}
	ac := EffectiveAC(p, 15)
	if ac != 15 {
		t.Errorf("expected AC 15 with no modifiers, got %d", ac)
	}
}

func TestEffectiveAC_WithShieldModifier(t *testing.T) {
	p := &models.ParticipantFull{
		RuntimeState: models.CreatureRuntimeState{
			StatModifiers: []models.StatModifier{
				{
					ID:   "shield-npc-1",
					Name: "Shield",
					Modifiers: []models.ModifierEffect{
						{Target: models.ModTargetAC, Operation: models.ModOpAdd, Value: 5},
					},
				},
			},
		},
	}
	ac := EffectiveAC(p, 15)
	if ac != 20 {
		t.Errorf("expected AC 20 with Shield +5, got %d", ac)
	}
}

func TestEffectiveAC_MultipleModifiers(t *testing.T) {
	p := &models.ParticipantFull{
		RuntimeState: models.CreatureRuntimeState{
			StatModifiers: []models.StatModifier{
				{
					ID:   "shield",
					Name: "Shield",
					Modifiers: []models.ModifierEffect{
						{Target: models.ModTargetAC, Operation: models.ModOpAdd, Value: 5},
					},
				},
				{
					ID:   "other",
					Name: "Shield of Faith",
					Modifiers: []models.ModifierEffect{
						{Target: models.ModTargetAC, Operation: models.ModOpAdd, Value: 2},
					},
				},
			},
		},
	}
	ac := EffectiveAC(p, 15)
	if ac != 22 {
		t.Errorf("expected AC 22 with Shield+5 and Shield of Faith+2, got %d", ac)
	}
}

func TestEffectiveAC_IgnoresNonACModifiers(t *testing.T) {
	p := &models.ParticipantFull{
		RuntimeState: models.CreatureRuntimeState{
			StatModifiers: []models.StatModifier{
				{
					ID:   "dodge",
					Name: "Dodge",
					Modifiers: []models.ModifierEffect{
						{Target: models.ModTargetAttackRolls, Operation: models.ModOpDisadvantage},
					},
				},
			},
		},
	}
	ac := EffectiveAC(p, 15)
	if ac != 15 {
		t.Errorf("expected AC 15 (non-AC modifiers ignored), got %d", ac)
	}
}

// --- Innate per-day spell tests ---

func TestFindShieldSpell_InnatePerDay_Available(t *testing.T) {
	creature := &models.Creature{
		InnateSpellcasting: &models.InnateSpellcasting{
			PerDay: map[int][]models.SpellKnown{
				3: {{SpellID: "shield", Name: "Shield", Level: 1}},
			},
		},
	}
	// AbilityUses nil → untracked → all uses available
	resources := &models.ResourceState{}
	id, level, innateKey := FindShieldSpell(creature, resources)
	if id == "" {
		t.Fatal("expected to find innate per-day Shield spell")
	}
	if level != 0 {
		t.Errorf("expected slot level 0 for innate, got %d", level)
	}
	if innateKey == "" {
		t.Error("expected non-empty innateKey for per-day spell")
	}
	if innateKey != "innate:shield" {
		t.Errorf("expected innateKey 'innate:shield', got %q", innateKey)
	}
}

func TestFindShieldSpell_InnatePerDay_Exhausted(t *testing.T) {
	creature := &models.Creature{
		InnateSpellcasting: &models.InnateSpellcasting{
			PerDay: map[int][]models.SpellKnown{
				3: {{SpellID: "shield", Name: "Shield", Level: 1}},
			},
		},
	}
	// AbilityUses[key] = 0 → all uses spent
	resources := &models.ResourceState{
		AbilityUses: map[string]int{"innate:shield": 0},
	}
	id, _, _ := FindShieldSpell(creature, resources)
	if id != "" {
		t.Errorf("expected innate per-day Shield not found when exhausted, got %q", id)
	}
}

func TestFindShieldSpell_InnatePerDay_PartiallyUsed(t *testing.T) {
	creature := &models.Creature{
		InnateSpellcasting: &models.InnateSpellcasting{
			PerDay: map[int][]models.SpellKnown{
				3: {{SpellID: "shield", Name: "Shield", Level: 1}},
			},
		},
	}
	// 1 of 3 uses remaining
	resources := &models.ResourceState{
		AbilityUses: map[string]int{"innate:shield": 1},
	}
	id, _, innateKey := FindShieldSpell(creature, resources)
	if id == "" {
		t.Fatal("expected innate per-day Shield available with 1 use remaining")
	}
	if innateKey != "innate:shield" {
		t.Errorf("expected innateKey 'innate:shield', got %q", innateKey)
	}
}

func TestFindShieldSpell_InnatePerDay_FallbackToRegular(t *testing.T) {
	creature := &models.Creature{
		InnateSpellcasting: &models.InnateSpellcasting{
			PerDay: map[int][]models.SpellKnown{
				1: {{SpellID: "shield", Name: "Shield", Level: 1}},
			},
		},
		Spellcasting: &models.Spellcasting{
			SpellSlots: map[int]int{1: 4},
			SpellsByLevel: map[int][]models.SpellKnown{
				1: {{SpellID: "shield", Name: "Shield", Level: 1}},
			},
		},
	}
	// Innate 1/day exhausted → should fall through to regular spellcasting
	resources := &models.ResourceState{
		AbilityUses: map[string]int{"innate:shield": 0},
	}
	id, level, innateKey := FindShieldSpell(creature, resources)
	if id == "" {
		t.Fatal("expected Shield found via regular spellcasting after innate exhausted")
	}
	if level != 1 {
		t.Errorf("expected slot level 1 from regular spellcasting, got %d", level)
	}
	if innateKey != "" {
		t.Errorf("expected empty innateKey for regular spellcasting, got %q", innateKey)
	}
}

// --- HasIncapacitatingCondition tests ---

func TestHasIncapacitatingCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition models.ConditionType
		expected  bool
	}{
		{"stunned", models.ConditionStunned, true},
		{"paralyzed", models.ConditionParalyzed, true},
		{"incapacitated", models.ConditionIncapacitated, true},
		{"petrified", models.ConditionPetrified, true},
		{"unconscious", models.ConditionUnconscious, true},
		{"frightened", models.ConditionFrightened, false},
		{"poisoned", models.ConditionPoisoned, false},
		{"restrained", models.ConditionRestrained, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &models.ParticipantFull{
				RuntimeState: models.CreatureRuntimeState{
					Conditions: []models.ActiveCondition{
						{Condition: tt.condition},
					},
				},
			}
			got := HasIncapacitatingCondition(p)
			if got != tt.expected {
				t.Errorf("HasIncapacitatingCondition(%s) = %v, want %v", tt.condition, got, tt.expected)
			}
		})
	}
}

func TestHasIncapacitatingCondition_NoConditions(t *testing.T) {
	p := &models.ParticipantFull{}
	if HasIncapacitatingCondition(p) {
		t.Error("expected false for participant with no conditions")
	}
}

func TestShouldCastShield_IncapacitatedBlocked(t *testing.T) {
	c := makeShieldCandidate(17, 14, 1.0, false)
	c.NPC.RuntimeState.Conditions = []models.ActiveCondition{
		{Condition: models.ConditionStunned},
	}
	rng := deterministicRng(42)
	if ShouldCastShield(c, rng) {
		t.Error("expected Shield NOT to trigger when NPC is stunned")
	}
}

func TestShouldCastCounterspell_IncapacitatedBlocked(t *testing.T) {
	c := makeCounterspellCandidate(3, 30, 1.0, false)
	c.NPC.RuntimeState.Conditions = []models.ActiveCondition{
		{Condition: models.ConditionParalyzed},
	}
	rng := deterministicRng(42)
	if ShouldCastCounterspell(c, rng) {
		t.Error("expected Counterspell NOT to trigger when NPC is paralyzed")
	}
}

func TestShouldParry_IncapacitatedBlocked(t *testing.T) {
	c := makeParryCandidate(15, 100, 1.0, false)
	c.NPC.RuntimeState.Conditions = []models.ActiveCondition{
		{Condition: models.ConditionUnconscious},
	}
	rng := deterministicRng(42)
	if ShouldParry(c, rng) {
		t.Error("expected Parry NOT to trigger when NPC is unconscious")
	}
}

func TestSpellIDOrName_NormalizesCase(t *testing.T) {
	tests := []struct {
		name     string
		spell    models.SpellKnown
		expected string
	}{
		{"SpellID lowercase", models.SpellKnown{SpellID: "shield"}, "shield"},
		{"SpellID mixed case", models.SpellKnown{SpellID: "Shield"}, "shield"},
		{"SpellID uppercase", models.SpellKnown{SpellID: "COUNTERSPELL"}, "counterspell"},
		{"Name only", models.SpellKnown{Name: "Shield"}, "shield"},
		{"Name lowercase", models.SpellKnown{Name: "shield"}, "shield"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SpellIDOrName(&tt.spell)
			if got != tt.expected {
				t.Errorf("SpellIDOrName() = %q, want %q", got, tt.expected)
			}
		})
	}
}
