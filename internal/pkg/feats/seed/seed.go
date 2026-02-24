package seed

import (
	"context"
	_ "embed"
	"encoding/json"

	featsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

//go:embed srd_feats.json
var srdFeatsJSON []byte

// SeedFeatDefinitions upserts SRD feat definitions into the database.
// Returns the number of feats successfully upserted.
func SeedFeatDefinitions(ctx context.Context, repo featsinterfaces.FeatsRepository) (int, error) {
	var feats []models.FeatDefinition
	if err := json.Unmarshal(srdFeatsJSON, &feats); err != nil {
		return 0, err
	}

	changed := 0

	for i := range feats {
		feats[i].SchemaVersion = 1

		modified, err := repo.UpsertFeat(ctx, &feats[i])
		if err != nil {
			return changed, err
		}
		if modified {
			changed++
		}
	}

	return changed, nil
}
