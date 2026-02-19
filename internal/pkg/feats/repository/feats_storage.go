package repository

import (
	"context"
	"errors"

	featsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const featDefinitionsCollection = "feat_definitions"

type featsStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewFeatsStorage(db *mongo.Database, metrics mymetrics.DBMetrics) featsinterfaces.FeatsRepository {
	return &featsStorage{db: db, metrics: metrics}
}

func (s *featsStorage) EnsureIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(featDefinitionsCollection)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "engName", Value: 1}},
			Options: options.Index().SetUnique(true),
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

func (s *featsStorage) GetFeats(ctx context.Context, filter models.FeatFilterParams) ([]*models.FeatDefinition, int64, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(featDefinitionsCollection)

	mongoFilter := bson.D{}

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

	var feats []*models.FeatDefinition

	for cursor.Next(ctx) {
		var feat models.FeatDefinition
		if err := cursor.Decode(&feat); err != nil {
			l.RepoError(err, nil)
			return nil, 0, apperrors.DecodeMongoDataErr
		}
		feats = append(feats, &feat)
	}

	if err := cursor.Err(); err != nil {
		l.RepoError(err, nil)
		return nil, 0, apperrors.FindMongoDataErr
	}

	return feats, total, nil
}

func (s *featsStorage) GetFeatByEngName(ctx context.Context, engName string) (*models.FeatDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(featDefinitionsCollection)

	result, err := dbcall.DBCall[*mongo.SingleResult](fnName, s.metrics, func() (*mongo.SingleResult, error) {
		return collection.FindOne(ctx, bson.D{{Key: "engName", Value: engName}}), nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": engName})
		return nil, apperrors.FindMongoDataErr
	}

	var feat models.FeatDefinition
	if err := result.Decode(&feat); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"engName": engName})
			return nil, apperrors.FeatNotFoundErr
		}
		l.RepoError(err, map[string]any{"engName": engName})
		return nil, apperrors.DecodeMongoDataErr
	}

	return &feat, nil
}

func (s *featsStorage) UpsertFeat(ctx context.Context, feat *models.FeatDefinition) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(featDefinitionsCollection)

	filter := bson.D{{Key: "engName", Value: feat.EngName}}
	update := bson.D{{Key: "$set", Value: feat}}
	opts := options.Update().SetUpsert(true)

	_, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update, opts)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": feat.EngName})
		return err
	}

	return nil
}
