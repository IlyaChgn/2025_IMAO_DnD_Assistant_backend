package repository

import (
	"context"
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"time"

	"github.com/redis/go-redis/v9"
)

type sessionManager struct {
	client  *redis.Client
	metrics *mymetrics.DBMetrics
}

func NewSessionManager(client *redis.Client, metrics *mymetrics.DBMetrics) authinterface.SessionManager {
	return &sessionManager{
		client:  client,
		metrics: metrics,
	}
}

func (manager *sessionManager) CreateSession(ctx context.Context, sessionID string, session *models.FullSessionData,
	sessionDuration time.Duration) error {
	rawSession, err := json.Marshal(session)
	if err != nil {
		return apperrors.MarshallingSessionError
	}

	err = manager.client.Set(ctx, sessionID, rawSession, sessionDuration).Err()
	if err != nil {
		return apperrors.AddToRedisError
	}

	return nil
}

func (manager *sessionManager) RemoveSession(ctx context.Context, sessionID string) error {
	if _, exists := manager.GetSession(ctx, sessionID); !exists {
		return apperrors.SessionNotExistsError
	}

	_, err := manager.client.Del(context.Background(), sessionID).Result()
	if err != nil {
		return apperrors.DeleteFromRedisError
	}

	return nil
}

func (manager *sessionManager) GetSession(ctx context.Context, sessionID string) (*models.FullSessionData, bool) {

	rawSession, _ := manager.client.Get(ctx, sessionID).Result()

	var session *models.FullSessionData
	if err := json.Unmarshal([]byte(rawSession), &session); err != nil {
		return nil, false
	}

	return session, session != nil
}
