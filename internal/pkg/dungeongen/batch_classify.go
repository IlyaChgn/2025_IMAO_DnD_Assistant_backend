package dungeongen

import (
	"context"
	"log"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	maptiles "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles"
)

// CategoryThemeTags maps a tile category name to dungeon theme tags.
// A6: category → themeTags mapping.
var CategoryThemeTags = map[string][]string{
	"dungeon":  {"dungeon", "basic"},
	"cave":     {"cave"},
	"fortress": {"fortress"},
}

var defaultThemeTags = []string{"basic"}

// ThemeTagsForCategory returns theme tags for a given category name.
func ThemeTagsForCategory(categoryName string) []string {
	if tags, ok := CategoryThemeTags[categoryName]; ok {
		return tags
	}
	return defaultThemeTags
}

// BatchClassifyTiles reads all tiles from map_tiles + map_tile_walkability,
// classifies each one, computes edge signatures for all rotations,
// and upserts the results into tile_metadata.
// Returns the number of tiles processed.
func BatchClassifyTiles(
	ctx context.Context,
	mapTilesRepo maptiles.MapTilesRepository,
	tileMetaRepo TileMetadataRepository,
) (int, error) {
	// Read all public categories (userID=0 matches "*" wildcard)
	categories, err := mapTilesRepo.GetCategories(ctx, 0)
	if err != nil {
		return 0, err
	}

	roleCounts := make(map[models.TileRole]int)
	processed := 0
	skipped := 0

	for _, cat := range categories {
		themeTags := ThemeTagsForCategory(cat.Name)

		for _, tile := range cat.Tiles {
			walkData, err := mapTilesRepo.GetWalkabilityByTileID(ctx, tile.ID)
			if err != nil {
				log.Printf("BatchClassify: skipping tile %s — no walkability: %v", tile.ID, err)
				skipped++
				continue
			}

			// Classify tile (A4)
			role, openings, ratio := ClassifyTile(walkData.Walkability, walkData.Edges)

			// Compute edge signatures for all 4 rotations (A3)
			sigs := ComputeAllRotationSignatures(walkData.Walkability, walkData.Edges)

			metadata := &models.TileMetadata{
				TileID:         tile.ID,
				Role:           role,
				ThemeTags:      themeTags,
				WalkableRatio:  ratio,
				Openings:       openings,
				EdgeSignatures: sigs,
				AutoClassified: true,
			}

			if err := tileMetaRepo.UpsertTileMetadata(ctx, metadata); err != nil {
				log.Printf("BatchClassify: failed to upsert tile %s: %v", tile.ID, err)
				continue
			}

			roleCounts[role]++
			processed++
		}
	}

	log.Printf("BatchClassify: processed %d tiles, skipped %d", processed, skipped)
	for role, count := range roleCounts {
		log.Printf("  %s: %d", role, count)
	}

	return processed, nil
}
