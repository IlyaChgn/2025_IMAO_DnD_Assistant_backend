package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
)

const actionLogCollection = "action_log"

// ttlSeconds is the TTL for audit log entries (30 days).
const ttlSeconds = 30 * 24 * 60 * 60

type auditLogStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewAuditLogStorage(db *mongo.Database, metrics mymetrics.DBMetrics) actionsinterfaces.AuditLogRepository {
	return &auditLogStorage{db: db, metrics: metrics}
}

func (s *auditLogStorage) EnsureIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(actionLogCollection)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "createdAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(int32(ttlSeconds)),
		},
		{
			Keys: bson.D{
				{Key: "encounterId", Value: 1},
				{Key: "createdAt", Value: -1},
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

func (s *auditLogStorage) Insert(ctx context.Context, entry *models.AuditLogEntry) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(actionLogCollection)

	entry.CreatedAt = time.Now()

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		_, err := collection.InsertOne(ctx, entry)
		return err
	})
	if err != nil {
		l.RepoError(err, map[string]any{"encounterId": entry.EncounterID})
		return apperrors.InsertMongoDataErr
	}

	return nil
}

func (s *auditLogStorage) GetByEncounterID(
	ctx context.Context,
	encounterID string,
	limit int,
	before time.Time,
) ([]*models.AuditLogEntry, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(actionLogCollection)

	filter := bson.D{{Key: "encounterId", Value: encounterID}}
	if !before.IsZero() {
		filter = append(filter, bson.E{Key: "createdAt", Value: bson.M{"$lt": before}})
	}

	if limit <= 0 || limit > 200 {
		limit = 50
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, filter, findOptions)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"encounterId": encounterID})
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var entries []*models.AuditLogEntry

	for cursor.Next(ctx) {
		var entry models.AuditLogEntry
		if err := cursor.Decode(&entry); err != nil {
			l.RepoError(err, map[string]any{"encounterId": encounterID})
			return nil, apperrors.DecodeMongoDataErr
		}
		entries = append(entries, &entry)
	}

	if err := cursor.Err(); err != nil {
		l.RepoError(err, map[string]any{"encounterId": encounterID})
		return nil, apperrors.FindMongoDataErr
	}

	return entries, nil
}
