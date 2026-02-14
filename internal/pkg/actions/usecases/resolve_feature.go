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
		resolveActiveAction(cmd, derived, feature.ActiveAction, resp)
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
	cmd *models.ActionCommand,
	derived *models.DerivedStats,
	action *models.CharacterActionDef,
	resp *models.ActionResponse,
) {
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
		resp.Summary += fmt.Sprintf(", %d to hit", total)

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
				resp.DamageRolls = append(resp.DamageRolls, models.ActionRollResult{
					Expression: expr,
					Rolls:      result.Rolls,
					Modifier:   result.Modifier,
					Total:      result.Total,
				})
				resp.Summary += fmt.Sprintf(", %d %s damage", result.Total, dmg.DamageType)
			}
		}
	}

	// Saving throw DC
	if action.SavingThrow != nil {
		resp.Summary += fmt.Sprintf(", DC %d %s save", action.SavingThrow.DC, action.SavingThrow.Ability)
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

	_ = derived // available for future use
}
