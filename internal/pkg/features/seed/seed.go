package seed

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	featuresinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features"
)

//go:embed srd_features.json
var srdFeaturesJSON []byte

// SeedFeatureDefinitions upserts SRD feature definitions into the database.
// Returns the number of features successfully upserted.
func SeedFeatureDefinitions(ctx context.Context, repo featuresinterfaces.FeaturesRepository) (int, error) {
	var features []models.FeatureDefinition
	if err := json.Unmarshal(srdFeaturesJSON, &features); err != nil {
		return 0, err
	}

	changed := 0

	for i := range features {
		features[i].SchemaVersion = 1

		modified, err := repo.UpsertFeature(ctx, &features[i])
		if err != nil {
			return changed, err
		}
		if modified {
			changed++
		}
	}

	return changed, nil
}
