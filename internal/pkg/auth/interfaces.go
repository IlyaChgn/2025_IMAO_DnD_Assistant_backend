package auth

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"time"
)

type AuthRepository interface {
	CheckUser(ctx context.Context, vkid string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) (*models.User, error)
}

type AuthUsecases interface {
	Login(ctx context.Context, sessionID string, vkUser *models.UserPublicInfo,
		tokens *models.VKTokensData, sessionDuration time.Duration) (*models.User, error)
	Logout(ctx context.Context, sessionID string) error
	CheckAuth(ctx context.Context, sessionID string) (*models.User, bool)
}

type SessionManager interface {
	CreateSession(ctx context.Context, sessionID string, session *models.FullSessionData,
		sessionDuration time.Duration) error
	RemoveSession(ctx context.Context, sessionID string) error
	GetSession(ctx context.Context, sessionID string) (*models.FullSessionData, bool)
}
