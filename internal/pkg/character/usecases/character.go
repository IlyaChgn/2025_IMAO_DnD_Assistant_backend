package usecases

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"

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

func (uc *characterUsecases) GetCharactersList(ctx context.Context, size, start int, order []models.Order,
	filter models.CharacterFilterParams, search models.SearchParams) ([]*models.CharacterShort, error) {
	if start < 0 || size <= 0 {
		return nil, apperrors.StartPosSizeError
	}

	return uc.repo.GetCharactersList(ctx, size, start, order, filter, search)
}

func (uc *characterUsecases) AddCharacter(ctx context.Context, file multipart.File) error {
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return apperrors.ReadFileError
	}

	var rawChar models.CharacterRaw

	err = json.Unmarshal(fileBytes, &rawChar)
	if err != nil {
		return apperrors.InvalidJSONError
	}

	if rawChar.Data == "" {
		return apperrors.InvalidInputError
	}

	return uc.repo.AddCharacter(ctx, rawChar)
}

func (uc *characterUsecases) GetCharacterByMongoId(ctx context.Context, id string) (*models.Character, error) {
	if id == "" {
		return nil, apperrors.InvalidInputError
	}

	return uc.repo.GetCharacterByMongoId(ctx, id)
}
