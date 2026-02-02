package auth

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_auth.go -package=mocks

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"time"
)

type AuthRepository interface {
	GetUserByID(ctx context.Context, userID int) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) (*models.User, error)
	UpdateLastLoginAt(ctx context.Context, userID int, t time.Time) error
}

type AuthUsecases interface {
	Login(ctx context.Context, sessionID string,
		loginData *models.LoginRequest, sessionDuration time.Duration) (*models.User, error)
	Logout(ctx context.Context, sessionID string) error
	CheckAuth(ctx context.Context, sessionID string) (*models.User, bool)
	// GetUserIDBySessionID следует использовать только если точно понятно, что пользователь авторизован и данные корректны
	GetUserIDBySessionID(ctx context.Context, sessionID string) int
}

type SessionManager interface {
	CreateSession(ctx context.Context, sessionID string, session *models.FullSessionData,
		sessionDuration time.Duration) error
	RemoveSession(ctx context.Context, sessionID string) error
	GetSession(ctx context.Context, sessionID string) (*models.FullSessionData, bool)
}

type IdentityRepository interface {
	FindByProvider(ctx context.Context, provider, providerUserID string) (*models.UserIdentity, error)
	CreateIdentity(ctx context.Context, identity *models.UserIdentity) error
	UpdateLastUsed(ctx context.Context, identityID int, t time.Time) error
	ListByUserID(ctx context.Context, userID int) ([]models.UserIdentity, error)
}

type VKApi interface {
	ExchangeCode(ctx context.Context, data *models.LoginRequest) ([]byte, error)
	GetPublicInfo(ctx context.Context, idToken string) ([]byte, error)
}
