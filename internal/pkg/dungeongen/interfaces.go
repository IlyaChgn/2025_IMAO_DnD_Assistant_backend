package dungeongen

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// TileMetadataRepository manages pre-computed tile classification data.
type TileMetadataRepository interface {
	EnsureIndexes(ctx context.Context) error
	UpsertTileMetadata(ctx context.Context, metadata *models.TileMetadata) error
	GetByRole(ctx context.Context, role models.TileRole) ([]*models.TileMetadata, error)
	GetByRoleAndTags(ctx context.Context, role models.TileRole, tags []string) ([]*models.TileMetadata, error)
	GetAll(ctx context.Context) ([]*models.TileMetadata, error)
}

// GenerateRequest is the HTTP request body for dungeon generation.
type GenerateRequest struct {
	Seed       int64  `json:"seed"`
	Size       string `json:"size"`
	PartyLevel int    `json:"partyLevel"`
	PartySize  int    `json:"partySize"`
	Difficulty string `json:"difficulty"`
	Theme      string `json:"theme"`
}

// DungeonGenUsecases is the business logic contract for dungeon generation.
type DungeonGenUsecases interface {
	GenerateDungeon(ctx context.Context, req *GenerateRequest) (*DungeonResponse, error)
}
