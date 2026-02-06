package maptiles

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_maptiles.go -package=mocks

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type MapTilesRepository interface {
	// Возвращает список категорий, доступных пользователю
	GetCategories(ctx context.Context, userID int) ([]*models.MapTileCategory, error)
	// Возвращает walkability для конкретного тайла
	GetWalkabilityByTileID(ctx context.Context, tileID string) (*models.TileWalkability, error)
	// Возвращает walkability для всех тайлов сета
	GetWalkabilityBySetID(ctx context.Context, setID string) ([]*models.TileWalkability, error)
	// Создаёт или обновляет walkability для тайла (upsert)
	UpsertWalkability(ctx context.Context, walkability *models.TileWalkability) error
}

type MapTilesUsecases interface {
	GetCategories(ctx context.Context, userID int) ([]*models.MapTileCategory, error)
	GetWalkabilityByTileID(ctx context.Context, tileID string) (*models.TileWalkability, error)
	GetWalkabilityBySetID(ctx context.Context, setID string) ([]*models.TileWalkability, error)
	UpsertWalkability(ctx context.Context, walkability *models.TileWalkability) error
}
