package bestiary

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type BestiaryRepository interface {
	GetCreaturesList(ctx context.Context, size, start int, order []models.Order, filter models.FilterParams,
		search models.SearchParams, searchInSecondCollection bool) ([]*models.BestiaryCreature, error)
	GetCreatureByEngName(ctx context.Context, engName string, searchInSecondCollection bool) (*models.Creature, error)
	AddGeneratedCreature(ctx context.Context, generatedCreature models.Creature) error
}

type BestiaryUsecases interface {
	GetCreaturesList(ctx context.Context, size, start int, order []models.Order, filter models.FilterParams,
		search models.SearchParams) ([]*models.BestiaryCreature, error)
	GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error)
	AddGeneratedCreature(ctx context.Context, creatureInput models.CreatureInput) error
	ParseCreatureFromImage(ctx context.Context, image []byte) (*models.Creature, error)
	GenerateCreatureFromDescription(ctx context.Context, description string) (*models.Creature, error)
}

type BestiaryS3Manager interface {
	UploadImage(base64Data string, objectName string) (string, error)
}

type GeminiAPI interface {
	GenerateFromImage(image []byte) (map[string]interface{}, error)
	GenerateFromDescription(desc string) (map[string]interface{}, error)
}

type LLMJobRepository interface {
	Create(ctx context.Context, job *models.LLMJob) error
	Get(ctx context.Context, id string) (*models.LLMJob, error)
	Update(ctx context.Context, job *models.LLMJob) error
}

type GenerationUsecases interface {
	SubmitText(ctx context.Context, desc string) (string, error)
	SubmitImage(ctx context.Context, img []byte) (string, error)
	GetJob(ctx context.Context, id string) (*models.LLMJob, error)
}

type GeneratedCreatureProcessorUsecases interface {
	ValidateAndProcessGeneratedCreature(*models.Creature) (*models.Creature, error)
}
