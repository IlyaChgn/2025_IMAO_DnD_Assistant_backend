package external

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

const (
	yandexTokenURL    = "https://oauth.yandex.ru/token"
	yandexUserInfoURL = "https://login.yandex.ru/info?format=json"
)

type yandexOAuth struct {
	clientID     string
	clientSecret string
}

func NewYandexOAuth(clientID, clientSecret string) authinterface.OAuthProvider {
	return &yandexOAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (y *yandexOAuth) Name() string {
	return "yandex"
}

func (y *yandexOAuth) Authenticate(ctx context.Context, loginData *models.LoginRequest) (*models.OAuthResult, error) {
	tokens, err := y.exchangeCode(ctx, loginData.Code)
	if err != nil {
		return nil, err
	}

	userInfo, err := y.getUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, err
	}

	displayName := userInfo.DisplayName
	if displayName == "" {
		displayName = fmt.Sprintf("%s %s", userInfo.FirstName, userInfo.LastName)
	}

	var avatarURL string
	if userInfo.DefaultAvatarID != "" && !userInfo.IsAvatarEmpty {
		avatarURL = fmt.Sprintf("https://avatars.yandex.net/get-yapic/%s/islands-200", userInfo.DefaultAvatarID)
	}

	return &models.OAuthResult{
		ProviderUserID: userInfo.ID,
		DisplayName:    displayName,
		AvatarURL:      avatarURL,
		Email:          userInfo.DefaultEmail,
		AccessToken:    tokens.AccessToken,
		RefreshToken:   tokens.RefreshToken,
	}, nil
}

type yandexTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type yandexUserInfo struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	DefaultEmail    string `json:"default_email"`
	DefaultAvatarID string `json:"default_avatar_id"`
	IsAvatarEmpty   bool   `json:"is_avatar_empty"`
}

func (y *yandexOAuth) exchangeCode(ctx context.Context, code string) (*yandexTokenResponse, error) {
	l := logger.FromContext(ctx)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("client_id", y.clientID)
	params.Set("client_secret", y.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, yandexTokenURL,
		strings.NewReader(params.Encode()))
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.OAuthProviderError
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}

	if resp.StatusCode != http.StatusOK {
		l.ExternalError(ctx, fmt.Errorf("yandex token exchange failed: %s", string(body)),
			map[string]any{"code": resp.StatusCode})
		return nil, apperrors.OAuthProviderError
	}

	var tokens yandexTokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}

	return &tokens, nil
}

func (y *yandexOAuth) getUserInfo(ctx context.Context, accessToken string) (*yandexUserInfo, error) {
	l := logger.FromContext(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, yandexUserInfoURL, nil)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.OAuthProviderError
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}

	if resp.StatusCode != http.StatusOK {
		l.ExternalError(ctx, fmt.Errorf("yandex userinfo failed: %s", string(body)),
			map[string]any{"code": resp.StatusCode})
		return nil, apperrors.OAuthProviderError
	}

	var info yandexUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}

	return &info, nil
}
