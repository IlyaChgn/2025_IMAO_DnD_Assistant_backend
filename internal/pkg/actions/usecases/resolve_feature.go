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

// resolveUseFeature handles the "use_feature" action type.
// Validates resource availability, deducts a use, and resolves the feature's
// active action (attack, save, or healing) if defined.
func resolveUseFeature(
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

	if cmd.FeatureID == "" {
		return nil, apperrors.MissingFeatureIDErr
	}

	// Find feature on character
	feature := findFeature(charBase, cmd.FeatureID)
	if feature == nil {
		return nil, apperrors.FeatureNotFoundErr
	}

	runtime := participant.CharacterRuntime
	if runtime == nil {
		return nil, apperrors.ParticipantNotFoundErr
	}

	var stateChanges []models.StateChange
	mutated := false

	// Check and deduct resource uses if the feature is limited-use
	if feature.Resource != nil && feature.Resource.MaxUses > 0 {
		if runtime.UsedFeatures == nil {
			runtime.UsedFeatures = make(map[string]int)
		}
		used := runtime.UsedFeatures[cmd.FeatureID]
		if used >= feature.Resource.MaxUses {
			return nil, apperrors.FeatureUsesExhaustedErr
		}

		runtime.UsedFeatures[cmd.FeatureID]++
		mutated = true
		stateChanges = append(stateChanges, models.StateChange{
			FeatureUsed: cmd.FeatureID,
			Description: fmt.Sprintf("Used %s (%d/%d uses remaining)",
				feature.Name, feature.Resource.MaxUses-used-1, feature.Resource.MaxUses),
		})
	}

	resp := &models.ActionResponse{
		StateChanges: stateChanges,
		Summary:      fmt.Sprintf("%s uses %s", charBase.Name, feature.Name),
	}

	// Resolve active action if defined
	if feature.ActiveAction != nil {
		resolveActiveAction(ctx, uc, cmd, derived, feature.ActiveAction, ed, resp, userID)
	}

	// Apply damage to target if we have damage rolls and a target
	if cmd.TargetID != "" && len(resp.DamageRolls) > 0 {
		// Skip damage application when hit wasn't resolved
		skipDamage := false
		if resp.Hit != nil && !*resp.Hit {
			skipDamage = true // Attack missed
		}
		if feature.ActiveAction != nil && feature.ActiveAction.Attack != nil && resp.Hit == nil {
			skipDamage = true // Attack feature but couldn't determine hit
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
						Description: fmt.Sprintf("%s takes %d damage from %s", targetName, totalDamage, feature.Name),
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

func findFeature(charBase *models.CharacterBase, featureID string) *models.FeatureInstance {
	for i := range charBase.Features {
		if charBase.Features[i].ID == featureID {
			return &charBase.Features[i]
		}
	}

	return nil
}

// resolveActiveAction resolves a feature's active action (attack roll, save DC, or healing).
func resolveActiveAction(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	derived *models.DerivedStats,
	action *models.CharacterActionDef,
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

	// Attack roll
	if action.Attack != nil {
		attackBonus := action.Attack.Bonus
		natural, total, rolls := dice.RollD20(attackBonus, cmd.Advantage, cmd.Disadvantage)
		resp.RollResult = &models.ActionRollResult{
			Expression: fmt.Sprintf("1d20%+d", attackBonus),
			Rolls:      rolls,
			Modifier:   attackBonus,
			Total:      total,
			Natural:    natural,
		}

		if ts != nil {
			hit := natural != 1 && (natural == 20 || total >= ts.AC)
			resp.Hit = &hit

			if hit {
				resp.Summary += fmt.Sprintf(", %d to hit vs AC %d — HIT", total, ts.AC)
				if natural == 20 {
					resp.Summary += " (CRITICAL!)"
				}
			} else {
				resp.Summary += fmt.Sprintf(", %d to hit vs AC %d — MISS", total, ts.AC)
				return // Miss — skip damage
			}
		} else {
			resp.Summary += fmt.Sprintf(", %d to hit", total)
		}

		// Roll damage
		for _, dmg := range action.Attack.Damage {
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

	// Saving throw DC
	if action.SavingThrow != nil {
		st := action.SavingThrow
		ability := strings.ToLower(string(st.Ability))

		if ts != nil {
			// Roll target's saving throw
			saveBonus := ts.SaveBonuses[ability]
			saveNatural, saveTotal, saveRolls := dice.RollD20(saveBonus, false, false)
			saved := saveTotal >= st.DC

			resp.Summary += fmt.Sprintf(", DC %d %s save: %s rolls %d (%d%+d)",
				st.DC, strings.ToUpper(ability), ts.Name, saveTotal, saveNatural, saveBonus)

			// Add save roll as state change (before potential early return)
			resp.StateChanges = append(resp.StateChanges, models.StateChange{
				Description: fmt.Sprintf("%s %s save: %d (1d20%+d = [%s])",
					ts.Name, strings.ToUpper(ability), saveTotal, saveBonus, formatRolls(saveRolls)),
			})

			if saved {
				onSuccess := st.OnSuccess
				if onSuccess == "half damage" || onSuccess == "half" {
					resp.Summary += " — SAVES (half damage)"
				} else {
					resp.Summary += " — SAVES (no effect)"
					return // No damage
				}
			} else {
				resp.Summary += " — FAILS"
			}

			// Roll damage for save-based features
			if st.Damage != nil {
				for _, dmg := range st.Damage {
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

						// Apply half damage on successful save
						if saved {
							finalDamage = finalDamage / 2
						}

						// Apply resistance/vulnerability/immunity
						if dmg.DamageType != "" {
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
		} else {
			resp.Summary += fmt.Sprintf(", DC %d %s save", st.DC, st.Ability)
		}
	}

	// Healing
	if action.Healing != nil {
		h := action.Healing
		diceType := strings.TrimPrefix(h.DiceType, "d")
		if h.DiceCount > 0 && diceType != "" {
			expr := fmt.Sprintf("%dd%s", h.DiceCount, diceType)
			if h.Bonus != 0 {
				expr = fmt.Sprintf("%s%+d", expr, h.Bonus)
			}
			result, err := dice.Roll(expr)
			if err == nil {
				resp.DamageRolls = append(resp.DamageRolls, models.ActionRollResult{
					Expression: expr,
					Rolls:      result.Rolls,
					Modifier:   result.Modifier,
					Total:      result.Total,
				})
				resp.Summary += fmt.Sprintf(", %d healing", result.Total)
			}
		}
	}

}
