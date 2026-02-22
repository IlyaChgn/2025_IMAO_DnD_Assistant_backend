package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
)

// aiTurnRequest is the JSON body for POST /encounter/{id}/ai-turn.
type aiTurnRequest struct {
	NpcID string `json:"npcID"`
}

// CombatAIHandler handles HTTP requests for AI-driven NPC turns.
type CombatAIHandler struct {
	usecases   combatai.CombatAIUsecases
	ctxUserKey string
}

// NewCombatAIHandler creates a new CombatAIHandler.
func NewCombatAIHandler(usecases combatai.CombatAIUsecases, ctxUserKey string) *CombatAIHandler {
	return &CombatAIHandler{usecases: usecases, ctxUserKey: ctxUserKey}
}

// AITurn handles POST /encounter/{id}/ai-turn.
func (h *CombatAIHandler) AITurn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	// 1. Parse encounter ID from URL.
	vars := mux.Vars(r)
	encounterID, ok := vars["id"]
	if !ok || encounterID == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		return
	}

	// 2. Decode request body.
	var req aiTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}
	if req.NpcID == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadRequest, nil,
			map[string]any{"reason": "npcID is required"})
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadRequest)
		return
	}

	// 3. Extract user from context.
	user := ctx.Value(h.ctxUserKey).(*models.User)

	// 4. Execute AI turn.
	result, err := h.usecases.ExecuteAITurn(ctx, encounterID, req.NpcID, user.ID)
	if err != nil {
		code, status := mapAITurnError(err)
		l.DeliveryError(ctx, code, status, err, map[string]any{
			"encounterID": encounterID,
			"npcID":       req.NpcID,
			"userID":      user.ID,
		})
		responses.SendErrResponse(w, code, status)
		return
	}

	// 5. Send success response.
	responses.SendOkResponse(w, result)
}

// AIRound handles POST /encounter/{id}/ai-round.
// Executes all NPC turns in initiative order for the current round.
func (h *CombatAIHandler) AIRound(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	// 1. Parse encounter ID from URL.
	vars := mux.Vars(r)
	encounterID, ok := vars["id"]
	if !ok || encounterID == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		return
	}

	// 2. Extract user from context.
	user := ctx.Value(h.ctxUserKey).(*models.User)

	// 3. Execute AI round.
	result, err := h.usecases.ExecuteAIRound(ctx, encounterID, user.ID)
	if err != nil {
		code, status := mapAITurnError(err)
		l.DeliveryError(ctx, code, status, err, map[string]any{
			"encounterID": encounterID,
			"userID":      user.ID,
		})
		responses.SendErrResponse(w, code, status)
		return
	}

	// 4. Send success response.
	responses.SendOkResponse(w, result)
}

// mapAITurnError maps usecase errors to HTTP status codes.
func mapAITurnError(err error) (int, string) {
	switch {
	case errors.Is(err, apperrors.PermissionDeniedError):
		return responses.StatusForbidden, responses.ErrForbidden
	case errors.Is(err, apperrors.ParticipantNotFoundErr):
		return responses.StatusNotFound, responses.ErrNotFound
	case errors.Is(err, apperrors.NPCIsPlayerCharacterErr),
		errors.Is(err, apperrors.NPCIsDeadErr):
		return responses.StatusBadRequest, responses.ErrBadRequest
	default:
		return responses.StatusInternalServerError, responses.ErrInternalServer
	}
}
