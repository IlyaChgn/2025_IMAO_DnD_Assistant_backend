package usecases

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai"
)

type reactionEvaluator struct {
	bestiaryRepo bestiaryinterfaces.BestiaryRepository
	mu           sync.Mutex
	rng          *rand.Rand
}

// NewReactionEvaluator creates a ReactionEvaluator that loads creature data
// from the bestiary and delegates to pure evaluation functions.
func NewReactionEvaluator(
	bestiaryRepo bestiaryinterfaces.BestiaryRepository,
) actionsinterfaces.ReactionEvaluator {
	return &reactionEvaluator{
		bestiaryRepo: bestiaryRepo,
		rng:          rand.New(rand.NewSource(rand.Int63())),
	}
}

// EvaluateShield checks whether the target NPC should cast Shield as a reaction.
// Only the target NPC itself is checked (Shield is self-cast).
func (re *reactionEvaluator) EvaluateShield(
	ctx context.Context,
	ed actionsinterfaces.EncounterDataReader,
	targetID string,
	attackTotal int,
) (*actionsinterfaces.ShieldReactionResult, error) {
	target, _, err := ed.FindParticipantByInstanceID(targetID)
	if err != nil {
		return nil, nil // target not found — no reaction
	}

	// Only NPCs react automatically.
	if target.IsPlayerCharacter {
		return nil, nil
	}

	// Skip dead NPCs.
	if combatai.GetCurrentHP(target) <= 0 {
		return nil, nil
	}

	// Load creature template.
	creature, err := re.bestiaryRepo.GetCreatureByID(ctx, target.CreatureID)
	if err != nil {
		return nil, fmt.Errorf("load creature %s for Shield evaluation: %w", target.CreatureID, err)
	}
	if creature == nil {
		return nil, nil
	}

	intelligence := combatai.ComputeIntelligence(creature.Ability.Int, 0.0)
	currentAC := combatai.EffectiveAC(target, creature.ArmorClass)

	candidate := &combatai.ShieldCandidate{
		NPC:          target,
		Creature:     creature,
		AttackTotal:  attackTotal,
		CurrentAC:    currentAC,
		Intelligence: intelligence,
	}

	re.mu.Lock()
	shouldShield := combatai.ShouldCastShield(candidate, re.rng)
	re.mu.Unlock()

	if !shouldShield {
		return nil, nil
	}

	// Find spell and slot to use.
	spellID, slotLevel, innateKey := combatai.FindShieldSpell(creature, &target.RuntimeState.Resources)
	if spellID == "" {
		return nil, nil // shouldn't happen since ShouldCastShield checks, but be safe
	}

	// Ensure slot/innate use is tracked in runtime (lazy init from template).
	// NOTE: This intentionally mutates participant state before the result is returned.
	// All callers unconditionally apply the result when non-nil, so this is safe.
	if slotLevel > 0 {
		ensureSlotInitialized(&target.RuntimeState.Resources, creature, slotLevel)
	}
	if innateKey != "" {
		ensureInnateInitialized(&target.RuntimeState.Resources, creature, innateKey)
	}

	reactorName := resolveReactorName(target, creature)

	return &actionsinterfaces.ShieldReactionResult{
		ReactorID:      target.InstanceID,
		ReactorName:    reactorName,
		SpellID:        spellID,
		SlotLevel:      slotLevel,
		InnateKey:      innateKey,
		ACBonus:        5,
		NewEffectiveAC: currentAC + 5,
	}, nil
}

