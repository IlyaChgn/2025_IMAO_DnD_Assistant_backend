package delivery

import (
	"encoding/json"
	"errors"
	"log"
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

	var reqData models.CharacterReq

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	list, err := h.usecases.GetCharactersList(ctx, reqData.Size, reqData.Start, userID, reqData.Search)
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

func (h *CharacterHandler) AddCharacter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := r.ParseMultipartForm(2 << 20)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongFileSize)

		return
	}

	file, header, err := r.FormFile("characterFile")
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}
	defer file.Close()

	if header.Header.Get("Content-Type") != "application/json" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongFileType)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err = h.usecases.AddCharacter(ctx, file, userID)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrEmptyCharacterData)
		case errors.Is(err, apperrors.InvalidJSONError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, nil)
}

func (h *CharacterHandler) GetCharacterByMongoId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	character, err := h.usecases.GetCharacterByMongoId(ctx, id, userID)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		case errors.Is(err, apperrors.PermissionDeniedError):
			responses.SendErrResponse(w, responses.StatusForbidden, responses.ErrForbidden)
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	if character == nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrCharacterNotFound)
	}

	responses.SendOkResponse(w, character)
}
