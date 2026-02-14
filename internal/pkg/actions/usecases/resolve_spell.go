package usecases

import (
	"context"
	"fmt"
	"strconv"
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
	mutated := false
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
		mutated = true
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
		mutated = true
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
		resolveSpellMechanics(ctx, uc, cmd, charBase, derived, spellDef, ed, resp, userID)
	}

	// Apply damage to target if we have damage rolls and a target
	if cmd.TargetID != "" && len(resp.DamageRolls) > 0 {
		// Skip damage application when hit/save wasn't resolved
		skipDamage := false
		if resp.Hit != nil && !*resp.Hit {
			skipDamage = true // Attack missed
		}
		if spellDef != nil && spellDef.Resolution.Type == "attack" && resp.Hit == nil {
			skipDamage = true // Attack spell but couldn't determine hit
		}

		if !skipDamage {
			target, _, tErr := ed.FindParticipantByInstanceID(cmd.TargetID)
			if tErr == nil {
				totalDamage := 0
				for _, dr := range resp.DamageRolls {
					if dr.FinalDamage != nil {
						totalDamage += *dr.FinalDamage
					} else {
						totalDamage += dr.Total
					}
				}

				if totalDamage > 0 {
					applyDamageToTarget(target, totalDamage)

					targetName := target.DisplayName
					if targetName == "" {
						targetName = cmd.TargetID
					}

					resp.StateChanges = append(resp.StateChanges, models.StateChange{
						TargetID:    cmd.TargetID,
						HPDelta:     -totalDamage,
						Description: fmt.Sprintf("%s takes %d damage from %s", targetName, totalDamage, spellRef.Name),
					})
					mutated = true
				}
			}
		}
	}

	// Persist encounter data only if state was mutated
	if mutated {
		if err := persistEncounterData(ctx, uc, ed, encounterID); err != nil {
			l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
			return nil, fmt.Errorf("persist encounter: %w", err)
		}
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

// spellSaveResult tracks the outcome of a target's saving throw.
type spellSaveResult struct {
	saved      bool
	halfDamage bool // true when target saved and spell has onSuccess="half"
	noDamage   bool // true when target saved and spell has onSuccess="none"
}

// resolveSpellMechanics adds roll results based on the spell definition's resolution type.
func resolveSpellMechanics(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	charBase *models.CharacterBase,
	derived *models.DerivedStats,
	spellDef *models.SpellDefinition,
	ed *EncounterData,
	resp *models.ActionResponse,
	userID int,
) {
	l := logger.FromContext(ctx)

	// Load target stats if target is provided
	var ts *TargetStats
	if cmd.TargetID != "" {
		target, _, tErr := ed.FindParticipantByInstanceID(cmd.TargetID)
		if tErr == nil {
			loaded, lErr := loadTargetStats(ctx, uc, target)
			if lErr != nil {
				l.UsecasesWarn(lErr, userID, map[string]any{"targetID": cmd.TargetID})
			} else {
				ts = loaded
			}
		}
	}

	var saveRes *spellSaveResult

	isCrit := false

	switch spellDef.Resolution.Type {
	case "attack":
		resolveSpellAttack(cmd, derived, ts, resp)
		if resp.Hit != nil && !*resp.Hit {
			return // Miss — skip damage
		}
		isCrit = resp.RollResult != nil && resp.RollResult.Natural == 20

	case "save":
		saveRes = resolveSpellSave(derived, spellDef, ts, resp)
		if saveRes != nil && saveRes.noDamage {
			return // Target saved with "no effect"
		}
		// Save-type spell but target stats unavailable — announce DC, skip damage
		if saveRes == nil && spellDef.Resolution.Save != nil {
			return
		}
	}

	// Roll spell damage if defined
	for _, effect := range spellDef.Effects {
		if effect.Damage == nil {
			continue
		}
		dmg := effect.Damage.Base
		diceType := strings.TrimPrefix(dmg.DiceType, "d")
		if dmg.DiceCount > 0 && diceType != "" {
			expr := fmt.Sprintf("%dd%s", dmg.DiceCount, diceType)
			if dmg.Bonus != 0 {
				expr = fmt.Sprintf("%s%+d", expr, dmg.Bonus)
			}
			result, err := dice.Roll(expr)
			if err != nil {
				continue
			}

			rollResult := models.ActionRollResult{
				Expression: expr,
				Rolls:      result.Rolls,
				Modifier:   result.Modifier,
				Total:      result.Total,
				DamageType: dmg.DamageType,
			}

			finalDamage := result.Total

			// Double dice on critical hit (roll extra dice, keep bonus unchanged)
			if isCrit {
				sides, pErr := strconv.Atoi(diceType)
				if pErr == nil {
					critRolls, critTotal := dice.RollDice(dmg.DiceCount, sides)
					rollResult.Rolls = append(rollResult.Rolls, critRolls...)
					rollResult.Total += critTotal
					finalDamage += critTotal
				}
			}

			// Apply half damage for successful save
			if saveRes != nil && saveRes.halfDamage {
				finalDamage = finalDamage / 2
			}

			// Apply resistance/vulnerability/immunity
			if ts != nil && dmg.DamageType != "" {
				adjusted, appliedMod := applyResistance(finalDamage, dmg.DamageType, ts)
				rollResult.AppliedModifier = appliedMod
				rollResult.FinalDamage = intPtr(adjusted)
				finalDamage = adjusted
			} else {
				rollResult.FinalDamage = intPtr(finalDamage)
			}

			resp.DamageRolls = append(resp.DamageRolls, rollResult)
			resp.Summary += fmt.Sprintf(", %d %s damage", finalDamage, dmg.DamageType)
			if rollResult.AppliedModifier != "" && rollResult.AppliedModifier != "normal" {
				resp.Summary += fmt.Sprintf(" (%s)", rollResult.AppliedModifier)
			}
		}
	}
}

// resolveSpellAttack handles attack-type spell resolution.
func resolveSpellAttack(
	cmd *models.ActionCommand,
	derived *models.DerivedStats,
	ts *TargetStats,
	resp *models.ActionResponse,
) {
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

	if ts != nil {
		// D&D 5e: nat 1 always misses, nat 20 always hits
		hit := natural != 1 && (natural == 20 || total >= ts.AC)
		resp.Hit = &hit

		if hit {
			resp.Summary += fmt.Sprintf(", %d to hit vs AC %d — HIT", total, ts.AC)
			if natural == 20 {
				resp.Summary += " (CRITICAL!)"
			}
		} else {
			resp.Summary += fmt.Sprintf(", %d to hit vs AC %d — MISS", total, ts.AC)
		}
	} else {
		resp.Summary += fmt.Sprintf(", %d to hit", total)
	}
}

// resolveSpellSave handles save-type spell resolution.
// Returns save result so the caller can decide on damage.
func resolveSpellSave(
	derived *models.DerivedStats,
	spellDef *models.SpellDefinition,
	ts *TargetStats,
	resp *models.ActionResponse,
) *spellSaveResult {
	if spellDef.Resolution.Save == nil || derived.Spellcasting == nil {
		return nil
	}

	dc := derived.Spellcasting.SpellSaveDC
	ability := strings.ToLower(string(spellDef.Resolution.Save.Ability))

	if ts == nil {
		resp.Summary += fmt.Sprintf(", DC %d %s save",
			dc, strings.ToUpper(ability))
		return nil
	}

	// Roll target's saving throw
	saveBonus := ts.SaveBonuses[ability]
	saveNatural, saveTotal, saveRolls := dice.RollD20(saveBonus, false, false)
	saved := saveTotal >= dc

	resp.Summary += fmt.Sprintf(", DC %d %s save: %s rolls %d (%d%+d)",
		dc, strings.ToUpper(ability), ts.Name, saveTotal, saveNatural, saveBonus)

	onSuccess := spellDef.Resolution.Save.OnSuccess
	result := &spellSaveResult{saved: saved}

	if saved {
		if onSuccess == "half" {
			resp.Summary += " — SAVES (half damage)"
			result.halfDamage = true
		} else {
			resp.Summary += " — SAVES (no effect)"
			result.noDamage = true
		}
	} else {
		resp.Summary += " — FAILS"
	}

	// Add save roll as state change for visibility
	resp.StateChanges = append(resp.StateChanges, models.StateChange{
		Description: fmt.Sprintf("%s %s save: %d (1d20%+d = [%s])",
			ts.Name, strings.ToUpper(ability), saveTotal, saveBonus, formatRolls(saveRolls)),
	})

	return result
}

// formatRolls converts a slice of ints to a comma-separated string.
func formatRolls(rolls []int) string {
	parts := make([]string, len(rolls))
	for i, r := range rolls {
		parts[i] = fmt.Sprintf("%d", r)
	}
	return strings.Join(parts, ", ")
}
