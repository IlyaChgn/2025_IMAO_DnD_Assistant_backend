package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

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

func (h *ActionsHandler) mapError(err error) (int, string) {
	switch {
	case errors.Is(err, apperrors.InvalidActionTypeErr),
		errors.Is(err, apperrors.MissingAbilityErr),
		errors.Is(err, apperrors.MissingDCErr),
		errors.Is(err, apperrors.MissingDiceExprErr),
		errors.Is(err, apperrors.InvalidDiceExprErr),
		errors.Is(err, apperrors.MissingCharacterIDErr),
		errors.Is(err, apperrors.MissingWeaponIDErr),
		errors.Is(err, apperrors.MissingSpellIDErr),
		errors.Is(err, apperrors.MissingFeatureIDErr),
		errors.Is(err, apperrors.InsufficientSlotsErr),
		errors.Is(err, apperrors.FeatureUsesExhaustedErr):
		return responses.StatusBadRequest, err.Error()

	case errors.Is(err, apperrors.PermissionDeniedError):
		return responses.StatusForbidden, responses.ErrForbidden

	case errors.Is(err, apperrors.ParticipantNotFoundErr),
		errors.Is(err, apperrors.EncounterNotFoundErr),
		errors.Is(err, apperrors.WeaponNotFoundErr),
		errors.Is(err, apperrors.SpellNotKnownErr),
		errors.Is(err, apperrors.FeatureNotFoundErr):
		return responses.StatusNotFound, err.Error()

	default:
		return responses.StatusInternalServerError, responses.ErrInternalServer
	}
}
