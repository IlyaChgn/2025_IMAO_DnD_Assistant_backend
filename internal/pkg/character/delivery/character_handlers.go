package delivery

import (
	"encoding/json"
	"errors"
	"io"
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

	list, err := h.usecases.GetCharactersList(ctx, reqData.Size, reqData.Start, reqData.Order, reqData.Filter, reqData.Search)
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

	// Ограничиваем размер файла (например, 10 МБ)
	r.ParseMultipartForm(10 << 20) // 10 MB

	// Получаем файл из формы
	file, handler, err := r.FormFile("characterFile") // "characterFile" — это имя поля в форме
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, "Failed to retrieve the file")
		return
	}
	defer file.Close()

	// Проверяем, что файл имеет допустимый тип (например, JSON)
	if handler.Header.Get("Content-Type") != "application/json" {
		responses.SendErrResponse(w, responses.StatusBadRequest, "Invalid file type. Only JSON files are allowed")
		return
	}

	// Читаем содержимое файла
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusInternalServerError, "Failed to read the file")
		return
	}

	// Парсим JSON в структуру CharacterRaw
	var rawChar models.CharacterRaw
	err = json.Unmarshal(fileBytes, &rawChar)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Проверяем, что данные персонажа не пустые
	if rawChar.Data == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, "Character data is empty")
		return
	}

	// Вызываем usecase для добавления персонажа
	err = h.usecases.AddCharacter(ctx, rawChar)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			responses.SendErrResponse(w, responses.StatusBadRequest, "Invalid character data")
		case errors.Is(err, apperrors.InsertMongoDataErr):
			responses.SendErrResponse(w, responses.StatusInternalServerError, "Failed to insert character into the database")
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, "Internal server error")
		}
		return
	}

	// Отправляем успешный ответ
	responses.SendOkResponse(w, "Character added successfully")
}

func (h *CharacterHandler) GetCharacterByMongoId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Извлечение id из Path-параметра
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok || id == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrDataNotValid)
		return
	}

	// Получение персонажа по id
	character, err := h.usecases.GetCharacterByMongoId(ctx, id)
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
	responses.SendOkResponse(w, character)
}
