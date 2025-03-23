package delivery

import (
	"encoding/json"
	"errors"
	"io"
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

// GetEncountersList обрабатывает запрос на получение списка энкаунтеров
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

// AddEncounter обрабатывает запрос на добавление нового энкаунтера
func (h *EncounterHandler) AddEncounter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Парсим JSON в структуру Encounter
	var encounter models.EncounterRaw
	err = json.Unmarshal(body, &encounter)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Проверяем, что данные энкаунтера не пустые
	if encounter.EncounterName == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, "Encounter name is required")
		return
	}

	// Вызываем usecase для добавления энкаунтера
	err = h.usecases.AddEncounter(ctx, encounter)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, "Invalid encounter data")
		case errors.Is(err, apperrors.InsertMongoDataErr):
			responses.SendErrResponse(w, responses.StatusInternalServerError, "Failed to insert encounter into the database")
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, "Internal server error")
		}
		return
	}

	// Отправляем успешный ответ
	responses.SendOkResponse(w, "Encounter added successfully")
}

// GetEncounterByMongoId обрабатывает запрос на получение энкаунтера по его ID
func (h *EncounterHandler) GetEncounterByMongoId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлечение id из Path-параметра
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok || id == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrDataNotValid)
		return
	}

	// Получение энкаунтера по id
	encounter, err := h.usecases.GetEncounterByMongoId(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrDataNotValid)
		case errors.Is(err, apperrors.NoDocsErr):
			responses.SendOkResponse(w, nil)
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}
		return
	}

	// Отправка успешного ответа
	responses.SendOkResponse(w, encounter)
}