// EvaluateCounterspell checks all alive NPC defenders within 60ft of the caster
// for a Counterspell reaction. Returns the first NPC that decides to counterspell.
func (re *reactionEvaluator) EvaluateCounterspell(
	ctx context.Context,
	ed actionsinterfaces.EncounterDataReader,
	casterID string,
	spellLevel int,
) (*actionsinterfaces.CounterspellReactionResult, error) {
	// Find caster to get position.
	caster, _, err := ed.FindParticipantByInstanceID(casterID)
	if err != nil {
		return nil, nil
	}

	// NPCs don't counterspell allied NPCs.
	if !caster.IsPlayerCharacter {
		return nil, nil
	}

	participants := ed.GetParticipants()

	for i := range participants {
		p := &participants[i]

		// Skip self, PCs, dead NPCs.
		if p.InstanceID == casterID || p.IsPlayerCharacter || combatai.GetCurrentHP(p) <= 0 {
			continue
		}

		// Calculate distance.
		dist := combatai.DistanceFt(p.CellsCoords, caster.CellsCoords)

		// Load creature template.
		creature, err := re.bestiaryRepo.GetCreatureByID(ctx, p.CreatureID)
		if err != nil || creature == nil {
			continue
		}

		intelligence := combatai.ComputeIntelligence(creature.Ability.Int, 0.0)

		candidate := &combatai.CounterspellCandidate{
			NPC:          p,
			Creature:     creature,
			SpellLevel:   spellLevel,
			Distance:     dist,
			Intelligence: intelligence,
		}

		re.mu.Lock()
		shouldCS := combatai.ShouldCastCounterspell(candidate, re.rng)
		re.mu.Unlock()

		if !shouldCS {
			continue
		}

		// Find spell and slot.
		spellID, slotLevel, innateKey := combatai.FindCounterspellSpell(creature, &p.RuntimeState.Resources)
		if spellID == "" {
			continue
		}

		// Ensure slot/innate use is tracked in runtime (lazy init from template).
		// NOTE: This intentionally mutates participant state before the result is returned.
		// All callers unconditionally apply the result when non-nil, so this is safe.
		if slotLevel > 0 {
			ensureSlotInitialized(&p.RuntimeState.Resources, creature, slotLevel)
		}
		if innateKey != "" {
			ensureInnateInitialized(&p.RuntimeState.Resources, creature, innateKey)
		}

		reactorName := resolveReactorName(p, creature)

		// Determine if counterspell succeeds.
		// D&D 5e: auto-success if counterspell slot >= spell level.
		// For innate Counterspell (slotLevel=0), treat as cast at 3rd level (its minimum).
		// Otherwise, ability check DC = 10 + spell level.
		effectiveLevel := slotLevel
		if effectiveLevel == 0 {
			effectiveLevel = 3 // Counterspell's minimum spell level
		}

		success := true
		var abilityCheck *int
		checkDC := 0

		if effectiveLevel < spellLevel {
			// Need ability check: DC = 10 + spell level.
			checkDC = 10 + spellLevel
			// Roll d20 + spellcasting ability modifier.
			spellMod := spellcastingAbilityMod(creature)
			re.mu.Lock()
			roll := re.rng.Intn(20) + 1 + spellMod
			re.mu.Unlock()
			abilityCheck = &roll
			success = roll >= checkDC
		}

		return &actionsinterfaces.CounterspellReactionResult{
			ReactorID:    p.InstanceID,
			ReactorName:  reactorName,
			SpellID:      spellID,
			SlotLevel:    slotLevel,
			InnateKey:    innateKey,
			Success:      success,
			AbilityCheck: abilityCheck,
			CheckDC:      checkDC,
		}, nil
	}

	return nil, nil
}

// EvaluateParry checks whether the target NPC should use a Parry reaction.
func (re *reactionEvaluator) EvaluateParry(
	ctx context.Context,
	ed actionsinterfaces.EncounterDataReader,
	targetID string,
	incomingDamage int,
) (*actionsinterfaces.ParryReactionResult, error) {
	target, _, err := ed.FindParticipantByInstanceID(targetID)
	if err != nil {
		return nil, nil
	}

	if target.IsPlayerCharacter || combatai.GetCurrentHP(target) <= 0 {
		return nil, nil
	}

	creature, err := re.bestiaryRepo.GetCreatureByID(ctx, target.CreatureID)
	if err != nil {
		return nil, fmt.Errorf("load creature %s for Parry evaluation: %w", target.CreatureID, err)
	}
	if creature == nil {
		return nil, nil
	}

	intelligence := combatai.ComputeIntelligence(creature.Ability.Int, 0.0)

	candidate := &combatai.ParryCandidate{
		NPC:          target,
		Creature:     creature,
		IncomingDmg:  incomingDamage,
		Intelligence: intelligence,
	}

	re.mu.Lock()
	shouldParry := combatai.ShouldParry(candidate, re.rng)
	re.mu.Unlock()

	if !shouldParry {
		return nil, nil
	}

	parryAction := combatai.FindParryReaction(creature)
	if parryAction == nil {
		return nil, nil
	}

	reactorName := resolveReactorName(target, creature)
	reduction := combatai.ParryReduction(creature)

	return &actionsinterfaces.ParryReactionResult{
		ReactorID:       target.InstanceID,
		ReactorName:     reactorName,
		ActionID:        parryAction.ID,
		DamageReduction: reduction,
	}, nil
}

