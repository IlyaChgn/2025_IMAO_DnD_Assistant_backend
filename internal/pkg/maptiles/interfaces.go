package maptiles

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_maptiles.go -package=mocks

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type MapTilesRepository interface {
	// Возвращает список категорий, доступных пользователю
	GetCategories(ctx context.Context, userID int) ([]*models.MapTileCategory, error)
}

type MapTilesUsecases interface {
	GetCategories(ctx context.Context, userID int) ([]*models.MapTileCategory, error)
}
