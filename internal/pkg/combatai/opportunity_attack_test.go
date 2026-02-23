package combatai

import (
	"math/rand"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// --- Helper factories ---

func makeNPCWithCoords(id string, hp int, x, y int) models.ParticipantFull {
	return models.ParticipantFull{
		InstanceID:        id,
		IsPlayerCharacter: false,
		CellsCoords:       &models.CellsCoordinates{CellsX: x, CellsY: y},
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: hp,
			MaxHP:     100,
		},
	}
}

func makePCWithCoords(id string, hp int, x, y int) models.ParticipantFull {
	return models.ParticipantFull{
		InstanceID:        id,
		IsPlayerCharacter: true,
		CellsCoords:       &models.CellsCoordinates{CellsX: x, CellsY: y},
		RuntimeState: models.CreatureRuntimeState{
			CurrentHP: hp,
			MaxHP:     100,
		},
	}
}

func makeCreatureWithMelee(reach int) *models.Creature {
	return &models.Creature{
		StructuredActions: []models.StructuredAction{
			{
				ID:       "claw",
				Name:     "Claw",
				Category: models.ActionCategoryAction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 5,
					Reach: reach,
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d6", Bonus: 3, DamageType: "slashing"},
					},
				},
			},
		},
	}
}

func makeCreatureWithReaction() *models.Creature {
	return &models.Creature{
		StructuredActions: []models.StructuredAction{
			{
				ID:       "claw",
				Name:     "Claw",
				Category: models.ActionCategoryAction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 5,
					Reach: 5,
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d6", Bonus: 3, DamageType: "slashing"},
					},
				},
			},
			{
				ID:       "tail_swipe",
				Name:     "Tail Swipe",
				Category: models.ActionCategoryReaction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 7,
					Reach: 10,
					Damage: []models.DamageRoll{
						{DiceCount: 2, DiceType: "d8", Bonus: 4, DamageType: "bludgeoning"},
					},
				},
			},
		},
	}
}

func makeCreatureRangedOnly() *models.Creature {
	return &models.Creature{
		StructuredActions: []models.StructuredAction{
			{
				ID:       "longbow",
				Name:     "Longbow",
				Category: models.ActionCategoryAction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollRangedWeapon,
					Bonus: 4,
					Range: &models.RangeData{Normal: 150, Long: 600},
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d8", Bonus: 2, DamageType: "piercing"},
					},
				},
			},
		},
	}
}

// --- FindOpportunityAttackCandidates tests ---

func TestFindOACandidates_PCLeavesReach(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("goblin-1", 30, 0, 0)
	pc := makePCWithCoords("fighter-1", 50, 1, 0)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"goblin-1": makeCreatureWithMelee(5),
	}

	oldPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0} // 5ft from NPC
	newPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0} // 10ft from NPC

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].NPC.InstanceID != "goblin-1" {
		t.Errorf("expected goblin-1, got %s", candidates[0].NPC.InstanceID)
	}
	if candidates[0].Action.ID != "claw" {
		t.Errorf("expected claw action, got %s", candidates[0].Action.ID)
	}
}

func TestFindOACandidates_PCStaysInReach(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("goblin-1", 30, 0, 0)
	pc := makePCWithCoords("fighter-1", 50, 0, 1)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"goblin-1": makeCreatureWithMelee(5),
	}

	oldPos := &models.CellsCoordinates{CellsX: 0, CellsY: 1} // 5ft
	newPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0} // 5ft (Chebyshev)

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (still in reach), got %d", len(candidates))
	}
}

func TestFindOACandidates_PCEntersReach(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("goblin-1", 30, 0, 0)
	pc := makePCWithCoords("fighter-1", 50, 3, 0)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"goblin-1": makeCreatureWithMelee(5),
	}

	oldPos := &models.CellsCoordinates{CellsX: 3, CellsY: 0} // 15ft (out)
	newPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0} // 5ft (in)

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (entering reach, not leaving), got %d", len(candidates))
	}
}

func TestFindOACandidates_NPCReactionUsed(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("goblin-1", 30, 0, 0)
	npc.RuntimeState.Resources.ReactionUsed = true
	pc := makePCWithCoords("fighter-1", 50, 1, 0)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"goblin-1": makeCreatureWithMelee(5),
	}

	oldPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0}
	newPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (reaction used), got %d", len(candidates))
	}
}

func TestFindOACandidates_NPCDead(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("goblin-1", 0, 0, 0) // dead
	pc := makePCWithCoords("fighter-1", 50, 1, 0)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"goblin-1": makeCreatureWithMelee(5),
	}

	oldPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0}
	newPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (NPC dead), got %d", len(candidates))
	}
}