// resolveReactorName returns a display name for an NPC participant.
func resolveReactorName(p *models.ParticipantFull, creature *models.Creature) string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	if creature.Name.Eng != "" {
		return creature.Name.Eng
	}
	return p.InstanceID
}

// spellcastingAbilityMod returns the creature's spellcasting ability modifier.
// Checks Spellcasting.Ability first, then InnateSpellcasting.Ability, falls back to INT.
func spellcastingAbilityMod(creature *models.Creature) int {
	if creature.Spellcasting != nil && creature.Spellcasting.Ability != "" {
		return abilityModFromCreature(creature, creature.Spellcasting.Ability)
	}
	if creature.InnateSpellcasting != nil && creature.InnateSpellcasting.Ability != "" {
		return abilityModFromCreature(creature, creature.InnateSpellcasting.Ability)
	}
	// Default to INT.
	return abilityMod(creature.Ability.Int)
}

// abilityModFromCreature returns the ability modifier for the given ability type.
// Uses math.Floor for correct D&D 5e floor division (odd scores below 10).
func abilityModFromCreature(creature *models.Creature, ability models.AbilityType) int {
	var score int
	switch models.AbilityType(strings.ToUpper(string(ability))) {
	case models.AbilitySTR:
		score = creature.Ability.Str
	case models.AbilityDEX:
		score = creature.Ability.Dex
	case models.AbilityCON:
		score = creature.Ability.Con
	case models.AbilityINT:
		score = creature.Ability.Int
	case models.AbilityWIS:
		score = creature.Ability.Wiz
	case models.AbilityCHA:
		score = creature.Ability.Cha
	default:
		score = creature.Ability.Int // fallback
	}
	return abilityMod(score)
}

// abilityMod computes D&D 5e ability modifier: floor((score - 10) / 2).
func abilityMod(score int) int {
	return int(math.Floor(float64(score-10) / 2))
}

// ensureSlotInitialized ensures the spell slot level is tracked in runtime state.
// If the key is missing, initializes from the creature template (lazy init).
// Same pattern as resolve_npc_spell.go slot initialization.
func ensureSlotInitialized(resources *models.ResourceState, creature *models.Creature, level int) {
	if resources.SpellSlots == nil {
		resources.SpellSlots = make(map[int]int)
	}
	if _, exists := resources.SpellSlots[level]; !exists {
		if creature.Spellcasting != nil && creature.Spellcasting.SpellSlots != nil {
			resources.SpellSlots[level] = creature.Spellcasting.SpellSlots[level]
		}
	}
}

// ensureInnateInitialized ensures that an innate per-day spell's usage count is
// tracked in AbilityUses. If the key is missing, initializes it to N (the per-day
// limit from the creature template). Without this, the first use would set it to 0,
// limiting N/day spells to exactly 1 use.
func ensureInnateInitialized(resources *models.ResourceState, creature *models.Creature, innateKey string) {
	if innateKey == "" {
		return
	}
	if resources.AbilityUses == nil {
		resources.AbilityUses = make(map[string]int)
	}
	if _, exists := resources.AbilityUses[innateKey]; exists {
		return // already tracked
	}
	// Scan PerDay to find the N for this spell.
	if inn := creature.InnateSpellcasting; inn != nil {
		for perDay, spells := range inn.PerDay {
			for _, s := range spells {
				key := "innate:" + combatai.SpellIDOrName(&s)
				if key == innateKey {
					resources.AbilityUses[innateKey] = perDay
					return
				}
			}
		}
	}
}
