package repository

import (
	"context"
	"errors"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	spellsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells"

	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const spellDefinitionsCollection = "spell_definitions"

type spellsStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewSpellsStorage(db *mongo.Database, metrics mymetrics.DBMetrics) spellsinterfaces.SpellsRepository {
	return &spellsStorage{db: db, metrics: metrics}
}

func (s *spellsStorage) EnsureIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(spellDefinitionsCollection)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "engName", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "level", Value: 1},
				{Key: "school", Value: 1},
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
		{
			Keys: bson.D{{Key: "classes", Value: 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		l.RepoError(err, map[string]any{"action": "EnsureIndexes"})
		return err
	}

	return nil
}

func (s *spellsStorage) GetSpells(ctx context.Context, filter models.SpellFilterParams) ([]*models.SpellDefinition, int64, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(spellDefinitionsCollection)

	mongoFilter := bson.D{}

	if filter.Class != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "classes", Value: bson.M{"$in": []string{filter.Class}}})
	}
	if filter.Level != nil {
		mongoFilter = append(mongoFilter, bson.E{Key: "level", Value: *filter.Level})
	}
	if filter.School != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "school", Value: filter.School})
	}
	if filter.Ritual != nil && *filter.Ritual {
		mongoFilter = append(mongoFilter, bson.E{Key: "ritual", Value: true})
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
		SetSort(bson.D{{Key: "level", Value: 1}, {Key: "name.eng", Value: 1}}).
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

	var spells []*models.SpellDefinition

	for cursor.Next(ctx) {
		var spell models.SpellDefinition
		if err := cursor.Decode(&spell); err != nil {
			l.RepoError(err, nil)
			return nil, 0, apperrors.DecodeMongoDataErr
		}
		spells = append(spells, &spell)
	}

	return spells, total, nil
}

func (s *spellsStorage) GetSpellByID(ctx context.Context, id string) (*models.SpellDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(spellDefinitionsCollection)

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

	var spell models.SpellDefinition
	if err := result.Decode(&spell); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"id": id})
			return nil, apperrors.SpellNotFoundErr
		}
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.DecodeMongoDataErr
	}

	return &spell, nil
}

func (s *spellsStorage) GetSpellsByClass(ctx context.Context, className string, level *int) ([]*models.SpellDefinition, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(spellDefinitionsCollection)

	mongoFilter := bson.D{
		{Key: "classes", Value: bson.M{"$in": []string{className}}},
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

	var spells []*models.SpellDefinition

	for cursor.Next(ctx) {
		var spell models.SpellDefinition
		if err := cursor.Decode(&spell); err != nil {
			l.RepoError(err, map[string]any{"class": className})
			return nil, apperrors.DecodeMongoDataErr
		}
		spells = append(spells, &spell)
	}

	return spells, nil
}
