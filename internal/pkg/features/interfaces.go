package features

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type FeaturesRepository interface {
	GetFeatures(ctx context.Context, filter models.FeatureFilterParams) ([]*models.FeatureDefinition, int64, error)
	GetFeatureByID(ctx context.Context, id string) (*models.FeatureDefinition, error)
	GetFeaturesByClass(ctx context.Context, className string, level *int) ([]*models.FeatureDefinition, error)
	EnsureIndexes(ctx context.Context) error
	UpsertFeature(ctx context.Context, feature *models.FeatureDefinition) error
}

type FeaturesUsecases interface {
	GetFeatures(ctx context.Context, filter models.FeatureFilterParams) (*models.FeatureListResponse, error)
	GetFeatureByID(ctx context.Context, id string) (*models.FeatureDefinition, error)
	GetFeaturesByClass(ctx context.Context, className string, level *int) ([]*models.FeatureDefinition, error)
}
