package delivery

import (
	"encoding/json"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
	"io"
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
	l := logger.FromContext(ctx)

	var reqData models.GetEncountersListReq

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	list, err := h.usecases.GetEncountersList(ctx, reqData.Size, reqData.Start, userID, &reqData.Search)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.StartPosSizeError):
			code = responses.StatusBadRequest
			status = responses.ErrSizeOrPosition
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, reqData)
		responses.SendErrResponse(w, code, status)

		return
	}

	responses.SendOkResponse(w, list)
}

func (h *EncounterHandler) GetEncounterByID(w http.ResponseWriter, r *http.Request) {
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

	encounter, err := h.usecases.GetEncounterByID(ctx, id, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.PermissionDeniedError):
			code = responses.StatusForbidden
			status = responses.ErrForbidden
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"id": id, "user_id": userID})
		responses.SendErrResponse(w, code, status)

		return
	}

	responses.SendOkResponse(w, encounter)
}

func (h *EncounterHandler) SaveEncounter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var encounter models.SaveEncounterReq

	err := json.NewDecoder(r.Body).Decode(&encounter)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err = h.usecases.SaveEncounter(ctx, &encounter, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			code = responses.StatusBadRequest
			status = responses.ErrWrongEncounterName
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"user_id": userID})
		responses.SendErrResponse(w, code, status)

		return
	}

	responses.SendOkResponse(w, nil)
}

func (h *EncounterHandler) UpdateEncounter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err = h.usecases.UpdateEncounter(ctx, data, id, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.PermissionDeniedError):
			code = responses.StatusForbidden
			status = responses.ErrForbidden
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"id": id, "user_id": userID})
		responses.SendErrResponse(w, code, status)

		return
	}

	responses.SendOkResponse(w, nil)
}

func (h *EncounterHandler) RemoveEncounter(w http.ResponseWriter, r *http.Request) {
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

	err := h.usecases.RemoveEncounter(ctx, id, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.PermissionDeniedError):
			code = responses.StatusForbidden
			status = responses.ErrForbidden
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"id": id, "user_id": userID})
		responses.SendErrResponse(w, code, status)

		return
	}

	responses.SendOkResponse(w, nil)
}
