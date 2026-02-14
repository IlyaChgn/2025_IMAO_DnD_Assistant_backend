package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
)

// resolveSpellCast handles the "spell_cast" action type.
// TODO(PR2): implement full spell cast resolution with state mutation.
func resolveSpellCast(
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
	if cmd.SpellID == "" {
		return nil, apperrors.MissingSpellIDErr
	}

	return nil, apperrors.InvalidActionTypeErr
}
