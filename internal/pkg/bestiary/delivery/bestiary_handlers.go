package delivery

import (
	"encoding/json"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type BestiaryHandler struct {
	usecases   bestiaryinterface.BestiaryUsecases
	ctxUserKey string
}

func NewBestiaryHandler(usecases bestiaryinterface.BestiaryUsecases, ctxUserKey string) *BestiaryHandler {
	return &BestiaryHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
	}
}

func (h *BestiaryHandler) GetCreaturesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var reqData models.BestiaryReq

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	list, err := h.usecases.GetCreaturesList(ctx, reqData.Size, reqData.Start, reqData.Order, reqData.Filter,
		reqData.Search)
	if err != nil {
		var status string
		var code int

		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			l.DeliveryInfo(ctx, "empty data", nil)
			responses.SendOkResponse(w, nil)
			return
		case errors.Is(err, apperrors.StartPosSizeError):
			code = responses.StatusBadRequest
			status = responses.ErrSizeOrPosition
		case errors.Is(err, apperrors.UnknownDirectionError):
			code = responses.StatusBadRequest
			status = responses.ErrWrongDirection
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

func (h *BestiaryHandler) GetCreatureByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	creatureName := vars["name"]

	creature, err := h.usecases.GetCreatureByEngName(ctx, creatureName)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	if creature == nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrCreatureNotFound, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrCreatureNotFound)

		return
	}

	responses.SendOkResponse(w, creature)
}

func (h *BestiaryHandler) GetUserCreaturesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var reqData models.BestiaryReq

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	list, err := h.usecases.GetUserCreaturesList(ctx, reqData.Size, reqData.Start, reqData.Order, reqData.Filter,
		reqData.Search, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.StartPosSizeError):
			code = responses.StatusBadRequest
			status = responses.ErrSizeOrPosition
		case errors.Is(err, apperrors.UnknownDirectionError):
			code = responses.StatusBadRequest
			status = responses.ErrWrongDirection
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

func (h *BestiaryHandler) GetUserCreatureByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	creatureName := vars["name"]

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	creature, err := h.usecases.GetUserCreatureByEngName(ctx, creatureName, userID)
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

		l.DeliveryError(ctx, code, status, err, map[string]string{"name": creatureName})
		responses.SendErrResponse(w, code, status)

		return
	}

	if creature == nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrCreatureNotFound, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrCreatureNotFound)

		return
	}

	responses.SendOkResponse(w, creature)
}

func (h *BestiaryHandler) AddGeneratedCreature(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var creatureInput models.CreatureInput

	err := json.NewDecoder(r.Body).Decode(&creatureInput)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err = h.usecases.AddGeneratedCreature(ctx, creatureInput, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidInputError):
			code = responses.StatusBadRequest
			status = responses.ErrInvalidID // NEED TO WRITE APPROPRIATE ERROR
		case errors.Is(err, apperrors.InvalidBase64Err):
			code = responses.StatusBadRequest
			status = responses.ErrWrongBase64
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, nil)
		responses.SendErrResponse(w, code, status)

		return
	}

	l.DeliveryInfo(ctx, "added generated creature", map[string]any{"user_id": userID, "id": creatureInput.ID})

	responses.SendOkResponse(w, nil)
}

func (h *BestiaryHandler) UploadCreatureStatblockImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrWrongFileSize, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongFileSize)

		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrWrongImage, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongImage)

		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	creature, err := h.usecases.ParseCreatureFromImage(ctx, imageBytes)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	responses.SendOkResponse(w, creature)
}

func (h *BestiaryHandler) SubmitCreatureGenerationPrompt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var input models.DescriptionGenPrompt

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	creature, err := h.usecases.GenerateCreatureFromDescription(r.Context(), input.Description)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	responses.SendOkResponse(w, creature)
}
