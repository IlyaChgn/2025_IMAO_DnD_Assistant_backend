package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
)

// resolveUseFeature handles the "use_feature" action type.
// TODO(PR2): implement full feature use resolution with state mutation.
func resolveUseFeature(
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
	if cmd.FeatureID == "" {
		return nil, apperrors.MissingFeatureIDErr
	}

	return nil, apperrors.InvalidActionTypeErr
}
