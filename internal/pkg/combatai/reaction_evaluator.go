package combatai

import (
	"math"
	"math/rand"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// ShieldCandidate holds data for evaluating a Shield reaction.
type ShieldCandidate struct {
	NPC          *models.ParticipantFull
	Creature     *models.Creature
	AttackTotal  int     // the attack roll total
	CurrentAC    int     // NPC's current AC (base + StatModifiers)
	Intelligence float64 // 0.05–1.0
}

// ShouldCastShield returns true if the NPC should use Shield as a reaction.
// Logic: attack hits normally but would miss with +5 AC bonus, and intelligence gate passes.
// Returns false if reaction is already used, no spell slots available, or Shield not found.
func ShouldCastShield(c *ShieldCandidate, rng *rand.Rand) bool {
	if c.NPC.RuntimeState.Resources.ReactionUsed {
		return false
	}
	// D&D 5e PHB p.290: an incapacitated creature cannot take reactions.
	if HasIncapacitatingCondition(c.NPC) {
		return false
	}

	// Attack must hit current AC but miss with Shield (+5).
	if c.AttackTotal < c.CurrentAC {
		return false // already misses, no need for Shield
	}
	if c.AttackTotal >= c.CurrentAC+5 {
		return false // still hits even with Shield, don't waste it
	}

	// Check spell availability.
	spellID, _, _ := FindShieldSpell(c.Creature, &c.NPC.RuntimeState.Resources)
	if spellID == "" {
		return false
	}

	// Intelligence gate.
	return rng.Float64() < c.Intelligence
}

// CounterspellCandidate holds data for evaluating a Counterspell reaction.
type CounterspellCandidate struct {
	NPC          *models.ParticipantFull
	Creature     *models.Creature
	SpellLevel   int     // level of spell being cast (0 = cantrip)
	Distance     int     // distance in feet between NPC and caster
	Intelligence float64 // 0.05–1.0
}

// ShouldCastCounterspell returns true if the NPC should use Counterspell.
// Logic: spell is not a cantrip, NPC is within 60ft, has Counterspell + slot,
// and intelligence gate passes (boosted for high-level spells).
func ShouldCastCounterspell(c *CounterspellCandidate, rng *rand.Rand) bool {
	if c.NPC.RuntimeState.Resources.ReactionUsed {
		return false
	}
	if HasIncapacitatingCondition(c.NPC) {
		return false
	}

	// Can't counterspell cantrips.
	if c.SpellLevel <= 0 {
		return false
	}

	// Must be within 60 feet.
	if c.Distance > 60 {
		return false
	}

	// Check spell availability.
	spellID, _, _ := FindCounterspellSpell(c.Creature, &c.NPC.RuntimeState.Resources)
	if spellID == "" {
		return false
	}

	// Intelligence gate — boosted for high-level spells.
	threshold := c.Intelligence
	if c.SpellLevel >= 5 {
		threshold = math.Min(threshold+0.2, 1.0) // more likely to counter powerful spells
	}
	return rng.Float64() < threshold
}

// ParryCandidate holds data for evaluating a Parry reaction.
type ParryCandidate struct {
	NPC          *models.ParticipantFull
	Creature     *models.Creature
	IncomingDmg  int     // total damage about to be applied
	Intelligence float64 // 0.05–1.0
}

// ShouldParry returns true if the NPC should use Parry reaction.
// Logic: damage is significant (>10% of max HP), NPC has Parry reaction, and intelligence gate passes.
func ShouldParry(c *ParryCandidate, rng *rand.Rand) bool {
	if c.NPC.RuntimeState.Resources.ReactionUsed {
		return false
	}
	if HasIncapacitatingCondition(c.NPC) {
		return false
	}

	// Must have a Parry reaction.
	if FindParryReaction(c.Creature) == nil {
		return false
	}

	// Damage must be significant (>10% of max HP).
	maxHP := c.NPC.RuntimeState.MaxHP
	if maxHP <= 0 {
		maxHP = c.Creature.Hits.Average
	}
	if maxHP > 0 && c.IncomingDmg*10 < maxHP {
		return false // trivial damage, don't waste reaction
	}

	// Intelligence gate.
	return rng.Float64() < c.Intelligence
}

// ParryReduction returns the damage reduction for Parry.
// D&D 5e Parry: reduce melee damage by proficiency bonus + DEX modifier.
func ParryReduction(creature *models.Creature) int {
	profBonus := parseProfBonus(creature.ProficiencyBonus)
	dexMod := int(math.Floor(float64(creature.Ability.Dex-10) / 2))
	reduction := profBonus + dexMod
	if reduction < 0 {
		reduction = 0
	}
	return reduction
}

// FindShieldSpell searches creature's spellcasting for the Shield spell.
// Checks both innate and regular spellcasting for a spell with name/ID matching "shield".
// Returns (spellID, slotLevel, innateKey) or ("", 0, "") if not available.
// innateKey is non-empty for per-day innate casts (e.g. "innate:shield").
func FindShieldSpell(creature *models.Creature, resources *models.ResourceState) (string, int, string) {
	return findReactionSpell(creature, resources, "shield", 1)
}

// FindCounterspellSpell searches creature's spellcasting for the Counterspell spell.
// Returns (spellID, slotLevel, innateKey) or ("", 0, "") if not available.
func FindCounterspellSpell(creature *models.Creature, resources *models.ResourceState) (string, int, string) {
	return findReactionSpell(creature, resources, "counterspell", 3)
}

// findReactionSpell searches for a reaction spell by name and checks slot availability.
// Returns (spellID, slotLevel, innateKey). innateKey is non-empty for per-day innate casts.
func findReactionSpell(creature *models.Creature, resources *models.ResourceState, name string, minLevel int) (string, int, string) {
	// Check innate spellcasting first (at-will or per-day).
	if inn := creature.InnateSpellcasting; inn != nil {
		for _, s := range inn.AtWill {
			if matchesSpellName(&s, name) {
				return SpellIDOrName(&s), 0, "" // at-will, no tracking needed
			}
		}
		for _, spells := range inn.PerDay {
			for _, s := range spells {
				if matchesSpellName(&s, name) {
					// Check if innate uses remain.
					key := "innate:" + SpellIDOrName(&s)
					if resources.AbilityUses != nil {
						remaining, exists := resources.AbilityUses[key]
						if exists && remaining <= 0 {
							continue
						}
					}
					return SpellIDOrName(&s), 0, key // per-day, needs tracking
				}
			}
		}
	}

	// Check regular spellcasting.
	if sc := creature.Spellcasting; sc != nil {
		found := false
		var foundSpell *models.SpellKnown

		// SpellsByLevel map.
		for _, spells := range sc.SpellsByLevel {
			for i := range spells {
				if matchesSpellName(&spells[i], name) {
					found = true
					foundSpell = &spells[i]
					break
				}
			}
			if found {
				break
			}
		}

		// Flat spells list.
		if !found {
			for i := range sc.Spells {
				if matchesSpellName(&sc.Spells[i], name) {
					found = true
					foundSpell = &sc.Spells[i]
					break
				}
			}
		}

		if found {
			// Find lowest available slot at or above minLevel.
			slotLevel := findAvailableSlot(sc, resources, minLevel)
			if slotLevel > 0 {
				return SpellIDOrName(foundSpell), slotLevel, ""
			}
		}
	}

	return "", 0, ""
}

// findAvailableSlot returns the lowest spell slot level >= minLevel that has slots remaining.
// Returns 0 if no slots available.
// Note: ResourceState.SpellSlots stores REMAINING count (not spent).
// If the key is missing, the slot has not been tracked yet (treat as all available).
func findAvailableSlot(sc *models.Spellcasting, resources *models.ResourceState, minLevel int) int {
	for level := minLevel; level <= 9; level++ {
		maxSlots, ok := sc.SpellSlots[level]
		if !ok || maxSlots <= 0 {
			continue
		}
		// Check runtime remaining.
		if resources.SpellSlots != nil {
			if remaining, exists := resources.SpellSlots[level]; exists {
				if remaining <= 0 {
					continue // exhausted
				}
				return level
			}
		}
		// Key not in runtime → slots not yet tracked → all available.
		return level
	}
	return 0
}

// FindParryReaction searches creature's StructuredActions for a reaction-category action
// with "parry" in its name (case-insensitive).
func FindParryReaction(creature *models.Creature) *models.StructuredAction {
	for i := range creature.StructuredActions {
		a := &creature.StructuredActions[i]
		if a.Category != models.ActionCategoryReaction {
			continue
		}
		if containsIgnoreCase(a.Name, "parry") {
			return a
		}
	}
	return nil
}

// EffectiveAC returns the creature's AC including all StatModifier bonuses.
func EffectiveAC(p *models.ParticipantFull, baseAC int) int {
	ac := baseAC
	for _, sm := range p.RuntimeState.StatModifiers {
		for _, eff := range sm.Modifiers {
			if eff.Target == models.ModTargetAC && eff.Operation == models.ModOpAdd {
				ac += eff.Value
			}
		}
	}
	return ac
}

// matchesSpellName checks if a SpellKnown matches the given name (case-insensitive).
// Checks both SpellID and Name fields.
func matchesSpellName(s *models.SpellKnown, name string) bool {
	lower := strings.ToLower(name)
	if s.SpellID != "" && strings.ToLower(s.SpellID) == lower {
		return true
	}
	return strings.ToLower(s.Name) == lower
}

// SpellIDOrName returns SpellID if set, otherwise Name. Always lowercased
// to ensure consistent innate per-day tracking keys across lookups.
func SpellIDOrName(s *models.SpellKnown) string {
	if s.SpellID != "" {
		return strings.ToLower(s.SpellID)
	}
	return strings.ToLower(s.Name)
}

// HasIncapacitatingCondition returns true if the participant has any condition
// that prevents taking reactions per D&D 5e PHB p.290.
// Incapacitating conditions: incapacitated, stunned, paralyzed, petrified, unconscious.
func HasIncapacitatingCondition(p *models.ParticipantFull) bool {
	for _, c := range p.RuntimeState.Conditions {
		switch c.Condition {
		case models.ConditionIncapacitated,
			models.ConditionStunned,
			models.ConditionParalyzed,
			models.ConditionPetrified,
			models.ConditionUnconscious:
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
