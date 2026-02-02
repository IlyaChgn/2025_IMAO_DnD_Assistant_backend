package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type authUsecases struct {
	repo           authinterface.AuthRepository
	identityRepo   authinterface.IdentityRepository
	providers      map[string]authinterface.OAuthProvider
	sessionManager authinterface.SessionManager
}

func NewAuthUsecases(repo authinterface.AuthRepository, identityRepo authinterface.IdentityRepository,
	providers map[string]authinterface.OAuthProvider, sessionManager authinterface.SessionManager) authinterface.AuthUsecases {
	return &authUsecases{
		repo:           repo,
		identityRepo:   identityRepo,
		providers:      providers,
		sessionManager: sessionManager,
	}
}

func (uc *authUsecases) Login(ctx context.Context, provider string, sessionID string,
	loginData *models.LoginRequest, sessionDuration time.Duration) (*models.User, error) {
	l := logger.FromContext(ctx)

	oauthProvider, ok := uc.providers[provider]
	if !ok {
		err := fmt.Errorf("unsupported OAuth provider: %s", provider)
		l.UsecasesError(err, 0, nil)
		return nil, apperrors.UnsupportedProviderError
	}

	oauthResult, err := oauthProvider.Authenticate(ctx, loginData)
	if err != nil {
		l.UsecasesError(err, 0, nil)
		return nil, err
	}

	// Try identity-based lookup first
	var actualUser *models.User

	identity, identityErr := uc.identityRepo.FindByProvider(ctx, provider, oauthResult.ProviderUserID)
	if identityErr != nil && !errors.Is(identityErr, apperrors.IdentityNotFoundError) {
		l.UsecasesError(identityErr, 0, nil)
		return nil, identityErr
	}

	if identity != nil {
		// Existing user via identity
		userDB, err := uc.repo.GetUserByID(ctx, identity.UserID)
		if err != nil {
			l.UsecasesError(err, 0, nil)
			return nil, err
		}

		if userDB.DisplayName != oauthResult.DisplayName || userDB.AvatarURL != oauthResult.AvatarURL {
			user := &models.User{
				ID:          userDB.ID,
				DisplayName: oauthResult.DisplayName,
				AvatarURL:   oauthResult.AvatarURL,
			}

			actualUser, err = uc.repo.UpdateUser(ctx, user)
			if err != nil {
				l.UsecasesError(err, 0, userDB.ID)
				return nil, err
			}

			l.UsecasesInfo("updated user", actualUser.ID)
		} else {
			actualUser = userDB
		}

		// Best-effort: update identity last_used_at
		if usedErr := uc.identityRepo.UpdateLastUsed(ctx, identity.ID, time.Now().UTC()); usedErr != nil {
			l.UsecasesWarn(usedErr, actualUser.ID, nil)
		}
	} else {
		// New user: create user + identity
		user := &models.User{
			DisplayName: oauthResult.DisplayName,
			AvatarURL:   oauthResult.AvatarURL,
		}

		actualUser, err = uc.repo.CreateUser(ctx, user)
		if err != nil {
			l.UsecasesError(err, 0, nil)
			return nil, err
		}

		l.UsecasesInfo("added new user", actualUser.ID)

		newIdentity := &models.UserIdentity{
			UserID:         actualUser.ID,
			Provider:       provider,
			ProviderUserID: oauthResult.ProviderUserID,
			Email:          oauthResult.Email,
		}

		if identityCreateErr := uc.identityRepo.CreateIdentity(ctx, newIdentity); identityCreateErr != nil {
			l.UsecasesError(identityCreateErr, actualUser.ID, nil)
			return nil, identityCreateErr
		}
	}

	sessionData := &models.FullSessionData{
		Provider: provider,
		Tokens: models.TokensData{
			AccessToken:  oauthResult.AccessToken,
			RefreshToken: oauthResult.RefreshToken,
			IDToken:      oauthResult.IDToken,
		},
		User: *actualUser,
	}

	err = uc.sessionManager.CreateSession(ctx, sessionID, sessionData, sessionDuration)
	if err != nil {
		l.UsecasesError(err, actualUser.ID, nil)
		return nil, err
	}

	if loginErr := uc.repo.UpdateLastLoginAt(ctx, actualUser.ID, time.Now().UTC()); loginErr != nil {
		l.UsecasesWarn(loginErr, actualUser.ID, nil)
	}

	l.UsecasesInfo("user logged in", actualUser.ID)

	return actualUser, nil
}

func (uc *authUsecases) Logout(ctx context.Context, sessionID string) error {
	return uc.sessionManager.RemoveSession(ctx, sessionID)
}

func (uc *authUsecases) CheckAuth(ctx context.Context, sessionID string) (*models.User, bool) {
	data, isAuth := uc.sessionManager.GetSession(ctx, sessionID)
	if !isAuth {
		return nil, false
	}

	return &data.User, true
}

func (uc *authUsecases) GetUserIDBySessionID(ctx context.Context, sessionID string) int {
	data, _ := uc.sessionManager.GetSession(ctx, sessionID)

	return data.User.ID
}
