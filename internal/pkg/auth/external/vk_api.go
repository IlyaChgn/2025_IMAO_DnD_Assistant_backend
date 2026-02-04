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
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/config"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
)

type vkAPI struct {
	RedirectURI string
	ClientID    string
	SecretKey   string
	ServiceKey  string
	Exchange    config.VKMethodConfig
	PublicInfo  config.VKMethodConfig
}

func NewVKApi(redirectURI, clientID, secretKey, serviceKey string,
	exchange, publicInfo config.VKMethodConfig) authinterface.OAuthProvider {
	return &vkAPI{
		RedirectURI: redirectURI,
		ClientID:    clientID,
		SecretKey:   secretKey,
		ServiceKey:  serviceKey,
		Exchange:    exchange,
		PublicInfo:  publicInfo,
	}
}

func (a *vkAPI) Name() string {
	return "vk"
}

func (a *vkAPI) Authenticate(ctx context.Context, loginData *models.LoginRequest) (*models.OAuthResult, error) {
	l := logger.FromContext(ctx)

	rawTokens, err := a.exchangeCode(ctx, loginData)
	if err != nil {
		return nil, err
	}

	var vkTokens models.VKTokensData
	if err := json.Unmarshal(rawTokens, &vkTokens); err != nil {
		l.UsecasesError(err, 0, nil)
		return nil, err
	}

	rawPublicInfo, err := a.getPublicInfo(ctx, vkTokens.IDToken)
	if err != nil {
		return nil, err
	}

	var publicInfo models.PublicInfo
	if err := json.Unmarshal(rawPublicInfo, &publicInfo); err != nil {
		l.UsecasesError(err, 0, nil)
		return nil, err
	}

	vkUser := publicInfo.User

	return &models.OAuthResult{
		ProviderUserID: vkUser.UserID,
		DisplayName:    fmt.Sprintf("%s %s", vkUser.FirstName, vkUser.LastName),
		AvatarURL:      vkUser.Avatar,
		Email:          vkUser.Email,
		AccessToken:    vkTokens.AccessToken,
		RefreshToken:   vkTokens.RefreshToken,
		IDToken:        vkTokens.IDToken,
	}, nil
}

func (a *vkAPI) exchangeCode(ctx context.Context, data *models.LoginRequest) ([]byte, error) {
	l := logger.FromContext(ctx)
	urlParams := url.Values{}

	urlParams.Set("grant_type", "authorization_code")
	urlParams.Set("code_verifier", data.CodeVerifier)
	urlParams.Set("redirect_uri", a.RedirectURI)
	urlParams.Set("code", data.Code)
	urlParams.Set("client_id", a.ClientID)
	urlParams.Set("device_id", data.DeviceID)
	urlParams.Set("state", data.State)

	req, err := http.NewRequest(a.Exchange.Method, a.Exchange.URL,
		strings.NewReader(urlParams.Encode()))
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}
	req.Header.Set("Content-Type", a.Exchange.ContentType)

	ctx = utils.SaveExternalRequestData(ctx, req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.VKApiError
	}
	defer resp.Body.Close()

	vkApiData, err := io.ReadAll(resp.Body)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}

	if resp.StatusCode != http.StatusOK {
		var vkErrData models.VKExchangeError

		err = json.Unmarshal(vkApiData, &vkErrData)
		if err != nil {
			l.ExternalError(ctx, err, nil)
			return nil, apperrors.ClientError
		}
		l.ExternalError(ctx, fmt.Errorf(vkErrData.Error),
			map[string]any{"desc": vkErrData.ErrorDescription, "code": resp.StatusCode})

		return nil, apperrors.VKApiError
	}

	return vkApiData, nil
}

func (a *vkAPI) getPublicInfo(ctx context.Context, idToken string) ([]byte, error) {
	l := logger.FromContext(ctx)
	urlParams := url.Values{}

	urlParams.Set("id_token", idToken)
	urlParams.Set("client_id", a.ClientID)

	req, err := http.NewRequest(a.PublicInfo.Method, a.PublicInfo.URL,
		strings.NewReader(urlParams.Encode()))
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}
	req.Header.Set("Content-Type", a.PublicInfo.ContentType)

	ctx = utils.SaveExternalRequestData(ctx, req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}
	defer resp.Body.Close()

	vkApiData, err := io.ReadAll(resp.Body)
	if err != nil {
		l.ExternalError(ctx, err, nil)
		return nil, apperrors.ClientError
	}

	if resp.StatusCode != http.StatusOK {
		var vkErrData models.VKInfoError

		err = json.Unmarshal(vkApiData, &vkErrData)
		if err != nil {
			l.ExternalError(ctx, err, nil)
			return nil, apperrors.ClientError
		}
		l.ExternalError(ctx, fmt.Errorf(vkErrData.Error),
			map[string]any{"desc": vkErrData.ErrorDescription, "state": vkErrData.State, "code": resp.StatusCode})

		return nil, apperrors.VKApiError
	}

	return vkApiData, nil
}
