package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	featuresinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

var validSources = map[string]bool{
	"class":      true,
	"race":       true,
	"feat":       true,
	"background": true,
}

type featuresUsecases struct {
	repo featuresinterfaces.FeaturesRepository
}

func NewFeaturesUsecases(repo featuresinterfaces.FeaturesRepository) featuresinterfaces.FeaturesUsecases {
	return &featuresUsecases{repo: repo}
}

func (uc *featuresUsecases) GetFeatures(ctx context.Context, filter models.FeatureFilterParams) (*models.FeatureListResponse, error) {
	l := logger.FromContext(ctx)

	// Default pagination
	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	if filter.Size > 100 {
		filter.Size = 100
	}

	// Validate source
	if filter.Source != "" {
		if !validSources[filter.Source] {
			l.UsecasesWarn(apperrors.InvalidFeatureSourceErr, 0, map[string]any{"source": filter.Source})
			return nil, apperrors.InvalidFeatureSourceErr
		}
	}

	// Validate level
	if filter.Level != nil {
		if *filter.Level < 0 || *filter.Level > 20 {
			l.UsecasesWarn(apperrors.InvalidFeatureLevelErr, 0, map[string]any{"level": *filter.Level})
			return nil, apperrors.InvalidFeatureLevelErr
		}
	}

	features, total, err := uc.repo.GetFeatures(ctx, filter)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"filter": filter})
		return nil, err
	}

	if features == nil {
		features = []*models.FeatureDefinition{}
	}

	return &models.FeatureListResponse{
		Features: features,
		Total:    total,
		Page:     filter.Page,
		Size:     filter.Size,
	}, nil
}

func (uc *featuresUsecases) GetFeatureByID(ctx context.Context, id string) (*models.FeatureDefinition, error) {
	l := logger.FromContext(ctx)

	if id == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"id": id})
		return nil, apperrors.InvalidIDErr
	}

	feature, err := uc.repo.GetFeatureByID(ctx, id)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"id": id})
		return nil, err
	}

	return feature, nil
}

func (uc *featuresUsecases) GetFeaturesByClass(ctx context.Context, className string, level *int) ([]*models.FeatureDefinition, error) {
	l := logger.FromContext(ctx)

	if className == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"className": className})
		return nil, apperrors.InvalidIDErr
	}

	if level != nil {
		if *level < 0 || *level > 20 {
			l.UsecasesWarn(apperrors.InvalidFeatureLevelErr, 0, map[string]any{"level": *level})
			return nil, apperrors.InvalidFeatureLevelErr
		}
	}

	features, err := uc.repo.GetFeaturesByClass(ctx, className, level)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"className": className})
		return nil, err
	}

	if features == nil {
		features = []*models.FeatureDefinition{}
	}

	return features, nil
}
