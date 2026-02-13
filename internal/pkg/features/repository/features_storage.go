package repository

import (
	"context"
	"errors"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	featuresinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"

	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const featureDefinitionsCollection = "feature_definitions"

type featuresStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewFeaturesStorage(db *mongo.Database, metrics mymetrics.DBMetrics) featuresinterfaces.FeaturesRepository {
	return &featuresStorage{db: db, metrics: metrics}
}

func (s *featuresStorage) EnsureIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(featureDefinitionsCollection)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "engName", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "source", Value: 1},
				{Key: "sourceDetail", Value: 1},
				{Key: "level", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "name.eng", Value: "text"},
				{Key: "name.rus", Value: "text"},
				{Key: "description.eng", Value: "text"},
				{Key: "description.rus", Value: "text"},
			},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		l.RepoError(err, map[string]any{"action": "EnsureIndexes"})
		return err
	}

	return nil
}

func (s *featuresStorage) GetFeatures(ctx context.Context, filter models.FeatureFilterParams) ([]*models.FeatureDefinition, int64, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(featureDefinitionsCollection)

	mongoFilter := bson.D{}

	if filter.Source != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "source", Value: filter.Source})
	}
	if filter.Class != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "sourceDetail", Value: primitive.Regex{Pattern: "^" + filter.Class, Options: "i"}})
	}
	if filter.Level != nil {
		mongoFilter = append(mongoFilter, bson.E{Key: "level", Value: *filter.Level})
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
		SetSort(bson.D{{Key: "source", Value: 1}, {Key: "level", Value: 1}, {Key: "name.eng", Value: 1}}).
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

	var features []*models.FeatureDefinition

	for cursor.Next(ctx) {
		var feature models.FeatureDefinition
		if err := cursor.Decode(&feature); err != nil {
			l.RepoError(err, nil)
			return nil, 0, apperrors.DecodeMongoDataErr
		}
		features = append(features, &feature)
	}

	return features, total, nil
}

func (s *featuresStorage) GetFeatureByID(ctx context.Context, id string) (*models.FeatureDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(featureDefinitionsCollection)

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

	var feature models.FeatureDefinition
	if err := result.Decode(&feature); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"id": id})
			return nil, apperrors.FeatureNotFoundErr
		}
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.DecodeMongoDataErr
	}

	return &feature, nil
}

func (s *featuresStorage) GetFeaturesByClass(ctx context.Context, className string, level *int) ([]*models.FeatureDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(featureDefinitionsCollection)

	mongoFilter := bson.D{
		{Key: "source", Value: "class"},
		{Key: "sourceDetail", Value: primitive.Regex{Pattern: "^" + className, Options: "i"}},
	}
	if level != nil {
		mongoFilter = append(mongoFilter, bson.E{Key: "level", Value: *level})
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "level", Value: 1}, {Key: "name.eng", Value: 1}})

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, mongoFilter, findOptions)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"class": className})
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var features []*models.FeatureDefinition

	for cursor.Next(ctx) {
		var feature models.FeatureDefinition
		if err := cursor.Decode(&feature); err != nil {
			l.RepoError(err, map[string]any{"class": className})
			return nil, apperrors.DecodeMongoDataErr
		}
		features = append(features, &feature)
	}

	return features, nil
}

func (s *featuresStorage) UpsertFeature(ctx context.Context, feature *models.FeatureDefinition) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(featureDefinitionsCollection)

	filter := bson.D{{Key: "engName", Value: feature.EngName}}
	update := bson.D{{Key: "$set", Value: feature}}
	opts := options.Update().SetUpsert(true)

	_, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update, opts)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": feature.EngName})
		return err
	}

	return nil
}
