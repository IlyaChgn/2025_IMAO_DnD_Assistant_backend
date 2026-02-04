package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type AuthHandler struct {
	usecases        authinterface.AuthUsecases
	sessionDuration time.Duration
	isProd          bool
	ctxUserKey      string
}

func NewAuthHandler(usecases authinterface.AuthUsecases, sessionDuration time.Duration,
	isProd bool, ctxUserKey string) *AuthHandler {
	return &AuthHandler{
		usecases:        usecases,
		sessionDuration: sessionDuration,
		isProd:          isProd,
		ctxUserKey:      ctxUserKey,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	provider := mux.Vars(r)["provider"]

	var reqData models.LoginRequest

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	sessionID := uuid.NewString()

	user, err := h.usecases.Login(ctx, provider, sessionID, &reqData, h.sessionDuration)
	if err != nil {
		var status string

		switch {
		case errors.Is(err, apperrors.UnsupportedProviderError):
			l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadRequest, err, nil)
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadRequest)
			return
		case errors.Is(err, apperrors.VKApiError):
			status = responses.ErrVKServer
		case errors.Is(err, apperrors.OAuthProviderError):
			status = responses.ErrOAuthProvider
		default:
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, responses.StatusInternalServerError, status, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, status)

		return
	}

	newSession := h.createSession(sessionID)
	http.SetCookie(w, newSession)
	l.DeliveryInfo(ctx, "user authorized", map[string]any{"session_id": sessionID, "user": user.DisplayName, "provider": provider})
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

	http.SetCookie(w, h.clearSession())
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

func (h *AuthHandler) ListIdentities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	user := ctx.Value(h.ctxUserKey).(*models.User)

	identities, err := h.usecases.ListIdentities(ctx, user.ID)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	responses.SendOkResponse(w, identities)
}

func (h *AuthHandler) LinkIdentity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	user := ctx.Value(h.ctxUserKey).(*models.User)
	provider := mux.Vars(r)["provider"]

	var reqData models.LoginRequest

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	err = h.usecases.LinkIdentity(ctx, user.ID, provider, &reqData)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.UnsupportedProviderError):
			l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadRequest, err, nil)
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadRequest)
		case errors.Is(err, apperrors.IdentityAlreadyLinkedError):
			l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrIdentityAlreadyLinked, err, nil)
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrIdentityAlreadyLinked)
		default:
			l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) UnlinkIdentity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	user := ctx.Value(h.ctxUserKey).(*models.User)
	provider := mux.Vars(r)["provider"]

	err := h.usecases.UnlinkIdentity(ctx, user.ID, provider)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.LastIdentityError):
			l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrLastIdentity, err, nil)
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrLastIdentity)
		case errors.Is(err, apperrors.IdentityNotFoundError):
			l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadRequest, err, nil)
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadRequest)
		default:
			l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
