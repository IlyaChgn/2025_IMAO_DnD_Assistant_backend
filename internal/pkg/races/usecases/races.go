package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	racesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races"
)

type racesUsecases struct {
	repo racesinterfaces.RacesRepository
}

func NewRacesUsecases(repo racesinterfaces.RacesRepository) racesinterfaces.RacesUsecases {
	return &racesUsecases{repo: repo}
}

func (uc *racesUsecases) GetRaces(ctx context.Context, filter models.RaceFilterParams) (*models.RaceListResponse, error) {
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

	races, total, err := uc.repo.GetRaces(ctx, filter)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"filter": filter})
		return nil, err
	}

	if races == nil {
		races = []*models.RaceDefinition{}
	}

	return &models.RaceListResponse{
		Races: races,
		Total: total,
		Page:  filter.Page,
		Size:  filter.Size,
	}, nil
}

func (uc *racesUsecases) GetRaceByEngName(ctx context.Context, engName string) (*models.RaceDefinition, error) {
	l := logger.FromContext(ctx)

	if engName == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"engName": engName})
		return nil, apperrors.InvalidIDErr
	}

	race, err := uc.repo.GetRaceByEngName(ctx, engName)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"engName": engName})
		return nil, err
	}

	return race, nil
}
