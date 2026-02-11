package delivery

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type CharacterBaseHandler struct {
	usecases   characterinterfaces.CharacterBaseUsecases
	ctxUserKey string
}

func NewCharacterBaseHandler(usecases characterinterfaces.CharacterBaseUsecases, ctxUserKey string) *CharacterBaseHandler {
	return &CharacterBaseHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
	}
}

func (h *CharacterBaseHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	id := mux.Vars(r)["id"]
	if id == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	char, err := h.usecases.GetByID(ctx, id, user.ID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	if char == nil {
		l.DeliveryError(ctx, responses.StatusNotFound, responses.ErrCharacterNotFound, nil, nil)
		responses.SendErrResponse(w, responses.StatusNotFound, responses.ErrCharacterNotFound)
		return
	}

	responses.SendOkResponse(w, char)
}

func (h *CharacterBaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var char models.CharacterBase
	if err := json.NewDecoder(r.Body).Decode(&char); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	char.UserID = strconv.Itoa(user.ID)

	if err := h.usecases.Create(ctx, &char); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(char)
}

type updateRequest struct {
	Character models.CharacterBase `json:"character"`
	Version   int                  `json:"version"`
}

func (h *CharacterBaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	id := mux.Vars(r)["id"]
	if id == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		return
	}

	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	if err := h.usecases.Update(ctx, &req.Character, req.Version, user.ID); err != nil {
		h.handleError(w, r, err)
		return
	}

	responses.SendOkResponse(w, req.Character)
}

func (h *CharacterBaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	id := mux.Vars(r)["id"]
	if id == "" {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	if err := h.usecases.Delete(ctx, id, user.ID); err != nil {
		h.handleError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CharacterBaseHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	user := ctx.Value(h.ctxUserKey).(*models.User)

	pageStr := r.URL.Query().Get("page")
	sizeStr := r.URL.Query().Get("size")
	search := r.URL.Query().Get("search")

	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)
	if size <= 0 {
		size = 20
	}

	chars, total, err := h.usecases.List(ctx, user.ID, page, size, search)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	type listResponse struct {
		Characters []*models.CharacterBase `json:"characters"`
		Total      int64                   `json:"total"`
		Page       int                     `json:"page"`
		Size       int                     `json:"size"`
	}

	if chars == nil {
		chars = []*models.CharacterBase{}
	}

	resp := listResponse{
		Characters: chars,
		Total:      total,
		Page:       page,
		Size:       size,
	}

	l.DeliveryInfo(ctx, "listed characters", map[string]any{"userId": user.ID, "total": total})
	responses.SendOkResponse(w, resp)
}

func (h *CharacterBaseHandler) ImportLSS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrWrongFileSize, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongFileSize)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, "File upload required", err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, "File upload required")
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	char, report, err := h.usecases.ImportLSS(ctx, fileData, user.ID)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	type importResponse struct {
		Character *models.CharacterBase   `json:"character"`
		Report    *models.ConversionReport `json:"report"`
	}

	resp := importResponse{
		Character: char,
		Report:    report,
	}

	l.DeliveryInfo(ctx, "imported LSS character", map[string]any{"userId": user.ID, "charName": char.Name})
	responses.SendOkResponse(w, resp)
}

func (h *CharacterBaseHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var code int
	var status string

	switch {
	case errors.Is(err, apperrors.InvalidInputError) || errors.Is(err, apperrors.InvalidIDErr):
		code = responses.StatusBadRequest
		status = responses.ErrInvalidID
	case errors.Is(err, apperrors.PermissionDeniedError):
		code = responses.StatusForbidden
		status = responses.ErrForbidden
	case errors.Is(err, apperrors.VersionConflictErr):
		code = http.StatusConflict
		status = "Version conflict: document was modified"
	case errors.Is(err, apperrors.ConversionFailedError):
		code = responses.StatusBadRequest
		status = "LSS conversion failed"
	case errors.Is(err, apperrors.InvalidJSONError):
		code = responses.StatusBadRequest
		status = responses.ErrBadJSON
	default:
		code = responses.StatusInternalServerError
		status = responses.ErrInternalServer
	}

	l.DeliveryError(ctx, code, status, err, nil)
	responses.SendErrResponse(w, code, status)
}
