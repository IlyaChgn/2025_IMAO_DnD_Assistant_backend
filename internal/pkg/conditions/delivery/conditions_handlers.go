package delivery

import (
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/conditions"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
)

type ConditionsHandler struct{}

func NewConditionsHandler() *ConditionsHandler {
	return &ConditionsHandler{}
}

// GetConditions handles GET /api/reference/conditions
func (h *ConditionsHandler) GetConditions(w http.ResponseWriter, _ *http.Request) {
	responses.SendOkResponse(w, conditions.AllConditions())
}

// GetConditionByType handles GET /api/reference/conditions/{type}
func (h *ConditionsHandler) GetConditionByType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	condType := models.ConditionType(vars["type"])

	cond := conditions.FindByType(condType)
	if cond == nil {
		l.DeliveryError(ctx, responses.StatusNotFound, responses.ErrNotFound, nil, map[string]any{"type": string(condType)})
		responses.SendErrResponse(w, responses.StatusNotFound, responses.ErrNotFound)

		return
	}

	responses.SendOkResponse(w, cond)
}
