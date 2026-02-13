package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

var validCategories = map[models.ItemCategory]bool{
	models.ItemCategoryEquipment:  true,
	models.ItemCategoryConsumable: true,
	models.ItemCategoryAmmo:       true,
	models.ItemCategoryUtility:    true,
	models.ItemCategoryQuest:      true,
}

var validRarities = map[models.ItemRarity]bool{
	models.ItemRarityCommon:    true,
	models.ItemRarityUncommon:  true,
	models.ItemRarityRare:      true,
	models.ItemRarityVeryRare:  true,
	models.ItemRarityLegendary: true,
	models.ItemRarityArtifact:  true,
}

type itemUsecases struct {
	repo itemsinterfaces.ItemDefinitionRepository
}

func NewItemUsecases(repo itemsinterfaces.ItemDefinitionRepository) itemsinterfaces.ItemUsecases {
	return &itemUsecases{repo: repo}
}

func (uc *itemUsecases) GetItems(ctx context.Context, filter models.ItemFilterParams) (*models.ItemListResponse, error) {
	l := logger.FromContext(ctx)

	if filter.Page < 0 {
		filter.Page = 0
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	if filter.Size > 100 {
		filter.Size = 100
	}

	if filter.Category != "" && !validCategories[filter.Category] {
		l.UsecasesWarn(apperrors.InvalidItemCategoryErr, 0, map[string]any{"category": filter.Category})
		return nil, apperrors.InvalidItemCategoryErr
	}
	if filter.Rarity != "" && !validRarities[filter.Rarity] {
		l.UsecasesWarn(apperrors.InvalidItemRarityErr, 0, map[string]any{"rarity": filter.Rarity})
		return nil, apperrors.InvalidItemRarityErr
	}

	items, total, err := uc.repo.GetItems(ctx, filter)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"filter": filter})
		return nil, err
	}

	if items == nil {
		items = []*models.ItemDefinition{}
	}

	return &models.ItemListResponse{
		Items: items,
		Total: total,
	}, nil
}

func (uc *itemUsecases) GetItemByEngName(ctx context.Context, engName string) (*models.ItemDefinition, error) {
	l := logger.FromContext(ctx)

	if engName == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"engName": engName})
		return nil, apperrors.InvalidIDErr
	}

	item, err := uc.repo.GetItemByEngName(ctx, engName)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"engName": engName})
		return nil, err
	}

	return item, nil
}

func (uc *itemUsecases) CreateItem(ctx context.Context, item *models.ItemDefinition, userID int) (*models.ItemDefinition, error) {
	l := logger.FromContext(ctx)

	item.IsCustom = true
	item.CreatedBy = &userID
	item.SchemaVersion = 1

	if item.Category != "" && !validCategories[item.Category] {
		l.UsecasesWarn(apperrors.InvalidItemCategoryErr, userID, map[string]any{"category": item.Category})
		return nil, apperrors.InvalidItemCategoryErr
	}
	if item.Rarity != "" && !validRarities[item.Rarity] {
		l.UsecasesWarn(apperrors.InvalidItemRarityErr, userID, map[string]any{"rarity": item.Rarity})
		return nil, apperrors.InvalidItemRarityErr
	}

	created, err := uc.repo.CreateItem(ctx, item)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"engName": item.EngName})
		return nil, err
	}

	return created, nil
}

func (uc *itemUsecases) UpdateItem(ctx context.Context, item *models.ItemDefinition, userID int) (*models.ItemDefinition, error) {
	l := logger.FromContext(ctx)

	existing, err := uc.repo.GetItemByID(ctx, item.ID.Hex())
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": item.ID.Hex()})
		return nil, err
	}

	if !existing.IsCustom {
		l.UsecasesWarn(apperrors.ItemNotCustomErr, userID, map[string]any{"id": item.ID.Hex()})
		return nil, apperrors.ItemNotCustomErr
	}
	if existing.CreatedBy == nil || *existing.CreatedBy != userID {
		l.UsecasesWarn(apperrors.ItemNotOwnedErr, userID, map[string]any{"id": item.ID.Hex()})
		return nil, apperrors.ItemNotOwnedErr
	}

	if item.Category != "" && !validCategories[item.Category] {
		l.UsecasesWarn(apperrors.InvalidItemCategoryErr, userID, map[string]any{"category": item.Category})
		return nil, apperrors.InvalidItemCategoryErr
	}
	if item.Rarity != "" && !validRarities[item.Rarity] {
		l.UsecasesWarn(apperrors.InvalidItemRarityErr, userID, map[string]any{"rarity": item.Rarity})
		return nil, apperrors.InvalidItemRarityErr
	}

	item.IsCustom = true
	item.CreatedBy = &userID
	item.SchemaVersion = existing.SchemaVersion

	updated, err := uc.repo.UpdateItem(ctx, item)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": item.ID.Hex()})
		return nil, err
	}

	return updated, nil
}

func (uc *itemUsecases) DeleteItem(ctx context.Context, id string, userID int) error {
	l := logger.FromContext(ctx)

	existing, err := uc.repo.GetItemByID(ctx, id)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return err
	}

	if !existing.IsCustom {
		l.UsecasesWarn(apperrors.ItemNotCustomErr, userID, map[string]any{"id": id})
		return apperrors.ItemNotCustomErr
	}
	if existing.CreatedBy == nil || *existing.CreatedBy != userID {
		l.UsecasesWarn(apperrors.ItemNotOwnedErr, userID, map[string]any{"id": id})
		return apperrors.ItemNotOwnedErr
	}

	if err := uc.repo.DeleteItem(ctx, id); err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return err
	}

	return nil
}

type inventoryUsecases struct {
	repo itemsinterfaces.InventoryRepository
}

func NewInventoryUsecases(repo itemsinterfaces.InventoryRepository) itemsinterfaces.InventoryUsecases {
	return &inventoryUsecases{repo: repo}
}

func (uc *inventoryUsecases) GetContainer(_ context.Context, _ string) (*models.InventoryContainer, error) {
	return nil, apperrors.NotImplementedErr
}

func (uc *inventoryUsecases) GetContainers(_ context.Context, _ models.ContainerFilterParams) ([]*models.InventoryContainer, error) {
	return nil, apperrors.NotImplementedErr
}
