package repository

import (
	"context"
	"errors"
	"log"
	"regexp"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const charactersV2Collection = "characters_v2"

type characterBaseStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewCharacterBaseStorage(db *mongo.Database, metrics mymetrics.DBMetrics) characterinterfaces.CharacterBaseRepository {
	storage := &characterBaseStorage{
		db:      db,
		metrics: metrics,
	}

	storage.ensureIndexes()

	return storage
}

func (s *characterBaseStorage) ensureIndexes() {
	collection := s.db.Collection(charactersV2Collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		{Keys: bson.D{{Key: "classes.className", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "updatedAt", Value: -1}}},
	}

	if _, err := collection.Indexes().CreateMany(ctx, indexes); err != nil {
		log.Printf("characters_v2: ensureIndexes warning: %v", err)
	}
}

func (s *characterBaseStorage) Create(ctx context.Context, char *models.CharacterBase) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(charactersV2Collection)

	if char.ID.IsZero() {
		char.ID = primitive.NewObjectID()
	}

	now := time.Now().UTC().Format(time.RFC3339)
	char.CreatedAt = now
	char.UpdatedAt = now
	char.Version = 1

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		_, err := collection.InsertOne(ctx, char)
		return err
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": char.ID})
		return apperrors.InsertMongoDataErr
	}

	return nil
}

func (s *characterBaseStorage) GetByID(ctx context.Context, id string) (*models.CharacterBase, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(charactersV2Collection)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return nil, apperrors.InvalidIDErr
	}

	var char models.CharacterBase

	_, err = dbcall.DBCall[*models.CharacterBase](fnName, s.metrics, func() (*models.CharacterBase, error) {
		err := collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&char)
		return &char, err
	})
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	} else if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.FindMongoDataErr
	}

	return &char, nil
}

// Update applies an update with optimistic locking.
// The filter matches both _id and version; if no document matches, a version conflict (409) is assumed.
func (s *characterBaseStorage) Update(ctx context.Context, char *models.CharacterBase, expectedVersion int) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(charactersV2Collection)

	filter := bson.M{
		"_id":     char.ID,
		"userId":  char.UserID,
		"version": expectedVersion,
	}

	now := time.Now().UTC().Format(time.RFC3339)
	char.UpdatedAt = now
	char.Version = expectedVersion + 1

	// Temporarily clear ID so $set doesn't include immutable _id field
	savedID := char.ID
	char.ID = primitive.ObjectID{}
	update := bson.M{
		"$set": char,
	}
	char.ID = savedID

	result, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": char.ID})
		return apperrors.UpdateMongoDataErr
	}

	if result.MatchedCount == 0 {
		return apperrors.VersionConflictErr
	}

	return nil
}

func (s *characterBaseStorage) Delete(ctx context.Context, id string, userID string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(charactersV2Collection)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return apperrors.InvalidIDErr
	}

	result, err := dbcall.DBCall[*mongo.DeleteResult](fnName, s.metrics, func() (*mongo.DeleteResult, error) {
		return collection.DeleteOne(ctx, bson.M{"_id": objID, "userId": userID})
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return apperrors.DeleteMongoDataErr
	}

	if result.DeletedCount == 0 {
		return apperrors.PermissionDeniedError
	}

	return nil
}

func (s *characterBaseStorage) List(ctx context.Context, userID string, page, size int,
	search string) ([]*models.CharacterBase, int64, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(charactersV2Collection)

	filter := bson.M{"userId": userID}
	if search != "" {
		filter["name"] = bson.M{"$regex": regexp.QuoteMeta(search), "$options": "i"}
	}

	// Count total
	total, err := dbcall.DBCall[int64](fnName, s.metrics, func() (int64, error) {
		return collection.CountDocuments(ctx, filter)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"userId": userID})
		return nil, 0, apperrors.FindMongoDataErr
	}

	// Fetch page
	skip := int64(page * size)
	findOpts := options.Find().
		SetSkip(skip).
		SetLimit(int64(size)).
		SetSort(bson.D{{Key: "updatedAt", Value: -1}})

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, filter, findOpts)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"userId": userID})
		return nil, 0, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var chars []*models.CharacterBase
	if err := cursor.All(ctx, &chars); err != nil {
		l.RepoError(err, nil)
		return nil, 0, apperrors.DecodeMongoDataErr
	}

	return chars, total, nil
}

func (s *characterBaseStorage) UpdateAvatarURL(ctx context.Context, id string, userID string, avatarURL string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(charactersV2Collection)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return apperrors.InvalidIDErr
	}

	filter := bson.M{"_id": objID, "userId": userID}
	update := bson.M{
		"$set": bson.M{
			"avatar.url": avatarURL,
			"updatedAt":  time.Now().UTC().Format(time.RFC3339),
		},
	}

	result, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return apperrors.UpdateMongoDataErr
	}

	if result.MatchedCount == 0 {
		return apperrors.PermissionDeniedError
	}

	return nil
}

func (s *characterBaseStorage) ClearAvatar(ctx context.Context, id string, userID string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(charactersV2Collection)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return apperrors.InvalidIDErr
	}

	filter := bson.M{"_id": objID, "userId": userID}
	update := bson.M{
		"$unset": bson.M{"avatar": ""},
		"$set":   bson.M{"updatedAt": time.Now().UTC().Format(time.RFC3339)},
	}

	result, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return apperrors.UpdateMongoDataErr
	}

	if result.MatchedCount == 0 {
		return apperrors.PermissionDeniedError
	}

	return nil
}
