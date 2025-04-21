package delivery

import (
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// exchangeCode возвращает access token, refresh token и ID token в обмен на код верификации с клиента
func (h *AuthHandler) exchangeCode(w http.ResponseWriter, data *models.LoginRequest) ([]byte, error) {
	urlParams := url.Values{}

	urlParams.Set("grant_type", "authorization_code")
	urlParams.Set("code_verifier", data.CodeVerifier)
	urlParams.Set("redirect_uri", h.vkApiCfg.RedirectURI)
	urlParams.Set("code", data.Code)
	urlParams.Set("client_id", h.vkApiCfg.ClientID)
	urlParams.Set("device_id", data.DeviceID)
	urlParams.Set("state", data.State)

	req, err := http.NewRequest(h.vkApiCfg.Exchange.Method, h.vkApiCfg.Exchange.URL,
		strings.NewReader(urlParams.Encode()))
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrVKServer)

		return nil, err
	}
	defer resp.Body.Close()

	vkApiData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var vkErrData models.VKExchangeError

		err = json.Unmarshal(vkApiData, &vkErrData)
		if err != nil {
			log.Println(err)
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

			return nil, err
		}

		log.Println(vkErrData.Error, vkErrData.ErrorDescription)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrVKServer)

		return nil, err
	}

	return vkApiData, nil
}

func (h *AuthHandler) getPublicInfo(w http.ResponseWriter, idToken string) ([]byte, error) {
	urlParams := url.Values{}

	urlParams.Set("id_token", idToken)
	urlParams.Set("client_id", h.vkApiCfg.ClientID)

	req, err := http.NewRequest(h.vkApiCfg.PublicInfo.Method, h.vkApiCfg.PublicInfo.URL,
		strings.NewReader(urlParams.Encode()))
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return nil, err
	}
	defer resp.Body.Close()

	vkApiData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var vkErrData models.VKInfoError

		err = json.Unmarshal(vkApiData, &vkErrData)
		if err != nil {
			log.Println(err)
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

			return nil, err
		}

		log.Println(vkErrData.Error, vkErrData.ErrorDescription)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrVKServer)

		return nil, err
	}

	return vkApiData, nil
}
