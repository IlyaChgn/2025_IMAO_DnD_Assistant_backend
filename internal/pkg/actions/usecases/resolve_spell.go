package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

// resolveSpellCast handles the "spell_cast" action type.
// Validates spell knowledge, deducts a spell slot, handles concentration,
// and resolves the spell's mechanical effects.
func resolveSpellCast(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	encounterID string,
	charBase *models.CharacterBase,
	derived *models.DerivedStats,
	participant *models.ParticipantFull,
	ed *EncounterData,
	userID int,
) (*models.ActionResponse, error) {
	l := logger.FromContext(ctx)

	if cmd.SpellID == "" {
		return nil, apperrors.MissingSpellIDErr
	}

	// Verify the character has spellcasting
	if charBase.Spellcasting == nil {
		return nil, apperrors.SpellNotKnownErr
	}

	// Find spell in character's known/prepared spells
	spellRef := findSpellRef(charBase, cmd.SpellID)
	if spellRef == nil {
		return nil, apperrors.SpellNotKnownErr
	}

	// Load spell definition (if spells repo is available)
	var spellDef *models.SpellDefinition
	if uc.spellsRepo != nil {
		var err error
		spellDef, err = uc.spellsRepo.GetSpellByID(ctx, cmd.SpellID)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{"spellID": cmd.SpellID})
			// Continue without definition — we can still deduct slots
		}
	}

	runtime := participant.CharacterRuntime
	if runtime == nil {
		return nil, apperrors.ParticipantNotFoundErr
	}

	var stateChanges []models.StateChange
	slotLevel := cmd.SlotLevel

	// Cantrips (level 0) don't require slots
	if slotLevel > 0 {
		// Validate slot availability
		if derived.Spellcasting == nil {
			return nil, apperrors.InsufficientSlotsErr
		}

		maxSlots := derived.Spellcasting.MaxSpellSlots[slotLevel]
		if runtime.SpentSpellSlots == nil {
			runtime.SpentSpellSlots = make(map[int]int)
		}
		spent := runtime.SpentSpellSlots[slotLevel]

		if spent >= maxSlots {
			return nil, apperrors.InsufficientSlotsErr
		}

		// Deduct spell slot
		runtime.SpentSpellSlots[slotLevel]++
		stateChanges = append(stateChanges, models.StateChange{
			SlotSpent:   slotLevel,
			Description: fmt.Sprintf("Spent a level %d spell slot (%d/%d remaining)", slotLevel, maxSlots-spent-1, maxSlots),
		})
	}

	// Handle concentration
	if spellDef != nil && spellDef.Concentration {
		// Clear existing concentration
		if runtime.Concentration != nil {
			stateChanges = append(stateChanges, models.StateChange{
				Description: fmt.Sprintf("Dropped concentration on %s", runtime.Concentration.SpellName),
			})
		}

		runtime.Concentration = &models.CharacterConcentration{
			SpellID:   cmd.SpellID,
			SpellName: spellRef.Name,
		}
		stateChanges = append(stateChanges, models.StateChange{
			Description: fmt.Sprintf("Concentrating on %s", spellRef.Name),
		})
	}

	// Resolve spell mechanics
	resp := &models.ActionResponse{
		StateChanges: stateChanges,
		Summary:      fmt.Sprintf("%s casts %s", charBase.Name, spellRef.Name),
	}

	if slotLevel > 0 {
		resp.Summary += fmt.Sprintf(" (level %d slot)", slotLevel)
	}

	if spellDef != nil {
		resolveSpellMechanics(cmd, charBase, derived, spellDef, resp)
	}

	// Persist encounter data
	if err := persistEncounterData(ctx, uc, ed, encounterID); err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, fmt.Errorf("persist encounter: %w", err)
	}

	return resp, nil
}

func findSpellRef(charBase *models.CharacterBase, spellID string) *models.SpellRef {
	sc := charBase.Spellcasting

	for i := range sc.CantripsKnown {
		if sc.CantripsKnown[i].SpellID == spellID {
			return &sc.CantripsKnown[i]
		}
	}
	for i := range sc.SpellsKnown {
		if sc.SpellsKnown[i].SpellID == spellID {
			return &sc.SpellsKnown[i]
		}
	}
	for i := range sc.PreparedSpells {
		if sc.PreparedSpells[i].SpellID == spellID {
			return &sc.PreparedSpells[i]
		}
	}
	for i := range sc.AlwaysPrepared {
		if sc.AlwaysPrepared[i].SpellID == spellID {
			return &sc.AlwaysPrepared[i]
		}
	}
	for i := range sc.Spellbook {
		if sc.Spellbook[i].SpellID == spellID {
			return &sc.Spellbook[i]
		}
	}

	return nil
}

// resolveSpellMechanics adds roll results based on the spell definition's resolution type.
func resolveSpellMechanics(
	cmd *models.ActionCommand,
	charBase *models.CharacterBase,
	derived *models.DerivedStats,
	spellDef *models.SpellDefinition,
	resp *models.ActionResponse,
) {
	switch spellDef.Resolution.Type {
	case "attack":
		if derived.Spellcasting == nil {
			return
		}
		attackBonus := derived.Spellcasting.SpellAttackBonus
		natural, total, rolls := dice.RollD20(attackBonus, cmd.Advantage, cmd.Disadvantage)
		resp.RollResult = &models.ActionRollResult{
			Expression: fmt.Sprintf("1d20%+d", attackBonus),
			Rolls:      rolls,
			Modifier:   attackBonus,
			Total:      total,
			Natural:    natural,
		}
		resp.Summary += fmt.Sprintf(", %d to hit", total)

	case "save":
		if spellDef.Resolution.Save != nil && derived.Spellcasting != nil {
			dc := derived.Spellcasting.SpellSaveDC
			resp.Summary += fmt.Sprintf(", DC %d %s save",
				dc, strings.ToUpper(string(spellDef.Resolution.Save.Ability)))
		}
	}

	// Roll spell damage if defined
	if len(spellDef.Effects) > 0 {
		for _, effect := range spellDef.Effects {
			if effect.Damage == nil {
				continue
			}
			dmg := effect.Damage.Base
			if dmg.DiceCount > 0 && dmg.DiceType != "" {
				expr := fmt.Sprintf("%dd%s", dmg.DiceCount, dmg.DiceType)
				if dmg.Bonus != 0 {
					expr = fmt.Sprintf("%s%+d", expr, dmg.Bonus)
				}
				result, err := dice.Roll(expr)
				if err != nil {
					continue
				}
				resp.DamageRolls = append(resp.DamageRolls, models.ActionRollResult{
					Expression: expr,
					Rolls:      result.Rolls,
					Modifier:   result.Modifier + dmg.Bonus,
					Total:      result.Total,
				})
				resp.Summary += fmt.Sprintf(", %d %s damage", result.Total, dmg.DamageType)
			}
		}
	}

	_ = charBase // available for future use (cantrip scaling by caster level)
}
