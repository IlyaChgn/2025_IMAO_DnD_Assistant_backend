package repository

import (
	"context"
	"errors"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	dungeongeninterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dungeongen"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const tileMetadataCollection = "tile_metadata"

type tileMetadataStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewTileMetadataStorage(db *mongo.Database, metrics mymetrics.DBMetrics) dungeongeninterfaces.TileMetadataRepository {
	return &tileMetadataStorage{db: db, metrics: metrics}
}

func (s *tileMetadataStorage) EnsureIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(tileMetadataCollection)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "tileId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "role", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "themeTags", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "role", Value: 1},
				{Key: "themeTags", Value: 1},
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

func (s *tileMetadataStorage) UpsertTileMetadata(ctx context.Context, metadata *models.TileMetadata) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(tileMetadataCollection)

	filter := bson.D{{Key: "tileId", Value: metadata.TileID}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "tileId", Value: metadata.TileID},
			{Key: "role", Value: metadata.Role},
			{Key: "themeTags", Value: metadata.ThemeTags},
			{Key: "walkableRatio", Value: metadata.WalkableRatio},
			{Key: "openings", Value: metadata.Openings},
			{Key: "edgeSignatures", Value: metadata.EdgeSignatures},
			{Key: "autoClassified", Value: metadata.AutoClassified},
		}},
	}
	opts := options.Update().SetUpsert(true)

	_, err := dbcall.DBCall[any](fnName, s.metrics, func() (any, error) {
		return collection.UpdateOne(ctx, filter, update, opts)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"tileId": metadata.TileID})
		return apperrors.UpdateMongoDataErr
	}

	return nil
}

func (s *tileMetadataStorage) GetByRole(ctx context.Context, role models.TileRole) ([]*models.TileMetadata, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(tileMetadataCollection)

	filter := bson.D{{Key: "role", Value: role}}

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, filter)
	})
	if err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var results []*models.TileMetadata

	for cursor.Next(ctx) {
		var m models.TileMetadata
		if err := cursor.Decode(&m); err != nil {
			l.RepoError(err, nil)
			return nil, apperrors.DecodeMongoDataErr
		}
		results = append(results, &m)
	}

	if len(results) == 0 {
		return nil, apperrors.NoDocsErr
	}

	return results, nil
}

func (s *tileMetadataStorage) GetByRoleAndTags(ctx context.Context, role models.TileRole, tags []string) ([]*models.TileMetadata, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(tileMetadataCollection)

	filter := bson.D{
		{Key: "role", Value: role},
		{Key: "themeTags", Value: bson.M{"$in": tags}},
	}

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, filter)
	})
	if err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var results []*models.TileMetadata

	for cursor.Next(ctx) {
		var m models.TileMetadata
		if err := cursor.Decode(&m); err != nil {
			l.RepoError(err, nil)
			return nil, apperrors.DecodeMongoDataErr
		}
		results = append(results, &m)
	}

	if len(results) == 0 {
		return nil, apperrors.NoDocsErr
	}

	return results, nil
}

func (s *tileMetadataStorage) GetAll(ctx context.Context) ([]*models.TileMetadata, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(tileMetadataCollection)

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, bson.D{})
	})
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		l.RepoWarn(err, nil)
		return nil, apperrors.NoDocsErr
	} else if err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var results []*models.TileMetadata

	for cursor.Next(ctx) {
		var m models.TileMetadata
		if err := cursor.Decode(&m); err != nil {
			l.RepoError(err, nil)
			return nil, apperrors.DecodeMongoDataErr
		}
		results = append(results, &m)
	}

	return results, nil
}