func TestFindOACandidates_NPCIncapacitated(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("goblin-1", 30, 0, 0)
	npc.RuntimeState.Conditions = []models.ActiveCondition{
		{Condition: models.ConditionStunned},
	}
	pc := makePCWithCoords("fighter-1", 50, 1, 0)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"goblin-1": makeCreatureWithMelee(5),
	}

	oldPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0}
	newPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (NPC incapacitated), got %d", len(candidates))
	}
}

func TestFindOACandidates_NPCNoMeleeAttack(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("archer-1", 30, 0, 0)
	pc := makePCWithCoords("fighter-1", 50, 1, 0)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"archer-1": makeCreatureRangedOnly(),
	}

	oldPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0}
	newPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (no melee attack), got %d", len(candidates))
	}
}

func TestFindOACandidates_MultipleNPCs(t *testing.T) {
	t.Parallel()

	npc1 := makeNPCWithCoords("goblin-1", 30, 0, 0)  // in reach
	npc2 := makeNPCWithCoords("goblin-2", 30, 0, 1)   // in reach (Chebyshev: dist to (1,0) = 1 cell = 5ft)
	npc3 := makeNPCWithCoords("goblin-3", 30, 10, 10) // out of reach
	pc := makePCWithCoords("fighter-1", 50, 1, 0)

	participants := []models.ParticipantFull{npc1, npc2, npc3, pc}
	creature := makeCreatureWithMelee(5)
	creatures := map[string]*models.Creature{
		"goblin-1": creature,
		"goblin-2": creature,
		"goblin-3": creature,
	}

	oldPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0}
	newPos := &models.CellsCoordinates{CellsX: 3, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	ids := map[string]bool{candidates[0].NPC.InstanceID: true, candidates[1].NPC.InstanceID: true}
	if !ids["goblin-1"] || !ids["goblin-2"] {
		t.Errorf("expected goblin-1 and goblin-2, got %v", ids)
	}
}

func TestFindOACandidates_LongReach(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("dragon-1", 200, 0, 0)
	pc := makePCWithCoords("fighter-1", 50, 2, 0)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"dragon-1": makeCreatureWithMelee(10), // 10ft reach
	}

	// PC at (2,0) = 10ft — within reach. Moving to (3,0) = 15ft — out of reach.
	oldPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0}
	newPos := &models.CellsCoordinates{CellsX: 3, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate (10ft reach), got %d", len(candidates))
	}
}

func TestFindOACandidates_NilOldPos(t *testing.T) {
	t.Parallel()

	npc := makeNPCWithCoords("goblin-1", 30, 0, 0)
	pc := makePCWithCoords("fighter-1", 50, 1, 0)

	participants := []models.ParticipantFull{npc, pc}
	creatures := map[string]*models.Creature{
		"goblin-1": makeCreatureWithMelee(5),
	}

	newPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", nil, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (nil old pos), got %d", len(candidates))
	}
}

func TestFindOACandidates_OnlyNPCsReact(t *testing.T) {
	t.Parallel()

	// Another PC is in reach — should not trigger (PCs don't auto-react).
	pc2 := makePCWithCoords("paladin-1", 60, 0, 0)
	pc := makePCWithCoords("fighter-1", 50, 1, 0)

	participants := []models.ParticipantFull{pc2, pc}
	creatures := map[string]*models.Creature{}

	oldPos := &models.CellsCoordinates{CellsX: 1, CellsY: 0}
	newPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "fighter-1", oldPos, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (PCs don't auto-react), got %d", len(candidates))
	}
}

func TestFindOACandidates_SelfExcluded(t *testing.T) {
	t.Parallel()

	// NPC moving — should not attack itself.
	npc := makeNPCWithCoords("goblin-1", 30, 0, 0)

	participants := []models.ParticipantFull{npc}
	creatures := map[string]*models.Creature{
		"goblin-1": makeCreatureWithMelee(5),
	}

	oldPos := &models.CellsCoordinates{CellsX: 0, CellsY: 0}
	newPos := &models.CellsCoordinates{CellsX: 2, CellsY: 0}

	candidates := FindOpportunityAttackCandidates(participants, "goblin-1", oldPos, newPos, creatures)

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates (self excluded), got %d", len(candidates))
	}
}

// --- selectOpportunityAttackAction tests ---

