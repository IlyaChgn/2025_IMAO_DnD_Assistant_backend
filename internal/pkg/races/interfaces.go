package races

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type RacesRepository interface {
	GetRaces(ctx context.Context, filter models.RaceFilterParams) ([]*models.RaceDefinition, int64, error)
	GetRaceByEngName(ctx context.Context, engName string) (*models.RaceDefinition, error)
	EnsureIndexes(ctx context.Context) error
	UpsertRace(ctx context.Context, race *models.RaceDefinition) (bool, error)
}

type RacesUsecases interface {
	GetRaces(ctx context.Context, filter models.RaceFilterParams) (*models.RaceListResponse, error)
	GetRaceByEngName(ctx context.Context, engName string) (*models.RaceDefinition, error)
}
