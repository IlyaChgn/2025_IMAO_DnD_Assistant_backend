package delivery

import (
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/config"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/google/uuid"
	"log"
	"net/http"
	"time"
)

type AuthHandler struct {
	usecases        authinterface.AuthUsecases
	vkApiCfg        *config.VKApiConfig
	sessionDuration time.Duration
}

func NewAuthHandler(usecases authinterface.AuthUsecases, vkApiCfg *config.VKApiConfig) *AuthHandler {
	return &AuthHandler{
		usecases:        usecases,
		vkApiCfg:        vkApiCfg,
		sessionDuration: 30 * 24 * time.Hour,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqData models.LoginRequest

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	vkRawData, err := h.exchangeCode(w, &reqData)
	if err != nil {
		return
	}

	var vkTokens models.VKTokensData

	err = json.Unmarshal(vkRawData, &vkTokens)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	rawPublicInfo, err := h.getPublicInfo(w, vkTokens.IDToken)
	if err != nil {
		return
	}

	var publicInfo models.PublicInfo

	err = json.Unmarshal(rawPublicInfo, &publicInfo)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	sessionID := uuid.NewString()

	user, err := h.usecases.Login(ctx, sessionID, &publicInfo.User, &vkTokens, h.sessionDuration)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	newSession := h.createSession(sessionID)
	http.SetCookie(w, newSession)

	responses.SendOkResponse(w, &models.AuthResponse{
		IsAuth: true,
		User:   *user,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, _ := r.Cookie("session_id")

	err := h.usecases.Logout(ctx, session.Value)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	session.Expires = time.Now().AddDate(0, 0, -1)
	http.SetCookie(w, session)

	responses.SendOkResponse(w, &models.AuthResponse{IsAuth: false})
}

func (h *AuthHandler) CheckAuth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, _ := r.Cookie("session_id")
	if session == nil {
		responses.SendOkResponse(w, &models.AuthResponse{IsAuth: false})

		return
	}

	user, isAuth := h.usecases.CheckAuth(ctx, session.Value)

	if !isAuth {
		responses.SendOkResponse(w, &models.AuthResponse{IsAuth: false})

		return
	}

	responses.SendOkResponse(w, &models.AuthResponse{IsAuth: true, User: *user})
}
