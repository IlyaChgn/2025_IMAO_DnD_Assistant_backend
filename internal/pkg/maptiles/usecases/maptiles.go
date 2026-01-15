package usecases

import (
	"context"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
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
	if userID < 0 { // примитивная валидация (по аналогии со стилем)
		l.UsecasesWarn(nil, userID, map[string]any{"userID": userID})
		// здесь особой бизнес-ошибки нет, просто логируем; можно вернуть пусто
	}

	list, err := uc.repo.GetCategories(ctx, userID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"userID": strconv.Itoa(userID)})
		return nil, err
	}

	return list, nil
}
