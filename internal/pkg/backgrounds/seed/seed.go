package seed

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	backgroundsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds"
)

//go:embed srd_backgrounds.json
var srdBackgroundsJSON []byte

// SeedBackgroundDefinitions upserts SRD background definitions into the database.
// Returns the number of backgrounds successfully upserted.
func SeedBackgroundDefinitions(ctx context.Context, repo backgroundsinterfaces.BackgroundsRepository) (int, error) {
	var backgrounds []models.BackgroundDefinition
	if err := json.Unmarshal(srdBackgroundsJSON, &backgrounds); err != nil {
		return 0, err
	}

	changed := 0

	for i := range backgrounds {
		backgrounds[i].SchemaVersion = 1

		modified, err := repo.UpsertBackground(ctx, &backgrounds[i])
		if err != nil {
			return changed, err
		}
		if modified {
			changed++
		}
	}

	return changed, nil
}
