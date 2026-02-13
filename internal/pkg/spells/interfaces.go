package spells

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type SpellsRepository interface {
	GetSpells(ctx context.Context, filter models.SpellFilterParams) ([]*models.SpellDefinition, int64, error)
	GetSpellByID(ctx context.Context, id string) (*models.SpellDefinition, error)
	GetSpellsByClass(ctx context.Context, className string, level *int) ([]*models.SpellDefinition, error)
	UpsertSpell(ctx context.Context, spell *models.SpellDefinition) error
	EnsureIndexes(ctx context.Context) error
}

type SpellsUsecases interface {
	GetSpells(ctx context.Context, filter models.SpellFilterParams) (*models.SpellListResponse, error)
	GetSpellByID(ctx context.Context, id string) (*models.SpellDefinition, error)
	GetSpellsByClass(ctx context.Context, className string, level *int) ([]*models.SpellDefinition, error)
}
