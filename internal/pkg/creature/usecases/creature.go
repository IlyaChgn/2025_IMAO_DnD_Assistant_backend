package usecases

import (
	"context"
	"fmt"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
)

// CreatureUsecases содержит бизнес-логику для работы с существами
type CreatureUsecases struct {
	repo creatureinterfaces.CreatureRepository
}

// NewCreatureUsecases создает новый экземпляр CreatureUsecases
func NewCreatureUsecases(repo creatureinterfaces.CreatureRepository) creatureinterfaces.CreatureUsecases {
	return &CreatureUsecases{repo: repo}
}

// GetCreatureByEngName возвращает существо по полю name.eng
func (uc *CreatureUsecases) GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error) {
	// Вызываем метод репозитория для получения существа
	creature, err := uc.repo.GetCreatureByEngName(ctx, engName)
	if err != nil {
		return nil, fmt.Errorf("failed to get creature by eng name: %w", err)
	}

	// Здесь можно добавить дополнительную бизнес-логику, если нужно
	// Например, проверку прав доступа, преобразование данных и т.д.

	return creature, nil
}
