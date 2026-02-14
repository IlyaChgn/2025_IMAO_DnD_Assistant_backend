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

// CharacterBaseRepository provides CRUD for CharacterBase documents in characters_v2 collection.
type CharacterBaseRepository interface {
	Create(ctx context.Context, char *models.CharacterBase) error
	GetByID(ctx context.Context, id string) (*models.CharacterBase, error)
	Update(ctx context.Context, char *models.CharacterBase, expectedVersion int) error
	Delete(ctx context.Context, id string, userID string) error
	List(ctx context.Context, userID string, page, size int, search string) ([]*models.CharacterBase, int64, error)
}

// CharacterBaseUsecases provides business logic for CharacterBase operations.
type CharacterBaseUsecases interface {
	Create(ctx context.Context, char *models.CharacterBase) error
	GetByID(ctx context.Context, id string, userID int) (*models.CharacterBase, error)
	GetComputed(ctx context.Context, id string, userID int) (*models.CharacterBase, *models.DerivedStats, error)
	Update(ctx context.Context, char *models.CharacterBase, expectedVersion int, userID int) error
	Delete(ctx context.Context, id string, userID int) error
	List(ctx context.Context, userID int, page, size int, search string) ([]*models.CharacterBase, int64, error)
	ImportLSS(ctx context.Context, fileData []byte, userID int) (*models.CharacterBase, *models.ConversionReport, error)
}
