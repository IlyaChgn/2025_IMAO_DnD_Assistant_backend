package usecases

import (
	"context"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mapsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps"
)

type mapsUsecases struct {
	repo mapsinterfaces.MapsRepository
}

func NewMapsUsecases(repo mapsinterfaces.MapsRepository) mapsinterfaces.MapsUsecases {
	return &mapsUsecases{
		repo: repo,
	}
}

func (uc *mapsUsecases) CreateMap(ctx context.Context, userID int, req *models.CreateMapRequest) (*models.MapFull, error) {
	l := logger.FromContext(ctx)

	// Validate request
	validationErrors := ValidateMapRequest(req.Name, &req.Data)
	if len(validationErrors) > 0 {
		l.UsecasesWarn(apperrors.MapValidationError, userID, map[string]any{
			"errors": validationErrors,
		})
		return nil, &ValidationErrorWrapper{Errors: validationErrors}
	}

	// Serialize data to JSON
	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"name": req.Name})
		return nil, apperrors.MapValidationError
	}

	return uc.repo.CreateMap(ctx, userID, req.Name, dataJSON)
}

func (uc *mapsUsecases) GetMapByID(ctx context.Context, userID int, id string) (*models.MapFull, error) {
	l := logger.FromContext(ctx)

	// Check permission first
	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		l.UsecasesWarn(apperrors.MapPermissionDenied, userID, map[string]any{"id": id})
		return nil, apperrors.MapPermissionDenied
	}

	return uc.repo.GetMapByID(ctx, userID, id)
}

func (uc *mapsUsecases) UpdateMap(ctx context.Context, userID int, id string, req *models.UpdateMapRequest) (*models.MapFull, error) {
	l := logger.FromContext(ctx)

	// Check permission first
	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		l.UsecasesWarn(apperrors.MapPermissionDenied, userID, map[string]any{"id": id})
		return nil, apperrors.MapPermissionDenied
	}

	// Validate request
	validationErrors := ValidateMapRequest(req.Name, &req.Data)
	if len(validationErrors) > 0 {
		l.UsecasesWarn(apperrors.MapValidationError, userID, map[string]any{
			"errors": validationErrors,
			"id":     id,
		})
		return nil, &ValidationErrorWrapper{Errors: validationErrors}
	}

	// Serialize data to JSON
	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"name": req.Name, "id": id})
		return nil, apperrors.MapValidationError
	}

	return uc.repo.UpdateMap(ctx, userID, id, req.Name, dataJSON)
}

func (uc *mapsUsecases) DeleteMap(ctx context.Context, userID int, id string) error {
	l := logger.FromContext(ctx)

	// Check permission first
	hasPermission := uc.repo.CheckPermission(ctx, id, userID)
	if !hasPermission {
		l.UsecasesWarn(apperrors.MapPermissionDenied, userID, map[string]any{"id": id})
		return apperrors.MapPermissionDenied
	}

	return uc.repo.DeleteMap(ctx, userID, id)
}

func (uc *mapsUsecases) ListMaps(ctx context.Context, userID int, start, size int) (*models.MapsList, error) {
	l := logger.FromContext(ctx)

	if start < 0 || size <= 0 {
		l.UsecasesWarn(apperrors.StartPosSizeError, userID, map[string]any{"start": start, "size": size})
		return nil, apperrors.StartPosSizeError
	}

	return uc.repo.ListMaps(ctx, userID, start, size)
}

// ValidationErrorWrapper wraps validation errors for type checking
type ValidationErrorWrapper struct {
	Errors []models.ValidationError
}

func (e *ValidationErrorWrapper) Error() string {
	return "validation error"
}
