package usecases

import (
	"context"

	backgroundsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type backgroundsUsecases struct {
	repo backgroundsinterfaces.BackgroundsRepository
}

func NewBackgroundsUsecases(repo backgroundsinterfaces.BackgroundsRepository) backgroundsinterfaces.BackgroundsUsecases {
	return &backgroundsUsecases{repo: repo}
}

func (uc *backgroundsUsecases) GetBackgrounds(ctx context.Context, filter models.BackgroundFilterParams) (*models.BackgroundListResponse, error) {
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

	backgrounds, total, err := uc.repo.GetBackgrounds(ctx, filter)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"filter": filter})
		return nil, err
	}

	if backgrounds == nil {
		backgrounds = []*models.BackgroundDefinition{}
	}

	return &models.BackgroundListResponse{
		Backgrounds: backgrounds,
		Total:       total,
		Page:        filter.Page,
		Size:        filter.Size,
	}, nil
}

func (uc *backgroundsUsecases) GetBackgroundByEngName(ctx context.Context, engName string) (*models.BackgroundDefinition, error) {
	l := logger.FromContext(ctx)

	if engName == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"engName": engName})
		return nil, apperrors.InvalidIDErr
	}

	bg, err := uc.repo.GetBackgroundByEngName(ctx, engName)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"engName": engName})
		return nil, err
	}

	return bg, nil
}
