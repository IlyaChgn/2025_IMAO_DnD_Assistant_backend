package usecases

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/compute"
)

// TargetStats holds combat-relevant stats for any target (PC or creature).
type TargetStats struct {
	Name            string
	AC              int
	SaveBonuses     map[string]int // "str" -> +5, "dex" -> +2
	Resistances     []string       // "fire", "bludgeoning", ...
	Immunities      []string
	Vulnerabilities []string
}

// loadTargetStats resolves stats for a participant.
// PC targets: load CharacterBase → ComputeDerived.
// Creature targets: load Creature template via bestiaryRepo.
func loadTargetStats(ctx context.Context, uc *actionsUsecases, target *models.ParticipantFull) (*TargetStats, error) {
	if target.IsPlayerCharacter && target.CharacterRuntime != nil {
		charBase, err := uc.characterRepo.GetByID(ctx, target.CharacterRuntime.CharacterID)
		if err != nil {
			return nil, fmt.Errorf("load target character: %w", err)
		}
		if charBase == nil {
			return nil, fmt.Errorf("target character not found: %s", target.CharacterRuntime.CharacterID)
		}

		derived := compute.ComputeDerived(charBase)

		saveBonuses := make(map[string]int, len(derived.SaveBonuses))
		for ability, bb := range derived.SaveBonuses {
			saveBonuses[ability] = bb.Total
		}

		return &TargetStats{
			Name:            charBase.Name,
			AC:              effectiveAC(target, derived.ArmorClass),
			SaveBonuses:     saveBonuses,
			Resistances:     derived.Resistances,
			Immunities:      derived.Immunities,
			Vulnerabilities: derived.Vulnerabilities,
		}, nil
	}

	// Creature target
	if uc.bestiaryRepo == nil {
		return nil, fmt.Errorf("bestiary repository not available")
	}

	creature, err := uc.bestiaryRepo.GetCreatureByID(ctx, target.CreatureID)
	if err != nil {
		return nil, fmt.Errorf("load target creature: %w", err)
	}
	if creature == nil {
		return nil, fmt.Errorf("target creature not found: %s", target.CreatureID)
	}

	saveBonuses := make(map[string]int, 6)
	for _, ability := range []string{"str", "dex", "con", "int", "wis", "cha"} {
		saveBonuses[ability] = creatureSaveBonus(creature, ability)
	}

	name := target.DisplayName
	if name == "" {
		name = creature.Name.Eng
	}

	return &TargetStats{
		Name:            name,
		AC:              effectiveAC(target, creature.ArmorClass),
		SaveBonuses:     saveBonuses,
		Resistances:     creature.DamageResistances,
		Immunities:      creature.DamageImmunities,
		Vulnerabilities: creature.DamageVulnerabilities,
	}, nil
}

// effectiveAC computes AC with StatModifier bonuses applied.
// Scans RuntimeState.StatModifiers for ModTargetAC + ModOpAdd entries (e.g., Shield +5).
func effectiveAC(p *models.ParticipantFull, baseAC int) int {
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

// creatureSaveBonus extracts the integer save bonus from Creature.SavingThrows.
// Falls back to ability modifier if no proficient save is listed.
func creatureSaveBonus(creature *models.Creature, ability string) int {
	// Check if the creature has a proficient saving throw for this ability
	for _, st := range creature.SavingThrows {
		if strings.EqualFold(st.ShortName, ability) || strings.EqualFold(st.Name, ability) {
			// Value is interface{} — handle float64 (JSON) and int32 (BSON)
			switch v := st.Value.(type) {
			case float64:
				return int(v)
			case int32:
				return int(v)
			case int64:
				return int(v)
			case int:
				return v
			}
		}
	}

	// Fallback: compute from ability score
	return creatureAbilityModifier(creature, ability)
}

// creatureAbilityModifier returns floor((score-10)/2) for the given ability.
func creatureAbilityModifier(creature *models.Creature, ability string) int {
	var score int
	switch strings.ToLower(ability) {
	case "str":
		score = creature.Ability.Str
	case "dex":
		score = creature.Ability.Dex
	case "con":
		score = creature.Ability.Con
	case "int":
		score = creature.Ability.Int
	case "wis":
		score = creature.Ability.Wiz // Note: field name is "Wiz" in the model
	case "cha":
		score = creature.Ability.Cha
	default:
		return 0
	}

	return int(math.Floor(float64(score-10) / 2))
}

// intPtr returns a pointer to the given int value.
func intPtr(v int) *int { return &v }

// matchesDamageType checks if a damage type matches an entry in a defense list.
// Handles both exact matches ("fire") and compound entries
// ("bludgeoning, piercing, and slashing from nonmagical attacks").
func matchesDamageType(damageType, entry string) bool {
	entry = strings.ToLower(entry)
	if entry == damageType {
		return true
	}

	return strings.Contains(entry, damageType)
}

// applyResistance adjusts raw damage based on target defenses.
// Returns (finalDamage, modifier) where modifier is "normal"|"resistance"|"vulnerability"|"immunity".
func applyResistance(rawDamage int, damageType string, ts *TargetStats) (int, string) {
	dt := strings.ToLower(damageType)

	// Check immunity first (takes precedence)
	for _, imm := range ts.Immunities {
		if matchesDamageType(dt, imm) {
			return 0, "immunity"
		}
	}

	// Check resistance
	for _, res := range ts.Resistances {
		if matchesDamageType(dt, res) {
			return rawDamage / 2, "resistance" // round down per D&D 5e
		}
	}

	// Check vulnerability
	for _, vuln := range ts.Vulnerabilities {
		if matchesDamageType(dt, vuln) {
			return rawDamage * 2, "vulnerability"
		}
	}

	return rawDamage, "normal"
}
