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
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
)

type googleOAuth struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

func NewGoogleOAuth(clientID, clientSecret, redirectURI string) authinterface.OAuthProvider {
	return &googleOAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}
}

func (g *googleOAuth) Name() string {
	return "google"
}

func (g *googleOAuth) Authenticate(ctx context.Context, loginData *models.LoginRequest) (*models.OAuthResult, error) {
	l := logger.FromContext(ctx)

	tokens, err := g.exchangeCode(ctx, loginData.Code)
	if err != nil {
		return nil, err
	}

	userInfo, err := g.getUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, err
	}

	displayName := userInfo.Name
	if displayName == "" {
		displayName = fmt.Sprintf("%s %s", userInfo.GivenName, userInfo.FamilyName)
	}

	_ = l // logger available for future debug use

	return &models.OAuthResult{
		ProviderUserID: userInfo.Sub,
		DisplayName:    displayName,
		AvatarURL:      userInfo.Picture,
		Email:          userInfo.Email,
		AccessToken:    tokens.AccessToken,
		RefreshToken:   tokens.RefreshToken,
		IDToken:        tokens.IDToken,
	}, nil
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type googleUserInfo struct {
	Sub        string `json:"sub"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Picture    string `json:"picture"`
	Email      string `json:"email"`
}

func (g *googleOAuth) exchangeCode(ctx context.Context, code string) (*googleTokenResponse, error) {
	l := logger.FromContext(ctx)

	params := url.Values{}
	params.Set("code", code)
	params.Set("client_id", g.clientID)
	params.Set("client_secret", g.clientSecret)
	params.Set("redirect_uri", g.redirectURI)
	params.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL,
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
		l.ExternalError(ctx, fmt.Errorf("google token exchange failed: %s", string(body)),
			map[string]any{"code": resp.StatusCode})
		return nil, apperrors.OAuthProviderError
	}

	var tokens googleTokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}

	return &tokens, nil
}

func (g *googleOAuth) getUserInfo(ctx context.Context, accessToken string) (*googleUserInfo, error) {
	l := logger.FromContext(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

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
		l.ExternalError(ctx, fmt.Errorf("google userinfo failed: %s", string(body)),
			map[string]any{"code": resp.StatusCode})
		return nil, apperrors.OAuthProviderError
	}

	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}

	return &info, nil
}
