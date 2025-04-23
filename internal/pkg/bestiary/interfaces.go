package bestiary

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type BestiaryRepository interface {
	GetCreaturesList(ctx context.Context, size, start int, order []models.Order, filter models.FilterParams,
		search models.SearchParams) ([]*models.BestiaryCreature, error)
	GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error)
	AddGeneratedCreature(ctx context.Context, generatedCreature models.Creature) error
}

type BestiaryUsecases interface {
	GetCreaturesList(ctx context.Context, size, start int, order []models.Order, filter models.FilterParams,
		search models.SearchParams) ([]*models.BestiaryCreature, error)
	GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error)
	AddGeneratedCreature(ctx context.Context, creatureInput models.CreatureInput) error
}

type BestiaryS3Manager interface {
	UploadImage(base64Data string, objectName string) (string, error)
}
