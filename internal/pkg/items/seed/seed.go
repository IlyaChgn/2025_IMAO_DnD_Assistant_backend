package seed

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

//go:embed srd_items.json
var srdItemsJSON []byte

// SeedItemDefinitions inserts SRD items into the database, skipping duplicates.
// Returns the number of items successfully inserted.
func SeedItemDefinitions(ctx context.Context, repo itemsinterfaces.ItemDefinitionRepository) (int, error) {
	var items []models.ItemDefinition
	if err := json.Unmarshal(srdItemsJSON, &items); err != nil {
		return 0, err
	}

	inserted := 0

	for i := range items {
		items[i].ID = primitive.NilObjectID
		items[i].IsCustom = false
		items[i].CreatedBy = nil
		items[i].SchemaVersion = 1

		_, err := repo.CreateItem(ctx, &items[i])
		if err != nil {
			// Skip duplicates (engName unique index)
			log.Printf("Seed: skipping %s: %v", items[i].EngName, err)
			continue
		}
		inserted++
	}

	return inserted, nil
}
