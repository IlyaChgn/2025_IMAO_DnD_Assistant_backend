package repository

import (
	"context"
	"errors"

	backgroundsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds"
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

const backgroundDefinitionsCollection = "background_definitions"

type backgroundsStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewBackgroundsStorage(db *mongo.Database, metrics mymetrics.DBMetrics) backgroundsinterfaces.BackgroundsRepository {
	return &backgroundsStorage{db: db, metrics: metrics}
}

func (s *backgroundsStorage) EnsureIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(backgroundDefinitionsCollection)

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

func (s *backgroundsStorage) GetBackgrounds(ctx context.Context, filter models.BackgroundFilterParams) ([]*models.BackgroundDefinition, int64, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(backgroundDefinitionsCollection)

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

	var backgrounds []*models.BackgroundDefinition

	for cursor.Next(ctx) {
		var bg models.BackgroundDefinition
		if err := cursor.Decode(&bg); err != nil {
			l.RepoError(err, nil)
			return nil, 0, apperrors.DecodeMongoDataErr
		}
		backgrounds = append(backgrounds, &bg)
	}

	if err := cursor.Err(); err != nil {
		l.RepoError(err, nil)
		return nil, 0, apperrors.FindMongoDataErr
	}

	return backgrounds, total, nil
}

func (s *backgroundsStorage) GetBackgroundByEngName(ctx context.Context, engName string) (*models.BackgroundDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(backgroundDefinitionsCollection)

	result, err := dbcall.DBCall[*mongo.SingleResult](fnName, s.metrics, func() (*mongo.SingleResult, error) {
		return collection.FindOne(ctx, bson.D{{Key: "engName", Value: engName}}), nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": engName})
		return nil, apperrors.FindMongoDataErr
	}

	var bg models.BackgroundDefinition
	if err := result.Decode(&bg); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"engName": engName})
			return nil, apperrors.BackgroundNotFoundErr
		}
		l.RepoError(err, map[string]any{"engName": engName})
		return nil, apperrors.DecodeMongoDataErr
	}

	return &bg, nil
}

func (s *backgroundsStorage) UpsertBackground(ctx context.Context, bg *models.BackgroundDefinition) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(backgroundDefinitionsCollection)

	filter := bson.D{{Key: "engName", Value: bg.EngName}}
	update := bson.D{{Key: "$set", Value: bg}}
	opts := options.Update().SetUpsert(true)

	_, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update, opts)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": bg.EngName})
		return err
	}

	return nil
}
