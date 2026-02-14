package usecases

import (
	"fmt"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
)

// resolveCustomRoll handles the "custom_roll" action type.
func resolveCustomRoll(cmd *models.ActionCommand) (*models.ActionResponse, error) {
	if cmd.Dice == "" {
		return nil, apperrors.MissingDiceExprErr
	}

	result, err := dice.Roll(cmd.Dice)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.InvalidDiceExprErr, err.Error())
	}

	totalWithMod := result.Total + cmd.Modifier
	modifier := result.Modifier + cmd.Modifier

	label := cmd.Label
	if label == "" {
		label = "Custom roll"
	}

	expr := cmd.Dice
	if cmd.Modifier > 0 {
		expr = fmt.Sprintf("%s+%d", cmd.Dice, cmd.Modifier)
	} else if cmd.Modifier < 0 {
		expr = fmt.Sprintf("%s%d", cmd.Dice, cmd.Modifier)
	}

	return &models.ActionResponse{
		RollResult: &models.ActionRollResult{
			Expression: expr,
			Rolls:      result.Rolls,
			Modifier:   modifier,
			Total:      totalWithMod,
		},
		Summary: fmt.Sprintf("%s: %d", label, totalWithMod),
	}, nil
}

// resolveAbilityCheck handles the "ability_check" action type.
func resolveAbilityCheck(cmd *models.ActionCommand, charName string, derived *models.DerivedStats) (*models.ActionResponse, error) {
	if cmd.Ability == "" {
		return nil, apperrors.MissingAbilityErr
	}

	ability := strings.ToLower(cmd.Ability)

	var bonus int
	var rollLabel string

	if cmd.Skill != "" {
		skill := strings.ToLower(strings.ReplaceAll(cmd.Skill, " ", "_"))
		if bd, ok := derived.SkillBonuses[skill]; ok {
			bonus = bd.Total
		} else {
			// Unknown skill — fall back to ability modifier
			bonus = derived.AbilityModifiers[ability]
		}
		rollLabel = fmt.Sprintf("%s (%s) check", cmd.Skill, strings.ToUpper(ability))
	} else {
		bonus = derived.AbilityModifiers[ability]
		rollLabel = fmt.Sprintf("%s check", strings.ToUpper(ability))
	}

	natural, total, rolls := dice.RollD20(bonus, cmd.Advantage, cmd.Disadvantage)

	return &models.ActionResponse{
		RollResult: &models.ActionRollResult{
			Expression: fmt.Sprintf("1d20%+d", bonus),
			Rolls:      rolls,
			Modifier:   bonus,
			Total:      total,
			Natural:    natural,
		},
		Summary: fmt.Sprintf("%s — %s: %d (natural %d)", charName, rollLabel, total, natural),
	}, nil
}

// resolveSavingThrow handles the "saving_throw" action type.
func resolveSavingThrow(cmd *models.ActionCommand, charName string, derived *models.DerivedStats) (*models.ActionResponse, error) {
	if cmd.Ability == "" {
		return nil, apperrors.MissingAbilityErr
	}

	ability := strings.ToLower(cmd.Ability)

	var bonus int
	if bd, ok := derived.SaveBonuses[ability]; ok {
		bonus = bd.Total
	} else {
		bonus = derived.AbilityModifiers[ability]
	}

	natural, total, rolls := dice.RollD20(bonus, cmd.Advantage, cmd.Disadvantage)

	summary := fmt.Sprintf("%s — %s save: %d (natural %d)",
		charName, strings.ToUpper(ability), total, natural)

	if cmd.DC > 0 {
		if total >= cmd.DC {
			summary += fmt.Sprintf(" — SUCCESS (DC %d)", cmd.DC)
		} else {
			summary += fmt.Sprintf(" — FAILURE (DC %d)", cmd.DC)
		}
	}

	return &models.ActionResponse{
		RollResult: &models.ActionRollResult{
			Expression: fmt.Sprintf("1d20%+d", bonus),
			Rolls:      rolls,
			Modifier:   bonus,
			Total:      total,
			Natural:    natural,
		},
		Summary: summary,
	}, nil
}
