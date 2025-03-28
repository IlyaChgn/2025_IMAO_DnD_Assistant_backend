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
	usecases characterinterfaces.CharacterUsecases
}

func NewCharacterHandler(usecases characterinterfaces.CharacterUsecases) *CharacterHandler {
	return &CharacterHandler{
		usecases: usecases,
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

	list, err := h.usecases.GetCharactersList(ctx, reqData.Size, reqData.Start, reqData.Order, reqData.Filter,
		reqData.Search)
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

	err := r.ParseMultipartForm(10 << 20)
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

	err = h.usecases.AddCharacter(ctx, file)
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

	character, err := h.usecases.GetCharacterByMongoId(ctx, id)
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

	responses.SendOkResponse(w, character)
}
