package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	maptilesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mapTilesStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewMapTilesStorage(db *mongo.Database, metrics mymetrics.DBMetrics) maptilesinterfaces.MapTilesRepository {
	return &mapTilesStorage{db: db, metrics: metrics}
}

func (s *mapTilesStorage) GetCategories(ctx context.Context, userID int) ([]*models.MapTileCategory, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection("map_tiles")

	possibleIDs := []string{"*", strconv.Itoa(userID)}
	filters := bson.D{
		{Key: "userID", Value: bson.M{"$in": possibleIDs}},
	}

	findOptions := options.Find()
	// Можно добавить сортировку по name, если нужно:
	// findOptions.SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, filters, findOptions)
	})
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		l.RepoWarn(err, nil)
		return nil, apperrors.NoDocsErr
	} else if err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var categories []*models.MapTileCategory

	for cursor.Next(ctx) {
		var cat models.MapTileCategory
		if err := cursor.Decode(&cat); err != nil {
			l.RepoError(err, nil)
			return nil, apperrors.DecodeMongoDataErr
		}
		categories = append(categories, &cat)
	}

	return categories, nil
}

func (s *mapTilesStorage) GetWalkabilityByTileID(ctx context.Context, tileID string) (*models.TileWalkability, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection("map_tile_walkability")

	filter := bson.D{{Key: "tileId", Value: tileID}}

	result, err := dbcall.DBCall[*mongo.SingleResult](fnName, s.metrics, func() (*mongo.SingleResult, error) {
		return collection.FindOne(ctx, filter), nil
	})
	if err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}

	var walkability models.TileWalkability
	if err := result.Decode(&walkability); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"tileId": tileID})
			return nil, apperrors.NoDocsErr
		}
		l.RepoError(err, nil)
		return nil, apperrors.DecodeMongoDataErr
	}

	return &walkability, nil
}

func (s *mapTilesStorage) GetWalkabilityBySetID(ctx context.Context, setID string) ([]*models.TileWalkability, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection("map_tile_walkability")

	filter := bson.D{{Key: "setId", Value: setID}}

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, filter)
	})
	if err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var walkabilities []*models.TileWalkability

	for cursor.Next(ctx) {
		var w models.TileWalkability
		if err := cursor.Decode(&w); err != nil {
			l.RepoError(err, nil)
			return nil, apperrors.DecodeMongoDataErr
		}
		walkabilities = append(walkabilities, &w)
	}

	if len(walkabilities) == 0 {
		l.RepoWarn(apperrors.NoDocsErr, map[string]any{"setId": setID})
		return nil, apperrors.NoDocsErr
	}

	return walkabilities, nil
}

func (s *mapTilesStorage) UpsertWalkability(ctx context.Context, walkability *models.TileWalkability) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection("map_tile_walkability")

	filter := bson.D{{Key: "tileId", Value: walkability.TileID}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "tileId", Value: walkability.TileID},
			{Key: "setId", Value: walkability.SetID},
			{Key: "rows", Value: walkability.Rows},
			{Key: "cols", Value: walkability.Cols},
			{Key: "walkability", Value: walkability.Walkability},
			{Key: "occlusion", Value: walkability.Occlusion},
			{Key: "edges", Value: walkability.Edges},
		}},
	}
	opts := options.Update().SetUpsert(true)

	_, err := dbcall.DBCall[any](fnName, s.metrics, func() (any, error) {
		return collection.UpdateOne(ctx, filter, update, opts)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"tileId": walkability.TileID})
		return apperrors.UpdateMongoDataErr
	}

	return nil
}

func (s *mapTilesStorage) AddTile(ctx context.Context, categoryID string, tile *models.MapTile) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection("map_tiles")
	filter := bson.D{{Key: "id", Value: categoryID}}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "tiles", Value: tile}}}}

	result, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"categoryID": categoryID, "tileID": tile.ID})
		return apperrors.InsertMongoDataErr
	}
	if result.MatchedCount == 0 {
		return apperrors.NoDocsErr
	}

	return nil
}

func (s *mapTilesStorage) UpdateTile(ctx context.Context, categoryID string, tile *models.MapTile) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection("map_tiles")
	filter := bson.D{
		{Key: "id", Value: categoryID},
		{Key: "tiles.id", Value: tile.ID},
	}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "tiles.$.name", Value: tile.Name},
		{Key: "tiles.$.imageUrl", Value: tile.ImageURL},
	}}}

	result, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"categoryID": categoryID, "tileID": tile.ID})
		return apperrors.UpdateMongoDataErr
	}
	if result.MatchedCount == 0 {
		return apperrors.NoDocsErr
	}

	return nil
}

func (s *mapTilesStorage) DeleteTile(ctx context.Context, categoryID string, tileID string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection("map_tiles")
	filter := bson.D{{Key: "id", Value: categoryID}}
	update := bson.D{{Key: "$pull", Value: bson.D{
		{Key: "tiles", Value: bson.D{{Key: "id", Value: tileID}}},
	}}}

	result, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"categoryID": categoryID, "tileID": tileID})
		return apperrors.DeleteMongoDataErr
	}
	if result.MatchedCount == 0 {
		return apperrors.NoDocsErr
	}

	return nil
}
