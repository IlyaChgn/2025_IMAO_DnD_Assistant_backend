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
	"go.mongodb.org/mongo-driver/mongo/options"
)

const itemDefinitionsCollection = "item_definitions"

type itemsStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewItemsStorage(db *mongo.Database, metrics mymetrics.DBMetrics) itemsinterfaces.ItemDefinitionRepository {
	return &itemsStorage{db: db, metrics: metrics}
}

func (s *itemsStorage) EnsureItemDefinitionIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(itemDefinitionsCollection)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "engName", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "category", Value: 1},
				{Key: "rarity", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "name.eng", Value: "text"},
				{Key: "name.rus", Value: "text"},
			},
		},
		{
			Keys: bson.D{{Key: "tags", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "isCustom", Value: 1},
				{Key: "createdBy", Value: 1},
			},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		l.RepoError(err, map[string]any{"action": "EnsureItemDefinitionIndexes"})
		return err
	}

	return nil
}

func (s *itemsStorage) GetItems(_ context.Context, _ models.ItemFilterParams) ([]*models.ItemDefinition, int64, error) {
	return nil, 0, apperrors.NotImplementedErr
}

func (s *itemsStorage) GetItemByEngName(_ context.Context, _ string) (*models.ItemDefinition, error) {
	return nil, apperrors.NotImplementedErr
}

func (s *itemsStorage) GetItemByID(_ context.Context, _ string) (*models.ItemDefinition, error) {
	return nil, apperrors.NotImplementedErr
}

func (s *itemsStorage) CreateItem(_ context.Context, _ *models.ItemDefinition) (*models.ItemDefinition, error) {
	return nil, apperrors.NotImplementedErr
}

func (s *itemsStorage) UpdateItem(_ context.Context, _ *models.ItemDefinition) (*models.ItemDefinition, error) {
	return nil, apperrors.NotImplementedErr
}

func (s *itemsStorage) DeleteItem(_ context.Context, _ string) error {
	return apperrors.NotImplementedErr
}
