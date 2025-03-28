package encounter

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type EncounterRepository interface {
	GetEncountersList(ctx context.Context, size, start int, order []models.Order, filter models.EncounterFilterParams,
		search models.SearchParams) ([]*models.EncounterShort, error)
	GetEncounterByMongoId(ctx context.Context, id string) (*models.Encounter, error)
	AddEncounter(ctx context.Context, encounter models.EncounterRaw) error
}

type EncounterUsecases interface {
	GetEncountersList(ctx context.Context, size, start int, order []models.Order, filter models.EncounterFilterParams,
		search models.SearchParams) ([]*models.EncounterShort, error)
	GetEncounterByMongoId(ctx context.Context, id string) (*models.Encounter, error)
	AddEncounter(ctx context.Context, encounter models.EncounterRaw) error
}
