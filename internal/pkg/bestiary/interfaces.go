package bestiary

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type BestiaryRepository interface {
	GetCreaturesList(ctx context.Context, size, start int, order []models.Order, filter models.FilterParams,
		search models.SearchParams) ([]*models.BestiaryCreature, error)
}

type BestiaryUsecases interface {
	GetCreaturesList(ctx context.Context, size, start int, order []models.Order, filter models.FilterParams,
		search models.SearchParams) ([]*models.BestiaryCreature, error)
}
