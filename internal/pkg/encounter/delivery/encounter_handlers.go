package delivery

import (
	"encoding/json"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
)

type EncounterHandler struct {
	usecases   encounterinterfaces.EncounterUsecases
	ctxUserKey string
}

func NewEncounterHandler(usecases encounterinterfaces.EncounterUsecases, sessionKey string) *EncounterHandler {
	return &EncounterHandler{
		usecases:   usecases,
		ctxUserKey: sessionKey,
	}
}

func (h *EncounterHandler) GetEncountersList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqData models.GetEncountersListReq

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	list, err := h.usecases.GetEncountersList(ctx, reqData.Size, reqData.Start, userID, &reqData.Search)
	if err != nil {
		switch {
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

func (h *EncounterHandler) GetEncounterByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	encounter, err := h.usecases.GetEncounterByID(ctx, id, userID)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.PermissionDeniedError):
			responses.SendErrResponse(w, responses.StatusForbidden, responses.ErrForbidden)
		default:
			log.Println(err)

			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, encounter)
}

func (h *EncounterHandler) SaveEncounter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var encounter models.SaveEncounterReq

	err := json.NewDecoder(r.Body).Decode(&encounter)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err = h.usecases.SaveEncounter(ctx, &encounter, userID)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongEncounterName)
		default:
			log.Println(err)

			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, nil)
}

func (h *EncounterHandler) UpdateEncounter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err = h.usecases.UpdateEncounter(ctx, data, id, userID)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.PermissionDeniedError):
			responses.SendErrResponse(w, responses.StatusForbidden, responses.ErrForbidden)
		default:
			log.Println(err)

			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, nil)
}

func (h *EncounterHandler) RemoveEncounter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err := h.usecases.RemoveEncounter(ctx, id, userID)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.PermissionDeniedError):
			responses.SendErrResponse(w, responses.StatusForbidden, responses.ErrForbidden)
		default:
			log.Println(err)

			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, nil)
}
