package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	classesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/classes"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type classesUsecases struct {
	repo classesinterfaces.ClassesRepository
}

func NewClassesUsecases(repo classesinterfaces.ClassesRepository) classesinterfaces.ClassesUsecases {
	return &classesUsecases{repo: repo}
}

func (uc *classesUsecases) GetClasses(ctx context.Context, filter models.ClassFilterParams) (*models.ClassListResponse, error) {
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

	classes, total, err := uc.repo.GetClasses(ctx, filter)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"filter": filter})
		return nil, err
	}

	if classes == nil {
		classes = []*models.ClassDefinition{}
	}

	return &models.ClassListResponse{
		Classes: classes,
		Total:   total,
		Page:    filter.Page,
		Size:    filter.Size,
	}, nil
}

func (uc *classesUsecases) GetClassByEngName(ctx context.Context, engName string) (*models.ClassDefinition, error) {
	l := logger.FromContext(ctx)

	if engName == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"engName": engName})
		return nil, apperrors.InvalidIDErr
	}

	class, err := uc.repo.GetClassByEngName(ctx, engName)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"engName": engName})
		return nil, err
	}

	return class, nil
}
