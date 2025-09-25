package usecases

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/google/uuid"

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
	l := logger.FromContext(ctx)

	if start < 0 || size <= 0 {
		l.UsecasesWarn(apperrors.StartPosSizeError, userID, map[string]any{"start": start, "size": size})
		return nil, apperrors.StartPosSizeError
	}

	if search.Value == "" {
		return uc.repo.GetEncountersList(ctx, size, start, userID)
	} else {
		return uc.repo.GetEncountersListWithSearch(ctx, size, start, userID, search)
	}
}

func (uc *encounterUsecases) GetEncounterByID(ctx context.Context, id string, userID int) (*models.Encounter, error) {
	l := logger.FromContext(ctx)

	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"id": id})
		return nil, apperrors.PermissionDeniedError
	}

	return uc.repo.GetEncounterByID(ctx, id)
}

func (uc *encounterUsecases) SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, userID int) error {
	l := logger.FromContext(ctx)

	if encounter.Name == "" || len(encounter.Name) > 60 {
		l.UsecasesWarn(apperrors.InvalidInputError, userID, map[string]any{"name": encounter.Name})
		return apperrors.InvalidInputError
	}

	id := uuid.NewString()

	return uc.repo.SaveEncounter(ctx, encounter, id, userID)
}

func (uc *encounterUsecases) UpdateEncounter(ctx context.Context, data []byte, id string, userID int) error {
	l := logger.FromContext(ctx)

	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"id": id})
		return apperrors.PermissionDeniedError
	}

	return uc.repo.UpdateEncounter(ctx, data, id)
}

func (uc *encounterUsecases) RemoveEncounter(ctx context.Context, id string, userID int) error {
	l := logger.FromContext(ctx)

	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"id": id})
		return apperrors.PermissionDeniedError
	}

	return uc.repo.RemoveEncounter(ctx, id)
}
