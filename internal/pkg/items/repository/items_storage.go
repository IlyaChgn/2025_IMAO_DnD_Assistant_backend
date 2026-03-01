package repository

import (
	"context"
	"errors"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func (s *itemsStorage) GetItems(ctx context.Context, filter models.ItemFilterParams) ([]*models.ItemDefinition, int64, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(itemDefinitionsCollection)

	mongoFilter := bson.D{}

	if filter.Category != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "category", Value: filter.Category})
	}
	if filter.Rarity != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "rarity", Value: filter.Rarity})
	}
	if len(filter.Tags) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "tags", Value: bson.M{"$all": filter.Tags}})
	}
	if filter.Search != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "$text", Value: bson.M{"$search": filter.Search}})
	}

	total, err := dbcall.DBCall[int64](fnName, s.metrics, func() (int64, error) {
		return collection.CountDocuments(ctx, mongoFilter)
	})
	if err != nil {
		l.RepoError(err, nil)
		return nil, 0, apperrors.FindMongoDataErr
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "name.eng", Value: 1}}).
		SetSkip(int64(filter.Page * filter.Size)).
		SetLimit(int64(filter.Size))

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, mongoFilter, findOptions)
	})
	if err != nil {
		l.RepoError(err, nil)
		return nil, 0, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var items []*models.ItemDefinition

	for cursor.Next(ctx) {
		var item models.ItemDefinition
		if err := cursor.Decode(&item); err != nil {
			l.RepoError(err, nil)
			return nil, 0, apperrors.DecodeMongoDataErr
		}
		items = append(items, &item)
	}

	return items, total, nil
}

func (s *itemsStorage) GetItemByEngName(ctx context.Context, engName string) (*models.ItemDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(itemDefinitionsCollection)

	result, err := dbcall.DBCall[*mongo.SingleResult](fnName, s.metrics, func() (*mongo.SingleResult, error) {
		return collection.FindOne(ctx, bson.D{{Key: "engName", Value: engName}}), nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": engName})
		return nil, apperrors.FindMongoDataErr
	}

	var item models.ItemDefinition
	if err := result.Decode(&item); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"engName": engName})
			return nil, apperrors.ItemNotFoundErr
		}
		l.RepoError(err, map[string]any{"engName": engName})
		return nil, apperrors.DecodeMongoDataErr
	}

	return &item, nil
}

func (s *itemsStorage) GetItemByID(ctx context.Context, id string) (*models.ItemDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(itemDefinitionsCollection)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return nil, apperrors.InvalidIDErr
	}

	result, err := dbcall.DBCall[*mongo.SingleResult](fnName, s.metrics, func() (*mongo.SingleResult, error) {
		return collection.FindOne(ctx, bson.D{{Key: "_id", Value: objID}}), nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.FindMongoDataErr
	}

	var item models.ItemDefinition
	if err := result.Decode(&item); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"id": id})
			return nil, apperrors.ItemNotFoundErr
		}
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.DecodeMongoDataErr
	}

	return &item, nil
}

func (s *itemsStorage) GetRandomItemsByRarity(ctx context.Context, rarity models.ItemRarity, count int) ([]*models.ItemDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(itemDefinitionsCollection)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "rarity", Value: rarity}}}},
		{{Key: "$sample", Value: bson.D{{Key: "size", Value: count}}}},
	}

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Aggregate(ctx, pipeline)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"rarity": rarity, "count": count})
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var items []*models.ItemDefinition
	for cursor.Next(ctx) {
		var item models.ItemDefinition
		if err := cursor.Decode(&item); err != nil {
			l.RepoError(err, nil)
			return nil, apperrors.DecodeMongoDataErr
		}
		items = append(items, &item)
	}

	return items, nil
}

func (s *itemsStorage) CreateItem(ctx context.Context, item *models.ItemDefinition) (*models.ItemDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(itemDefinitionsCollection)

	res, err := dbcall.DBCall[*mongo.InsertOneResult](fnName, s.metrics, func() (*mongo.InsertOneResult, error) {
		return collection.InsertOne(ctx, item)
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, apperrors.DuplicateEngNameErr
		}
		l.RepoError(err, map[string]any{"engName": item.EngName})
		return nil, apperrors.UpdateMongoDataErr
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		item.ID = oid
	}

	return item, nil
}

func (s *itemsStorage) UpdateItem(ctx context.Context, item *models.ItemDefinition) (*models.ItemDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(itemDefinitionsCollection)

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		res, err := collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: item.ID}}, item)
		if err != nil {
			return err
		}
		if res.MatchedCount == 0 {
			return apperrors.ItemNotFoundErr
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, apperrors.ItemNotFoundErr) {
			return nil, err
		}
		if mongo.IsDuplicateKeyError(err) {
			return nil, apperrors.DuplicateEngNameErr
		}
		l.RepoError(err, map[string]any{"id": item.ID.Hex()})
		return nil, apperrors.UpdateMongoDataErr
	}

	return item, nil
}

func (s *itemsStorage) DeleteItem(ctx context.Context, id string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(itemDefinitionsCollection)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return apperrors.InvalidIDErr
	}

	err = dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		res, err := collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: objID}})
		if err != nil {
			return err
		}
		if res.DeletedCount == 0 {
			return apperrors.ItemNotFoundErr
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, apperrors.ItemNotFoundErr) {
			return err
		}
		l.RepoError(err, map[string]any{"id": id})
		return apperrors.DeleteMongoDataErr
	}

	return nil
}
