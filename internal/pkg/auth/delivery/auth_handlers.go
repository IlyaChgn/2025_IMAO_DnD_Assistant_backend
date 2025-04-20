package delivery

import (
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/config"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"log"
	"net/http"
)

type AuthHandler struct {
	usecases authinterface.AuthUsecases
	vkApiCfg *config.VKApiConfig
}

func NewAuthHandler(usecases authinterface.AuthUsecases, vkApiCfg *config.VKApiConfig) *AuthHandler {
	return &AuthHandler{
		usecases: usecases,
		vkApiCfg: vkApiCfg,
	}
}

func (h *AuthHandler) Exchange(w http.ResponseWriter, r *http.Request) {
	var reqData models.CodeExchangeRequest
	log.Println("Exchanging")

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	vkRawData, err := h.exchangeCode(&reqData)
	if err != nil {
		log.Println(err)

		return
	}

	var vkData models.VKTokensData
	err = json.Unmarshal(vkRawData, &vkData)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	rawPublicInfo, err := h.getPublicInfo(vkData.IDToken)
	if err != nil {
		log.Println(err)

		return
	}

	var publicInfo models.PublicInfo
	err = json.Unmarshal(rawPublicInfo, &publicInfo)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	responses.SendOkResponse(w, publicInfo)
}
