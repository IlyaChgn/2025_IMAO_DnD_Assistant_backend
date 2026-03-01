package usecases

import (
	"fmt"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
)

// resolveNpcAbilityCheck handles the "ability_check" action type for NPC participants.
func resolveNpcAbilityCheck(
	cmd *models.ActionCommand,
	actorName string,
	creature *models.Creature,
) (*models.ActionResponse, error) {
	if cmd.Ability == "" {
		return nil, apperrors.MissingAbilityErr
	}

	ability := strings.ToLower(cmd.Ability)

	var bonus int
	var rollLabel string

	if cmd.Skill != "" {
		bonus = creatureSkillBonus(creature, cmd.Skill, ability)
		rollLabel = fmt.Sprintf("%s (%s) check", cmd.Skill, strings.ToUpper(ability))
	} else {
		bonus = creatureAbilityModifier(creature, ability)
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
		Summary: fmt.Sprintf("%s — %s: %d (natural %d)", actorName, rollLabel, total, natural),
	}, nil
}

// resolveNpcSavingThrow handles the "saving_throw" action type for NPC participants.
func resolveNpcSavingThrow(
	cmd *models.ActionCommand,
	actorName string,
	creature *models.Creature,
) (*models.ActionResponse, error) {
	if cmd.Ability == "" {
		return nil, apperrors.MissingAbilityErr
	}

	ability := strings.ToLower(cmd.Ability)
	bonus := creatureSaveBonus(creature, ability)

	natural, total, rolls := dice.RollD20(bonus, cmd.Advantage, cmd.Disadvantage)

	summary := fmt.Sprintf("%s — %s save: %d (natural %d)",
		actorName, strings.ToUpper(ability), total, natural)

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
