package repository

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const inventoryContainersCollection = "inventory_containers"

type inventoryStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewInventoryStorage(db *mongo.Database, metrics mymetrics.DBMetrics) itemsinterfaces.InventoryRepository {
	return &inventoryStorage{db: db, metrics: metrics}
}

func (s *inventoryStorage) EnsureInventoryContainerIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(inventoryContainersCollection)

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "encounterId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "ownerId", Value: 1},
				{Key: "kind", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "items.id", Value: 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		l.RepoError(err, map[string]any{"action": "EnsureInventoryContainerIndexes"})
		return err
	}

	return nil
}

func (s *inventoryStorage) GetContainer(_ context.Context, _ string) (*models.InventoryContainer, error) {
	return nil, apperrors.NotImplementedErr
}

func (s *inventoryStorage) GetContainers(_ context.Context, _ models.ContainerFilterParams) ([]*models.InventoryContainer, error) {
	return nil, apperrors.NotImplementedErr
}

func (s *inventoryStorage) CreateContainer(_ context.Context, _ *models.InventoryContainer) (*models.InventoryContainer, error) {
	return nil, apperrors.NotImplementedErr
}

func (s *inventoryStorage) DeleteContainer(_ context.Context, _ string) error {
	return apperrors.NotImplementedErr
}

func (s *inventoryStorage) UpdateContainer(_ context.Context, _ *models.InventoryContainer) (*models.InventoryContainer, error) {
	return nil, apperrors.NotImplementedErr
}
