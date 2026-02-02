package character

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_character.go -package=mocks

import (
	"context"
	"mime/multipart"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type CharacterRepository interface {
	GetCharactersList(ctx context.Context, size, start, userID int,
		search models.SearchParams) ([]*models.CharacterShort, error)
	GetCharacterByMongoId(ctx context.Context, id string) (*models.Character, error)
	AddCharacter(ctx context.Context, rawChar models.CharacterRaw, userID int) error
}

type CharacterUsecases interface {
	GetCharactersList(ctx context.Context, size, start, userID int,
		search models.SearchParams) ([]*models.CharacterShort, error)
	GetCharacterByMongoId(ctx context.Context, id string, userID int) (*models.Character, error)
	AddCharacter(ctx context.Context, file multipart.File, userID int) error
}
