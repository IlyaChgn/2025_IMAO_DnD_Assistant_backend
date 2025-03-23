package delivery

import (
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"net/http"

	"github.com/gorilla/mux"

	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
	responses "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type CreatureHandler struct {
	usecases creatureinterfaces.CreatureUsecases
}

func NewCreatureHandler(usecases creatureinterfaces.CreatureUsecases) *CreatureHandler {
	return &CreatureHandler{
		usecases: usecases,
	}
}

func (h *CreatureHandler) GetCreatureByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatureName := vars["name"]

	creature, err := h.usecases.GetCreatureByEngName(r.Context(), creatureName)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrCreatureNotFound)
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, creature)
}
