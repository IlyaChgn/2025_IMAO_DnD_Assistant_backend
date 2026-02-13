package usecases

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

func (uc *inventoryUsecases) GenerateLoot(ctx context.Context, req *models.GenerateLootRequest, userID int) (*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)

	if req.EncounterID == "" {
		l.UsecasesWarn(apperrors.MissingEncounterIDErr, userID, nil)
		return nil, apperrors.MissingEncounterIDErr
	}
	if req.CR < 0 || req.CR > 30 {
		l.UsecasesWarn(apperrors.InvalidCRErr, userID, map[string]any{"cr": req.CR})
		return nil, apperrors.InvalidCRErr
	}

	name := req.Name
	if name == "" {
		name = "Loot"
	}

	tier := crToTier(req.CR)

	items, err := uc.generateLootItems(ctx, tier)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"cr": req.CR})
		return nil, err
	}

	coins := generateCoins(tier)

	container := &models.InventoryContainer{
		EncounterID: req.EncounterID,
		Kind:        models.ContainerKindLoot,
		Name:        name,
		Layout:      models.LayoutTypeList,
		Items:       items,
		Coins:       coins,
	}

	created, err := uc.repo.CreateContainer(ctx, container)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"name": name})
		return nil, err
	}

	return created, nil
}

func (uc *inventoryUsecases) generateLootItems(ctx context.Context, tier crTier) ([]models.ItemInstance, error) {
	countRange := tierItemCounts[tier]
	itemCount := randRange(countRange.min, countRange.max)
	weights := tierRarityWeights[tier]

	// Group picks by rarity to minimize DB queries.
	rarityCounts := make(map[models.ItemRarity]int)
	for range itemCount {
		roll := rand.Intn(100)
		for _, rw := range weights {
			if roll < rw.cumul {
				rarityCounts[rw.rarity]++
				break
			}
		}
	}

	items := make([]models.ItemInstance, 0)
	now := time.Now()

	for rarity, count := range rarityCounts {
		defs, err := uc.itemRepo.GetRandomItemsByRarity(ctx, rarity, count)
		if err != nil {
			return nil, err
		}

		for _, def := range defs {
			item := models.ItemInstance{
				ID:           uuid.New().String(),
				DefinitionID: def.ID.Hex(),
				Quantity:     1,
				Placement:    models.ItemPlacement{Index: len(items)},
				IsIdentified: false,
				AcquiredAt:   now,
				AcquiredFrom: "loot",
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func generateCoins(tier crTier) models.Coins {
	tc := tierCoinTables[tier]
	return models.Coins{
		Cp: randRange(tc.cp.min, tc.cp.max),
		Sp: randRange(tc.sp.min, tc.sp.max),
		Gp: randRange(tc.gp.min, tc.gp.max),
		Pp: randRange(tc.pp.min, tc.pp.max),
	}
}

func randRange(min, max int) int {
	if min >= max {
		return min
	}
	return min + rand.Intn(max-min+1)
}
