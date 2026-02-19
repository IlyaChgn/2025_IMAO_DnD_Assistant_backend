package backgrounds

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type BackgroundsRepository interface {
	GetBackgrounds(ctx context.Context, filter models.BackgroundFilterParams) ([]*models.BackgroundDefinition, int64, error)
	GetBackgroundByEngName(ctx context.Context, engName string) (*models.BackgroundDefinition, error)
	EnsureIndexes(ctx context.Context) error
	UpsertBackground(ctx context.Context, bg *models.BackgroundDefinition) error
}

type BackgroundsUsecases interface {
	GetBackgrounds(ctx context.Context, filter models.BackgroundFilterParams) (*models.BackgroundListResponse, error)
	GetBackgroundByEngName(ctx context.Context, engName string) (*models.BackgroundDefinition, error)
}
