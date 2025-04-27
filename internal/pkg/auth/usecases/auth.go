package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
)

type authUsecases struct {
	repo           authinterface.AuthRepository
	sessionManager authinterface.SessionManager
}

func NewAuthUsecases(repo authinterface.AuthRepository,
	sessionManager authinterface.SessionManager) authinterface.AuthUsecases {
	return &authUsecases{
		repo:           repo,
		sessionManager: sessionManager,
	}
}

func (uc *authUsecases) Login(ctx context.Context, sessionID string, vkUser *models.UserPublicInfo,
	tokens *models.VKTokensData, sessionDuration time.Duration) (*models.User, error) {
	var userDB *models.User

	user := &models.User{
		VKID:   vkUser.UserID,
		Name:   fmt.Sprintf("%s %s", vkUser.FirstName, vkUser.LastName),
		Avatar: vkUser.Avatar,
	}

	userDB, err := uc.repo.CheckUser(ctx, vkUser.UserID)
	if err != nil {
		userDB, err = uc.repo.CreateUser(ctx, user)
		if err != nil {
			return nil, err
		}
	} else {
		if userDB.Name != user.Name || userDB.Avatar != user.Avatar {
			userDB, err = uc.repo.UpdateUser(ctx, user)
			if err != nil {
				return nil, err
			}
		}
	}

	sessionData := &models.FullSessionData{
		Tokens: models.TokensData{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			IDToken:      tokens.IDToken,
		},
		User: *userDB,
	}

	err = uc.sessionManager.CreateSession(ctx, sessionID, sessionData, sessionDuration)
	if err != nil {
		return nil, err
	}

	return userDB, nil
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
