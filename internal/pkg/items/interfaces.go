package items

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_items.go -package=mocks

type ItemDefinitionRepository interface {
	GetItems(ctx context.Context, filter models.ItemFilterParams) ([]*models.ItemDefinition, int64, error)
	GetItemByEngName(ctx context.Context, engName string) (*models.ItemDefinition, error)
	GetItemByID(ctx context.Context, id string) (*models.ItemDefinition, error)
	CreateItem(ctx context.Context, item *models.ItemDefinition) (*models.ItemDefinition, error)
	UpdateItem(ctx context.Context, item *models.ItemDefinition) (*models.ItemDefinition, error)
	DeleteItem(ctx context.Context, id string) error
	EnsureItemDefinitionIndexes(ctx context.Context) error
}

type InventoryRepository interface {
	GetContainer(ctx context.Context, id string) (*models.InventoryContainer, error)
	GetContainers(ctx context.Context, filter models.ContainerFilterParams) ([]*models.InventoryContainer, error)
	CreateContainer(ctx context.Context, container *models.InventoryContainer) (*models.InventoryContainer, error)
	DeleteContainer(ctx context.Context, id string) error
	UpdateContainer(ctx context.Context, container *models.InventoryContainer) (*models.InventoryContainer, error)
	EnsureInventoryContainerIndexes(ctx context.Context) error
}

type ItemUsecases interface {
	GetItems(ctx context.Context, filter models.ItemFilterParams) (*models.ItemListResponse, error)
	GetItemByEngName(ctx context.Context, engName string) (*models.ItemDefinition, error)
	CreateItem(ctx context.Context, item *models.ItemDefinition, userID int) (*models.ItemDefinition, error)
	UpdateItem(ctx context.Context, item *models.ItemDefinition, userID int) (*models.ItemDefinition, error)
	DeleteItem(ctx context.Context, id string, userID int) error
}

type InventoryUsecases interface {
	GetContainer(ctx context.Context, id string) (*models.InventoryContainer, error)
	GetContainers(ctx context.Context, filter models.ContainerFilterParams) ([]*models.InventoryContainer, error)
}
