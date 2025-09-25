package delivery

import (
	"encoding/json"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type AuthHandler struct {
	usecases        authinterface.AuthUsecases
	sessionDuration time.Duration
}

func NewAuthHandler(usecases authinterface.AuthUsecases) *AuthHandler {
	return &AuthHandler{
		usecases:        usecases,
		sessionDuration: 30 * 24 * time.Hour,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var reqData models.LoginRequest

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	sessionID := uuid.NewString()

	user, err := h.usecases.Login(ctx, sessionID, &reqData, h.sessionDuration)
	if err != nil {
		var status string

		switch {
		case errors.Is(err, apperrors.VKApiError):
			status = responses.ErrVKServer
		default:
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, responses.StatusInternalServerError, status, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, status)

		return
	}

	newSession := h.createSession(sessionID)
	http.SetCookie(w, newSession)
	l.DeliveryInfo(ctx, "user authorized", map[string]any{"session_id": sessionID, "user": user.Name})
	responses.SendOkResponse(w, &models.AuthResponse{
		IsAuth: true,
		User:   *user,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)
	session, _ := r.Cookie("session_id")

	err := h.usecases.Logout(ctx, session.Value)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err,
			map[string]any{"session_id": session.Value})
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	session.Expires = time.Now().AddDate(0, 0, -1)
	http.SetCookie(w, session)
	l.DeliveryInfo(ctx, "user logged out", map[string]any{"session_id": session.Value})
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
