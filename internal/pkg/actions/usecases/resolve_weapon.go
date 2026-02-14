package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
)

// resolveWeaponAttack handles the "weapon_attack" action type.
// TODO(PR2): implement full weapon attack resolution with state mutation.
func resolveWeaponAttack(
	_ context.Context,
	_ *actionsUsecases,
	cmd *models.ActionCommand,
	_ string,
	_ *models.CharacterBase,
	_ *models.DerivedStats,
	_ *models.ParticipantFull,
	_ *EncounterData,
	_ int,
) (*models.ActionResponse, error) {
	if cmd.WeaponID == "" {
		return nil, apperrors.MissingWeaponIDErr
	}

	return nil, apperrors.InvalidActionTypeErr
}
