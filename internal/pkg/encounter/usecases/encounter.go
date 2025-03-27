package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
)

type encounterUsecases struct {
	repo encounterinterfaces.EncounterRepository
}

func NewEncounterUsecases(repo encounterinterfaces.EncounterRepository) encounterinterfaces.EncounterUsecases {
	return &encounterUsecases{
		repo: repo,
	}
}

func (uc *encounterUsecases) GetEncountersList(ctx context.Context, size, start int, order []models.Order,
	filter models.EncounterFilterParams, search models.SearchParams) ([]*models.EncounterShort, error) {
	if start < 0 || size <= 0 {
		return nil, apperrors.StartPosSizeError
	}

	return uc.repo.GetEncountersList(ctx, size, start, order, filter, search)
}

func (uc *encounterUsecases) AddEncounter(ctx context.Context, encounter models.EncounterRaw) error {
	if encounter.EncounterName == "" {
		return apperrors.InvalidInputError
	}

	return uc.repo.AddEncounter(ctx, encounter)
}

func (uc *encounterUsecases) GetEncounterByMongoId(ctx context.Context, id string) (*models.Encounter, error) {
	if id == "" {
		return nil, apperrors.InvalidInputError
	}

	return uc.repo.GetEncounterByMongoId(ctx, id)
}
