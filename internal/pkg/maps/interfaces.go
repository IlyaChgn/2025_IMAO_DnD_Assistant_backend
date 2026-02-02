package maps

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_maps.go -package=mocks

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type MapsRepository interface {
	CreateMap(ctx context.Context, userID int, name string, data []byte) (*models.MapFull, error)
	GetMapByID(ctx context.Context, userID int, id string) (*models.MapFull, error)
	UpdateMap(ctx context.Context, userID int, id string, name string, data []byte) (*models.MapFull, error)
	DeleteMap(ctx context.Context, userID int, id string) error
	ListMaps(ctx context.Context, userID int, start, size int) (*models.MapsList, error)
	CheckPermission(ctx context.Context, id string, userID int) bool
}

type MapsUsecases interface {
	CreateMap(ctx context.Context, userID int, req *models.CreateMapRequest) (*models.MapFull, error)
	GetMapByID(ctx context.Context, userID int, id string) (*models.MapFull, error)
	UpdateMap(ctx context.Context, userID int, id string, req *models.UpdateMapRequest) (*models.MapFull, error)
	DeleteMap(ctx context.Context, userID int, id string) error
	ListMaps(ctx context.Context, userID int, start, size int) (*models.MapsList, error)
}
