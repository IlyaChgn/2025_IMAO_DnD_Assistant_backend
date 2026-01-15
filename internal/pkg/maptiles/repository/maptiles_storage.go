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
