package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
)

type encounterUsecases struct {
	repo encounterinterfaces.EncounterRepository
}

func NewEncounterUsecases(repo encounterinterfaces.EncounterRepository) encounterinterfaces.EncounterUsecases {
	return &encounterUsecases{
		repo: repo,
	}
}

// GetEncountersList возвращает список энкаунтеров с пагинацией, фильтрацией и сортировкой
func (uc *encounterUsecases) GetEncountersList(ctx context.Context, size, start int, order []models.Order,
	filter models.EncounterFilterParams, search models.SearchParams) ([]*models.EncounterShort, error) {
	// Валидация параметров пагинации
	if start < 0 || size <= 0 {
		return nil, apperrors.StartPosSizeError
	}

	// Получение списка энкаунтеров из репозитория
	return uc.repo.GetEncountersList(ctx, size, start, order, filter, search)
}

// AddEncounter добавляет новый энкаунтер
func (uc *encounterUsecases) AddEncounter(ctx context.Context, encounter models.EncounterRaw) error {
	// Валидация входных данных
	if encounter.EncounterName == "" {
		return apperrors.InvalidInputError
	}

	// Добавление энкаунтера через репозиторий
	return uc.repo.AddEncounter(ctx, encounter)
}

// GetEncounterByMongoId возвращает энкаунтер по его ID
func (uc *encounterUsecases) GetEncounterByMongoId(ctx context.Context, id string) (*models.Encounter, error) {
	// Валидация ID
	if id == "" {
		return nil, apperrors.InvalidInputError
	}

	// Получение энкаунтера через репозиторий
	return uc.repo.GetEncounterByMongoId(ctx, id)
}
