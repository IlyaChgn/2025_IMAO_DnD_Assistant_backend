package feats

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type FeatsRepository interface {
	GetFeats(ctx context.Context, filter models.FeatFilterParams) ([]*models.FeatDefinition, int64, error)
	GetFeatByEngName(ctx context.Context, engName string) (*models.FeatDefinition, error)
	EnsureIndexes(ctx context.Context) error
	UpsertFeat(ctx context.Context, feat *models.FeatDefinition) (bool, error)
}

type FeatsUsecases interface {
	GetFeats(ctx context.Context, filter models.FeatFilterParams) (*models.FeatListResponse, error)
	GetFeatByEngName(ctx context.Context, engName string) (*models.FeatDefinition, error)
}
