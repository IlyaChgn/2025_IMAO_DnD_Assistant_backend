package delivery

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"

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
		case errors.Is(err, apperrors.UnknownDirectionError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongDirection)
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

func (h *BestiaryHandler) AddGeneratedCreature(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var creatureInput models.CreatureInput

	err := json.NewDecoder(r.Body).Decode(&creatureInput)
	if err != nil {
		log.Println("JSON decode error:", err)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	err = h.usecases.AddGeneratedCreature(ctx, creatureInput)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID) // NEED TO WRITE APROPRIATE ERROR
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}
		return
	}

	responses.SendOkResponse(w, nil)
}

func (h *BestiaryHandler) UploadCreatureStatblockImage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Println("Ошибка при парсинге формы:", err)
		responses.SendErrResponse(w, responses.StatusBadRequest, "Invalid form data")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		log.Println("Ошибка получения файла:", err)
		responses.SendErrResponse(w, responses.StatusBadRequest, "Image not provided")
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		log.Println("Ошибка чтения файла:", err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, "Failed to read image")
		return
	}

	creature, err := h.usecases.ParseCreatureFromImage(r.Context(), imageBytes)
	if err != nil {
		log.Println("Ошибка при вызове AI:", err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, "AI error")
		return
	}

	responses.SendOkResponse(w, creature)
}

func (h *BestiaryHandler) SubmitCreatureGenerationPrompt(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Description string `json:"description"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		log.Println("Ошибка декодирования JSON:", err)
		responses.SendErrResponse(w, responses.StatusBadRequest, "Invalid JSON")
		return
	}

	log.Println("Получено описание существа:", input.Description)

	creature, err := h.usecases.GenerateCreatureFromDescription(r.Context(), input.Description)
	if err != nil {
		log.Println("Ошибка генерации существа:", err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, "AI generation failed")
		return
	}

	responses.SendOkResponse(w, creature)
}
