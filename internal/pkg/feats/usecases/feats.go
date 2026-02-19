package usecases

import (
	"context"

	featsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type featsUsecases struct {
	repo featsinterfaces.FeatsRepository
}

func NewFeatsUsecases(repo featsinterfaces.FeatsRepository) featsinterfaces.FeatsUsecases {
	return &featsUsecases{repo: repo}
}

func (uc *featsUsecases) GetFeats(ctx context.Context, filter models.FeatFilterParams) (*models.FeatListResponse, error) {
	l := logger.FromContext(ctx)

	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	if filter.Size > 100 {
		filter.Size = 100
	}

	feats, total, err := uc.repo.GetFeats(ctx, filter)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"filter": filter})
		return nil, err
	}

	if feats == nil {
		feats = []*models.FeatDefinition{}
	}

	return &models.FeatListResponse{
		Feats: feats,
		Total: total,
		Page:  filter.Page,
		Size:  filter.Size,
	}, nil
}

func (uc *featsUsecases) GetFeatByEngName(ctx context.Context, engName string) (*models.FeatDefinition, error) {
	l := logger.FromContext(ctx)

	if engName == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"engName": engName})
		return nil, apperrors.InvalidIDErr
	}

	feat, err := uc.repo.GetFeatByEngName(ctx, engName)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"engName": engName})
		return nil, err
	}

	return feat, nil
}
