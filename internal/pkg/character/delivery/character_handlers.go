package delivery

import (
	"encoding/json"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type CharacterHandler struct {
	usecases   characterinterfaces.CharacterUsecases
	ctxUserKey string
}

func NewCharacterHandler(usecases characterinterfaces.CharacterUsecases, ctxUserKey string) *CharacterHandler {
	return &CharacterHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
	}
}

func (h *CharacterHandler) GetCharactersList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var reqData models.CharacterReq

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	list, err := h.usecases.GetCharactersList(ctx, reqData.Size, reqData.Start, userID, reqData.Search)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			l.DeliveryInfo(ctx, "empty data", nil)
			responses.SendOkResponse(w, nil)
			return
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

func (h *CharacterHandler) AddCharacter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	err := r.ParseMultipartForm(2 << 20)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrWrongFileSize, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongFileSize)

		return
	}

	file, header, err := r.FormFile("characterFile")
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}
	defer file.Close()

	if header.Header.Get("Content-Type") != "application/json" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrWrongFileType, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongFileType)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err = h.usecases.AddCharacter(ctx, file, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			code = responses.StatusBadRequest
			status = responses.ErrEmptyCharacterData
		case errors.Is(err, apperrors.InvalidJSONError):
			code = responses.StatusBadRequest
			status = responses.ErrBadJSON
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, nil)
		responses.SendErrResponse(w, code, status)

		return
	}

	l.DeliveryInfo(ctx, "added character from JSON", map[string]any{"user_id": userID})
	responses.SendOkResponse(w, nil)
}

func (h *CharacterHandler) GetCharacterByMongoId(w http.ResponseWriter, r *http.Request) {
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

	character, err := h.usecases.GetCharacterByMongoId(ctx, id, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			code = responses.StatusBadRequest
			status = responses.ErrInvalidID
		case errors.Is(err, apperrors.PermissionDeniedError):
			code = responses.StatusForbidden
			status = responses.ErrForbidden
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, nil)
		responses.SendErrResponse(w, code, status)

		return
	}

	if character == nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrCharacterNotFound, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrCharacterNotFound)

		return
	}

	responses.SendOkResponse(w, character)
}
