package creature

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type CreatureRepository interface {
	GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error)
}

type CreatureUsecases interface {
	GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error)
}
