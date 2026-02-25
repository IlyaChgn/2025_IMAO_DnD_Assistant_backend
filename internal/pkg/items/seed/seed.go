package seed

import (
	"context"
	"embed"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Item data is split into per-category JSON files for easier maintenance.
// The glob pattern matches: srd_items_equipment.json, srd_items_magic.json,
// srd_items_consumable.json, srd_items_ammo.json, srd_items_utility.json,
// srd_items_reagent.json.
//
//go:embed srd_items_*.json
var itemFiles embed.FS

// combinedItemsJSON holds the merged JSON array of all per-category files.
// Built once at init; panics on malformed data (same guarantee as single-file embed).
var combinedItemsJSON []byte

func init() {
	entries, err := itemFiles.ReadDir(".")
	if err != nil {
		panic("items/seed: failed to read embedded directory: " + err.Error())
	}

	var all []json.RawMessage

	for _, entry := range entries {
		data, err := itemFiles.ReadFile(entry.Name())
		if err != nil {
			panic("items/seed: failed to read " + entry.Name() + ": " + err.Error())
		}

		var batch []json.RawMessage
		if err := json.Unmarshal(data, &batch); err != nil {
			panic("items/seed: failed to parse " + entry.Name() + ": " + err.Error())
		}

		all = append(all, batch...)
	}

	combinedItemsJSON, err = json.Marshal(all)
	if err != nil {
		panic("items/seed: failed to marshal combined items: " + err.Error())
	}
}

// SeedItemDefinitions inserts SRD items into the database, skipping duplicates.
// Returns the number of items successfully inserted.
func SeedItemDefinitions(ctx context.Context, repo itemsinterfaces.ItemDefinitionRepository) (int, error) {
	var items []models.ItemDefinition
	if err := json.Unmarshal(combinedItemsJSON, &items); err != nil {
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
			continue // skip duplicates (engName unique index)
		}
		inserted++
	}

	return inserted, nil
}
