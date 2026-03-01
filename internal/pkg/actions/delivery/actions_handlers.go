package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
)

type ActionsHandler struct {
	usecases   actionsinterfaces.ActionsUsecases
	ctxUserKey string
}

func NewActionsHandler(usecases actionsinterfaces.ActionsUsecases, ctxUserKey string) *ActionsHandler {
	return &ActionsHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
	}
}

func (h *ActionsHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	var req models.ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	result, err := h.usecases.ExecuteAction(ctx, id, &req, userID)
	if err != nil {
		code, status := h.mapError(err)
		l.DeliveryError(ctx, code, status, err, map[string]any{
			"encounter_id": id,
			"user_id":      userID,
			"action_type":  req.Action.Type,
		})
		responses.SendErrResponse(w, code, status)

		return
	}

	responses.SendOkResponse(w, result)
}

func (h *ActionsHandler) GetActionLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var before time.Time
	if beforeStr := r.URL.Query().Get("before"); beforeStr != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, beforeStr); err == nil {
			before = parsed
		}
	}

	entries, err := h.usecases.GetActionLog(ctx, id, userID, limit, before)
	if err != nil {
		code, status := h.mapError(err)
		l.DeliveryError(ctx, code, status, err, map[string]any{
			"encounter_id": id,
			"user_id":      userID,
		})
		responses.SendErrResponse(w, code, status)

		return
	}

	responses.SendOkResponse(w, entries)
}

func (h *ActionsHandler) mapError(err error) (int, string) {
	switch {
	case errors.Is(err, apperrors.InvalidActionTypeErr),
		errors.Is(err, apperrors.MissingAbilityErr),
		errors.Is(err, apperrors.MissingDiceExprErr),
		errors.Is(err, apperrors.InvalidDiceExprErr),
		errors.Is(err, apperrors.MissingCharacterIDErr),
		errors.Is(err, apperrors.MissingWeaponIDErr),
		errors.Is(err, apperrors.MissingSpellIDErr),
		errors.Is(err, apperrors.MissingFeatureIDErr),
		errors.Is(err, apperrors.InsufficientSlotsErr),
		errors.Is(err, apperrors.FeatureUsesExhaustedErr),
		errors.Is(err, apperrors.InvalidIDErr):
		return responses.StatusBadRequest, responses.ErrBadRequest

	case errors.Is(err, apperrors.PermissionDeniedError):
		return responses.StatusForbidden, responses.ErrForbidden

	case errors.Is(err, apperrors.ParticipantNotFoundErr),
		errors.Is(err, apperrors.WeaponNotFoundErr),
		errors.Is(err, apperrors.SpellNotKnownErr),
		errors.Is(err, apperrors.FeatureNotFoundErr),
		errors.Is(err, apperrors.NoDocsErr):
		return responses.StatusNotFound, responses.ErrNotFound

	default:
		return responses.StatusInternalServerError, responses.ErrInternalServer
	}
}
