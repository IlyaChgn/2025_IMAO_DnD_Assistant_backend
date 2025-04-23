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

func (uc *encounterUsecases) GetEncountersList(ctx context.Context, size, start, userID int,
	search *models.SearchParams) (*models.EncountersList, error) {
	if start < 0 || size <= 0 {
		return nil, apperrors.StartPosSizeError
	}

	if search.Value == "" {
		return uc.repo.GetEncountersList(ctx, size, start, userID)
	} else {
		return uc.repo.GetEncountersListWithSearch(ctx, size, start, userID, search)
	}
}

func (uc *encounterUsecases) GetEncounterByID(ctx context.Context, id, userID int) (*models.Encounter, error) {
	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		return nil, apperrors.PermissionDeniedError
	}

	return uc.repo.GetEncounterByID(ctx, id)
}

func (uc *encounterUsecases) SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, userID int) error {
	if encounter.Name == "" || len(encounter.Name) > 60 {
		return apperrors.InvalidInputError
	}

	return uc.repo.SaveEncounter(ctx, encounter, userID)
}

func (uc *encounterUsecases) UpdateEncounter(ctx context.Context, data []byte, id, userID int) error {
	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		return apperrors.PermissionDeniedError
	}

	return uc.repo.UpdateEncounter(ctx, data, id)
}

func (uc *encounterUsecases) RemoveEncounter(ctx context.Context, id, userID int) error {
	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		return apperrors.PermissionDeniedError
	}

	return uc.repo.RemoveEncounter(ctx, id)
}
