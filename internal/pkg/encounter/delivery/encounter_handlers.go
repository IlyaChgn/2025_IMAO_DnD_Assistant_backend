package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type EncounterHandler struct {
	usecases encounterinterfaces.EncounterUsecases
}

func NewEncounterHandler(usecases encounterinterfaces.EncounterUsecases) *EncounterHandler {
	return &EncounterHandler{
		usecases: usecases,
	}
}

func (h *EncounterHandler) GetEncountersList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqData models.EncounterReq

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	list, err := h.usecases.GetEncountersList(ctx, reqData.Size, reqData.Start, reqData.Order, reqData.Filter, reqData.Search)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			responses.SendOkResponse(w, nil)
		case errors.Is(err, apperrors.StartPosSizeError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrSizeOrPosition)
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}
		return
	}

	responses.SendOkResponse(w, list)
}

func (h *EncounterHandler) AddEncounter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var encounter models.EncounterRaw

	err := json.NewDecoder(r.Body).Decode(&encounter)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	err = h.usecases.AddEncounter(ctx, encounter)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrEmptyEncounterName)
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, nil)
}

func (h *EncounterHandler) GetEncounterByMongoId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		return
	}

	encounter, err := h.usecases.GetEncounterByMongoId(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		case errors.Is(err, apperrors.NoDocsErr):
			responses.SendOkResponse(w, nil)
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}
		return
	}

	responses.SendOkResponse(w, encounter)
}
