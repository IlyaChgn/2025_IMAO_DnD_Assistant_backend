package repository

import (
	"context"
	"errors"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	racesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const raceDefinitionsCollection = "race_definitions"

type racesStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewRacesStorage(db *mongo.Database, metrics mymetrics.DBMetrics) racesinterfaces.RacesRepository {
	return &racesStorage{db: db, metrics: metrics}
}

func (s *racesStorage) EnsureIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(raceDefinitionsCollection)

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

func (s *racesStorage) GetRaces(ctx context.Context, filter models.RaceFilterParams) ([]*models.RaceDefinition, int64, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(raceDefinitionsCollection)

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

	var races []*models.RaceDefinition

	for cursor.Next(ctx) {
		var race models.RaceDefinition
		if err := cursor.Decode(&race); err != nil {
			l.RepoError(err, nil)
			return nil, 0, apperrors.DecodeMongoDataErr
		}
		races = append(races, &race)
	}

	if err := cursor.Err(); err != nil {
		l.RepoError(err, nil)
		return nil, 0, apperrors.FindMongoDataErr
	}

	return races, total, nil
}

func (s *racesStorage) GetRaceByEngName(ctx context.Context, engName string) (*models.RaceDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(raceDefinitionsCollection)

	result, err := dbcall.DBCall[*mongo.SingleResult](fnName, s.metrics, func() (*mongo.SingleResult, error) {
		return collection.FindOne(ctx, bson.D{{Key: "engName", Value: engName}}), nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": engName})
		return nil, apperrors.FindMongoDataErr
	}

	var race models.RaceDefinition
	if err := result.Decode(&race); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"engName": engName})
			return nil, apperrors.RaceNotFoundErr
		}
		l.RepoError(err, map[string]any{"engName": engName})
		return nil, apperrors.DecodeMongoDataErr
	}

	return &race, nil
}

func (s *racesStorage) UpsertRace(ctx context.Context, race *models.RaceDefinition) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(raceDefinitionsCollection)

	filter := bson.D{{Key: "engName", Value: race.EngName}}
	update := bson.D{{Key: "$set", Value: race}}
	opts := options.Update().SetUpsert(true)

	_, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update, opts)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": race.EngName})
		return err
	}

	return nil
}