func TestSelectOAAction_ExplicitReaction(t *testing.T) {
	t.Parallel()

	creature := makeCreatureWithReaction()

	action := selectOpportunityAttackAction(creature)

	if action == nil {
		t.Fatal("expected reaction action, got nil")
	}
	if action.ID != "tail_swipe" {
		t.Errorf("expected tail_swipe (reaction category), got %s", action.ID)
	}
}

func TestSelectOAAction_FallbackToBestMelee(t *testing.T) {
	t.Parallel()

	creature := &models.Creature{
		StructuredActions: []models.StructuredAction{
			{
				ID:       "bite",
				Name:     "Bite",
				Category: models.ActionCategoryAction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 5,
					Reach: 5,
					Damage: []models.DamageRoll{
						{DiceCount: 2, DiceType: "d6", Bonus: 3, DamageType: "piercing"},
					},
				},
			},
			{
				ID:       "claw",
				Name:     "Claw",
				Category: models.ActionCategoryAction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 5,
					Reach: 5,
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d4", Bonus: 3, DamageType: "slashing"},
					},
				},
			},
		},
	}

	action := selectOpportunityAttackAction(creature)

	if action == nil {
		t.Fatal("expected fallback melee action, got nil")
	}
	// Bite: 2d6+3 avg = 10, Claw: 1d4+3 avg = 5.5 → bite is best
	if action.ID != "bite" {
		t.Errorf("expected bite (higher damage), got %s", action.ID)
	}
}

func TestSelectOAAction_SkipsRanged(t *testing.T) {
	t.Parallel()

	creature := makeCreatureRangedOnly()

	action := selectOpportunityAttackAction(creature)

	if action != nil {
		t.Errorf("expected nil (only ranged), got %s", action.ID)
	}
}

func TestSelectOAAction_SkipsRechargeActions(t *testing.T) {
	t.Parallel()

	creature := &models.Creature{
		StructuredActions: []models.StructuredAction{
			{
				ID:       "breath",
				Name:     "Fire Breath",
				Category: models.ActionCategoryAction,
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 8,
					Reach: 5,
					Damage: []models.DamageRoll{
						{DiceCount: 6, DiceType: "d6", Bonus: 0, DamageType: "fire"},
					},
				},
				Recharge: &models.RechargeData{MinRoll: 5},
			},
		},
	}

	action := selectOpportunityAttackAction(creature)

	if action != nil {
		t.Errorf("expected nil (recharge-gated), got %s", action.ID)
	}
}

// --- ShouldTakeOpportunityAttack tests ---

func TestShouldTakeOA_HighIntelligence(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(42))

	// Intelligence = 1.0 → should always take OA.
	for i := 0; i < 100; i++ {
		if !ShouldTakeOpportunityAttack(1.0, rng) {
			t.Fatalf("intelligence=1.0 should always return true, failed at iteration %d", i)
		}
	}
}

func TestShouldTakeOA_LowIntelligence(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(42))

	// Intelligence = 0.05 → should mostly return false.
	hits := 0
	total := 1000
	for i := 0; i < total; i++ {
		if ShouldTakeOpportunityAttack(0.05, rng) {
			hits++
		}
	}

	// Should be around 5% ± tolerance.
	ratio := float64(hits) / float64(total)
	if ratio > 0.15 {
		t.Errorf("intelligence=0.05: hit ratio %.2f, expected ~0.05", ratio)
	}
}

func TestShouldTakeOA_MediumIntelligence(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(42))

	hits := 0
	total := 1000
	for i := 0; i < total; i++ {
		if ShouldTakeOpportunityAttack(0.5, rng) {
			hits++
		}
	}

	// Should be around 50% ± 10%.
	ratio := float64(hits) / float64(total)
	if ratio < 0.35 || ratio > 0.65 {
		t.Errorf("intelligence=0.5: hit ratio %.2f, expected ~0.50", ratio)
	}
}

// --- meleeReach tests ---

func TestMeleeReach_Default(t *testing.T) {
	t.Parallel()

	if r := meleeReach(nil); r != 5 {
		t.Errorf("nil action: got %d, want 5", r)
	}

	action := &models.StructuredAction{} // no Attack data
	if r := meleeReach(action); r != 5 {
		t.Errorf("no attack data: got %d, want 5", r)
	}
}

func TestMeleeReach_CustomReach(t *testing.T) {
	t.Parallel()

	action := &models.StructuredAction{
		Attack: &models.AttackRollData{Reach: 10},
	}
	if r := meleeReach(action); r != 10 {
		t.Errorf("custom reach: got %d, want 10", r)
	}
}
