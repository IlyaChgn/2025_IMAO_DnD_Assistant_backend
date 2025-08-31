package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
)

type authUsecases struct {
	repo           authinterface.AuthRepository
	vkApi          authinterface.VKApi
	sessionManager authinterface.SessionManager
}

func NewAuthUsecases(repo authinterface.AuthRepository, vkApi authinterface.VKApi,
	sessionManager authinterface.SessionManager) authinterface.AuthUsecases {
	return &authUsecases{
		repo:           repo,
		vkApi:          vkApi,
		sessionManager: sessionManager,
	}
}

func (uc *authUsecases) Login(ctx context.Context, sessionID string,
	loginData *models.LoginRequest, sessionDuration time.Duration) (*models.User, error) {
	vkRawData, err := uc.vkApi.ExchangeCode(ctx, loginData)
	if err != nil {
		return nil, err
	}

	var vkTokens models.VKTokensData

	err = json.Unmarshal(vkRawData, &vkTokens)
	if err != nil {
		return nil, err
	}

	rawPublicInfo, err := uc.vkApi.GetPublicInfo(ctx, vkTokens.IDToken)
	if err != nil {
		return nil, err
	}

	var publicInfo models.PublicInfo

	err = json.Unmarshal(rawPublicInfo, &publicInfo)
	if err != nil {
		return nil, err
	}

	var userDB *models.User
	vkUser := publicInfo.User
	user := &models.User{
		VKID:   vkUser.UserID,
		Name:   fmt.Sprintf("%s %s", vkUser.FirstName, vkUser.LastName),
		Avatar: vkUser.Avatar,
	}

	userDB, err = uc.repo.CheckUser(ctx, vkUser.UserID)
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
			AccessToken:  vkTokens.AccessToken,
			RefreshToken: vkTokens.RefreshToken,
			IDToken:      vkTokens.IDToken,
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
