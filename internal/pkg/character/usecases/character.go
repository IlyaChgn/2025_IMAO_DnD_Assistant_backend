package usecases

import (
	"context"
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"io"
	"mime/multipart"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
)

type characterUsecases struct {
	repo characterinterfaces.CharacterRepository
}

func NewCharacterUsecases(repo characterinterfaces.CharacterRepository) characterinterfaces.CharacterUsecases {
	return &characterUsecases{
		repo: repo,
	}
}

func (uc *characterUsecases) GetCharactersList(ctx context.Context, size, start, userID int,
	search models.SearchParams) ([]*models.CharacterShort, error) {
	l := logger.FromContext(ctx)

	if start < 0 || size <= 0 {
		l.UsecasesWarn(apperrors.StartPosSizeError, userID, map[string]any{"start": start, "size": size})
		return nil, apperrors.StartPosSizeError
	}

	return uc.repo.GetCharactersList(ctx, size, start, userID, search)
}

func (uc *characterUsecases) AddCharacter(ctx context.Context, file multipart.File, userID int) error {
	l := logger.FromContext(ctx)

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		l.UsecasesError(err, userID, nil)
		return apperrors.ReadFileError
	}

	var rawChar models.CharacterRaw

	err = json.Unmarshal(fileBytes, &rawChar)
	if err != nil {
		l.UsecasesError(err, userID, nil)
		return apperrors.InvalidJSONError
	}

	if rawChar.Data == "" {
		l.UsecasesWarn(apperrors.InvalidInputError, userID, nil)
		return apperrors.InvalidInputError
	}

	return uc.repo.AddCharacter(ctx, rawChar, userID)
}

func (uc *characterUsecases) GetCharacterByMongoId(ctx context.Context, id string,
	userID int) (*models.Character, error) {
	l := logger.FromContext(ctx)

	if id == "" {
		l.UsecasesWarn(apperrors.InvalidInputError, userID, nil)
		return nil, apperrors.InvalidInputError
	}

	character, err := uc.repo.GetCharacterByMongoId(ctx, id)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return nil, err
	}

	if character.UserID != "*" && character.UserID != strconv.Itoa(userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"id": id})
		return nil, apperrors.PermissionDeniedError
	}

	return character, nil
}
