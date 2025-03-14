package usecases

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
)

type creatureUsecases struct {
	repo creatureinterfaces.CreatureRepository
}

func NewCreatureUsecases(repo creatureinterfaces.CreatureRepository) creatureinterfaces.CreatureUsecases {
	return &creatureUsecases{
		repo: repo,
	}
}

func (uc *creatureUsecases) GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error) {
	return uc.repo.GetCreatureByEngName(ctx, engName)
}
