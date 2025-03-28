package delivery

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"log"
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type BestiaryHandler struct {
	usecases bestiaryinterface.BestiaryUsecases
}

func NewBestiaryHandler(usecases bestiaryinterface.BestiaryUsecases) *BestiaryHandler {
	return &BestiaryHandler{
		usecases: usecases,
	}
}

func (h *BestiaryHandler) GetCreaturesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqData models.BestiaryReq

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	list, err := h.usecases.GetCreaturesList(ctx, reqData.Size, reqData.Start, reqData.Order, reqData.Filter,
		reqData.Search)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			responses.SendOkResponse(w, nil)
		case errors.Is(err, apperrors.StartPosSizeError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrSizeOrPosition)
		default:
			log.Println(err)

			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, list)
}

func (h *BestiaryHandler) GetCreatureByName(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	creatureName := vars["name"]

	creature, err := h.usecases.GetCreatureByEngName(r.Context(), creatureName)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrCreatureNotFound)
		default:
			log.Println(err)

			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, creature)
}
