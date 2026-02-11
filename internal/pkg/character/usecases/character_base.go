package usecases

import (
	"context"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/converter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type characterBaseUsecases struct {
	repo characterinterfaces.CharacterBaseRepository
}

func NewCharacterBaseUsecases(repo characterinterfaces.CharacterBaseRepository) characterinterfaces.CharacterBaseUsecases {
	return &characterBaseUsecases{repo: repo}
}

func (uc *characterBaseUsecases) Create(ctx context.Context, char *models.CharacterBase) error {
	return uc.repo.Create(ctx, char)
}

func (uc *characterBaseUsecases) GetByID(ctx context.Context, id string, userID int) (*models.CharacterBase, error) {
	l := logger.FromContext(ctx)

	if id == "" {
		l.UsecasesWarn(apperrors.InvalidInputError, userID, nil)
		return nil, apperrors.InvalidInputError
	}

	char, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return nil, err
	}

	if char == nil {
		return nil, nil
	}

	// Check ownership
	if char.UserID != strconv.Itoa(userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"id": id})
		return nil, apperrors.PermissionDeniedError
	}

	return char, nil
}

func (uc *characterBaseUsecases) Update(ctx context.Context, char *models.CharacterBase, expectedVersion int, userID int) error {
	l := logger.FromContext(ctx)

	// Verify ownership first
	existing, err := uc.repo.GetByID(ctx, char.ID.Hex())
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": char.ID})
		return err
	}

	if existing == nil {
		return apperrors.FindMongoDataErr
	}

	if existing.UserID != strconv.Itoa(userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"id": char.ID})
		return apperrors.PermissionDeniedError
	}

	return uc.repo.Update(ctx, char, expectedVersion)
}

func (uc *characterBaseUsecases) Delete(ctx context.Context, id string, userID int) error {
	l := logger.FromContext(ctx)

	// Verify ownership first
	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return err
	}

	if existing == nil {
		return apperrors.FindMongoDataErr
	}

	if existing.UserID != strconv.Itoa(userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"id": id})
		return apperrors.PermissionDeniedError
	}

	return uc.repo.Delete(ctx, id)
}

func (uc *characterBaseUsecases) List(ctx context.Context, userID int, page, size int,
	search string) ([]*models.CharacterBase, int64, error) {
	l := logger.FromContext(ctx)

	if page < 0 || size <= 0 {
		l.UsecasesWarn(apperrors.StartPosSizeError, userID, map[string]any{"page": page, "size": size})
		return nil, 0, apperrors.StartPosSizeError
	}

	return uc.repo.List(ctx, strconv.Itoa(userID), page, size, search)
}

func (uc *characterBaseUsecases) ImportLSS(ctx context.Context, fileData []byte,
	userID int) (*models.CharacterBase, *models.ConversionReport, error) {
	l := logger.FromContext(ctx)

	char, report, err := converter.ConvertLSS(fileData, strconv.Itoa(userID))
	if err != nil {
		l.UsecasesError(err, userID, nil)
		return nil, nil, apperrors.ConversionFailedError
	}

	if err := uc.repo.Create(ctx, char); err != nil {
		l.UsecasesError(err, userID, map[string]any{"charName": char.Name})
		return nil, nil, err
	}

	return char, report, nil
}
