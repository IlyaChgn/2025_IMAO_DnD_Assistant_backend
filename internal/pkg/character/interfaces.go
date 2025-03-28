package character

import (
	"context"
	"mime/multipart"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type CharacterRepository interface {
	GetCharactersList(ctx context.Context, size, start int, order []models.Order, filter models.CharacterFilterParams,
		search models.SearchParams) ([]*models.CharacterShort, error)
	GetCharacterByMongoId(ctx context.Context, id string) (*models.Character, error)
	AddCharacter(ctx context.Context, rawChar models.CharacterRaw) error
}

type CharacterUsecases interface {
	GetCharactersList(ctx context.Context, size, start int, order []models.Order, filter models.CharacterFilterParams,
		search models.SearchParams) ([]*models.CharacterShort, error)
	GetCharacterByMongoId(ctx context.Context, id string) (*models.Character, error)
	AddCharacter(ctx context.Context, file multipart.File) error
}
