package auth

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"time"
)

type AuthRepository interface{}

type AuthUsecases interface{}

type SessionManager interface {
	CreateSession(ctx context.Context, sessionID string, session *models.FullSessionData,
		sessionDuration time.Duration) error
	RemoveSession(ctx context.Context, sessionID string) error
	GetSession(ctx context.Context, sessionID string) (*models.FullSessionData, bool)
}
