package usecases

import (
	"context"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	maptilesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles"
)

type mapTilesUsecases struct {
	repo maptilesinterfaces.MapTilesRepository
}

func NewMapTilesUsecases(repo maptilesinterfaces.MapTilesRepository) maptilesinterfaces.MapTilesUsecases {
	return &mapTilesUsecases{repo: repo}
}

func (uc *mapTilesUsecases) GetCategories(ctx context.Context, userID int) ([]*models.MapTileCategory, error) {
	l := logger.FromContext(ctx)
	if userID < 0 {
		l.UsecasesWarn(apperrors.InvalidUserIDError, userID, map[string]any{"userID": userID})
		return nil, apperrors.InvalidUserIDError
	}

	list, err := uc.repo.GetCategories(ctx, userID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"userID": strconv.Itoa(userID)})
		return nil, err
	}

	return list, nil
}
