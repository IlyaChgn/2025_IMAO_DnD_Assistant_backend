package external

import (
	"context"
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/config"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
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
	exchange, publicInfo config.VKMethodConfig) authinterface.VKApi {
	return &vkAPI{
		RedirectURI: redirectURI,
		ClientID:    clientID,
		SecretKey:   secretKey,
		ServiceKey:  serviceKey,
		Exchange:    exchange,
		PublicInfo:  publicInfo,
	}
}

// ExchangeCode возвращает access token, refresh token и ID token в обмен на код верификации с клиента
func (a *vkAPI) ExchangeCode(ctx context.Context, data *models.LoginRequest) ([]byte, error) {
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
		return nil, apperrors.ClientError
	}
	req.Header.Set("Content-Type", a.Exchange.ContentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.VKApiError
	}
	defer resp.Body.Close()

	vkApiData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.ClientError
	}

	if resp.StatusCode != http.StatusOK {
		var vkErrData models.VKExchangeError

		err = json.Unmarshal(vkApiData, &vkErrData)
		if err != nil {
			return nil, apperrors.ClientError
		}

		log.Println(vkErrData.Error, vkErrData.ErrorDescription)

		return nil, apperrors.VKApiError
	}

	return vkApiData, nil
}

func (a *vkAPI) GetPublicInfo(ctx context.Context, idToken string) ([]byte, error) {
	urlParams := url.Values{}

	urlParams.Set("id_token", idToken)
	urlParams.Set("client_id", a.ClientID)

	req, err := http.NewRequest(a.PublicInfo.Method, a.PublicInfo.URL,
		strings.NewReader(urlParams.Encode()))
	if err != nil {
		return nil, apperrors.ClientError
	}
	req.Header.Set("Content-Type", a.PublicInfo.ContentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.ClientError
	}
	defer resp.Body.Close()

	vkApiData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.ClientError
	}

	if resp.StatusCode != http.StatusOK {
		var vkErrData models.VKInfoError

		err = json.Unmarshal(vkApiData, &vkErrData)
		if err != nil {
			return nil, apperrors.ClientError
		}

		log.Println(vkErrData.Error, vkErrData.ErrorDescription)

		return nil, apperrors.VKApiError
	}

	return vkApiData, nil
}
