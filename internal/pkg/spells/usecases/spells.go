package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	spellsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells"
)

var validSchools = map[string]bool{
	"abjuration":    true,
	"conjuration":   true,
	"divination":    true,
	"enchantment":   true,
	"evocation":     true,
	"illusion":      true,
	"necromancy":    true,
	"transmutation": true,
}

type spellsUsecases struct {
	repo spellsinterfaces.SpellsRepository
}

func NewSpellsUsecases(repo spellsinterfaces.SpellsRepository) spellsinterfaces.SpellsUsecases {
	return &spellsUsecases{repo: repo}
}

func (uc *spellsUsecases) GetSpells(ctx context.Context, filter models.SpellFilterParams) (*models.SpellListResponse, error) {
	l := logger.FromContext(ctx)

	// Default pagination
	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	if filter.Size > 100 {
		filter.Size = 100
	}

	// Validate level
	if filter.Level != nil {
		if *filter.Level < 0 || *filter.Level > 9 {
			l.UsecasesWarn(apperrors.InvalidSpellLevelErr, 0, map[string]any{"level": *filter.Level})
			return nil, apperrors.InvalidSpellLevelErr
		}
	}

	// Validate school
	if filter.School != "" {
		if !validSchools[filter.School] {
			l.UsecasesWarn(apperrors.InvalidSpellSchoolErr, 0, map[string]any{"school": filter.School})
			return nil, apperrors.InvalidSpellSchoolErr
		}
	}

	spells, total, err := uc.repo.GetSpells(ctx, filter)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"filter": filter})
		return nil, err
	}

	return &models.SpellListResponse{
		Spells: spells,
		Total:  total,
		Page:   filter.Page,
		Size:   filter.Size,
	}, nil
}

func (uc *spellsUsecases) GetSpellByID(ctx context.Context, id string) (*models.SpellDefinition, error) {
	l := logger.FromContext(ctx)

	if id == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"id": id})
		return nil, apperrors.InvalidIDErr
	}

	spell, err := uc.repo.GetSpellByID(ctx, id)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"id": id})
		return nil, err
	}

	return spell, nil
}

func (uc *spellsUsecases) GetSpellsByClass(ctx context.Context, className string, level *int) ([]*models.SpellDefinition, error) {
	l := logger.FromContext(ctx)

	if className == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"className": className})
		return nil, apperrors.InvalidIDErr
	}

	if level != nil {
		if *level < 0 || *level > 9 {
			l.UsecasesWarn(apperrors.InvalidSpellLevelErr, 0, map[string]any{"level": *level})
			return nil, apperrors.InvalidSpellLevelErr
		}
	}

	spells, err := uc.repo.GetSpellsByClass(ctx, className, level)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"className": className})
		return nil, err
	}

	return spells, nil
}
