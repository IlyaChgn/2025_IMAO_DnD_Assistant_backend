package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
)

type itemUsecases struct {
	repo itemsinterfaces.ItemDefinitionRepository
}

func NewItemUsecases(repo itemsinterfaces.ItemDefinitionRepository) itemsinterfaces.ItemUsecases {
	return &itemUsecases{repo: repo}
}

func (uc *itemUsecases) GetItems(_ context.Context, _ models.ItemFilterParams) (*models.ItemListResponse, error) {
	return nil, apperrors.NotImplementedErr
}

func (uc *itemUsecases) GetItemByEngName(_ context.Context, _ string) (*models.ItemDefinition, error) {
	return nil, apperrors.NotImplementedErr
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
