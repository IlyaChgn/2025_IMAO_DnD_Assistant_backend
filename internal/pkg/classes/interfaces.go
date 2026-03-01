package classes

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type ClassesRepository interface {
	GetClasses(ctx context.Context, filter models.ClassFilterParams) ([]*models.ClassDefinition, int64, error)
	GetClassByEngName(ctx context.Context, engName string) (*models.ClassDefinition, error)
	EnsureIndexes(ctx context.Context) error
	UpsertClass(ctx context.Context, class *models.ClassDefinition) (bool, error)
}

type ClassesUsecases interface {
	GetClasses(ctx context.Context, filter models.ClassFilterParams) (*models.ClassListResponse, error)
	GetClassByEngName(ctx context.Context, engName string) (*models.ClassDefinition, error)
}
