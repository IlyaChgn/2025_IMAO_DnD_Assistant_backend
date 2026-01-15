package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mapsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps/usecases"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	DefaultStart   = 0
	DefaultSize    = 20
	MaxSize        = 100
	MaxRequestBody = 10 * 1024 * 1024 // 10MB
)

type MapsHandler struct {
	usecases   mapsinterfaces.MapsUsecases
	ctxUserKey string
}

func NewMapsHandler(usecases mapsinterfaces.MapsUsecases, ctxUserKey string) *MapsHandler {
	return &MapsHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
	}
}

// sendMapsError sends a structured error response for maps API
func sendMapsError(w http.ResponseWriter, code int, errCode, message string, details []models.ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := models.MapsErrorResponse{
		Error:   errCode,
		Message: message,
		Details: details,
	}

	json.NewEncoder(w).Encode(response)
}

// sendMapsOkResponse sends a successful response
func sendMapsOkResponse(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if body != nil {
		json.NewEncoder(w).Encode(body)
	}
}

func (h *MapsHandler) ListMaps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	// Parse query params
	start := DefaultStart
	size := DefaultSize

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if val, err := strconv.Atoi(startStr); err == nil && val >= 0 {
			start = val
		}
	}

	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
		if val, err := strconv.Atoi(sizeStr); err == nil && val > 0 {
			size = val
		}
	}

	// Clamp size to max
	if size > MaxSize {
		size = MaxSize
	}

	list, err := h.usecases.ListMaps(ctx, userID, start, size)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.StartPosSizeError):
			l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", err, nil)
			sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid start or size parameters", nil)
		default:
			l.DeliveryError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err, nil)
			sendMapsError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		}
		return
	}

	sendMapsOkResponse(w, http.StatusOK, list)
}

func (h *MapsHandler) GetMapByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", nil, nil)
		sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Map ID is required", nil)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", err, nil)
		sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid map ID format", nil)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	mapFull, err := h.usecases.GetMapByID(ctx, userID, id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.MapPermissionDenied):
			l.DeliveryError(ctx, http.StatusForbidden, "FORBIDDEN", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusForbidden, "FORBIDDEN", "Access denied to this map", nil)
		case errors.Is(err, apperrors.MapNotFoundError):
			l.DeliveryError(ctx, http.StatusNotFound, "NOT_FOUND", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusNotFound, "NOT_FOUND", "Map not found", nil)
		default:
			l.DeliveryError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		}
		return
	}

	sendMapsOkResponse(w, http.StatusOK, mapFull)
}

func (h *MapsHandler) CreateMap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBody)

	var reqData models.CreateMapRequest
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", err, nil)
		sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON format", nil)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	mapFull, err := h.usecases.CreateMap(ctx, userID, &reqData)
	if err != nil {
		// Check for validation errors
		var validationErr *usecases.ValidationErrorWrapper
		if errors.As(err, &validationErr) {
			errCode := usecases.CategorizeValidationErrors(validationErr.Errors)
			l.DeliveryError(ctx, http.StatusUnprocessableEntity, errCode, err, nil)
			sendMapsError(w, http.StatusUnprocessableEntity, errCode, "Validation failed", validationErr.Errors)
			return
		}

		l.DeliveryError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err, nil)
		sendMapsError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	sendMapsOkResponse(w, http.StatusCreated, mapFull)
}

func (h *MapsHandler) UpdateMap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", nil, nil)
		sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Map ID is required", nil)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", err, nil)
		sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid map ID format", nil)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBody)

	var reqData models.UpdateMapRequest
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", err, nil)
		sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON format", nil)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	mapFull, err := h.usecases.UpdateMap(ctx, userID, id, &reqData)
	if err != nil {
		// Check for validation errors
		var validationErr *usecases.ValidationErrorWrapper
		if errors.As(err, &validationErr) {
			errCode := usecases.CategorizeValidationErrors(validationErr.Errors)
			l.DeliveryError(ctx, http.StatusUnprocessableEntity, errCode, err, map[string]any{"id": id})
			sendMapsError(w, http.StatusUnprocessableEntity, errCode, "Validation failed", validationErr.Errors)
			return
		}

		switch {
		case errors.Is(err, apperrors.MapPermissionDenied):
			l.DeliveryError(ctx, http.StatusForbidden, "FORBIDDEN", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusForbidden, "FORBIDDEN", "Access denied to this map", nil)
		case errors.Is(err, apperrors.MapNotFoundError):
			l.DeliveryError(ctx, http.StatusNotFound, "NOT_FOUND", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusNotFound, "NOT_FOUND", "Map not found", nil)
		default:
			l.DeliveryError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		}
		return
	}

	sendMapsOkResponse(w, http.StatusOK, mapFull)
}

func (h *MapsHandler) DeleteMap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", nil, nil)
		sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Map ID is required", nil)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		l.DeliveryError(ctx, http.StatusBadRequest, "BAD_REQUEST", err, nil)
		sendMapsError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid map ID format", nil)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	err := h.usecases.DeleteMap(ctx, userID, id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.MapPermissionDenied):
			l.DeliveryError(ctx, http.StatusForbidden, "FORBIDDEN", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusForbidden, "FORBIDDEN", "Access denied to this map", nil)
		case errors.Is(err, apperrors.MapNotFoundError):
			l.DeliveryError(ctx, http.StatusNotFound, "NOT_FOUND", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusNotFound, "NOT_FOUND", "Map not found", nil)
		default:
			l.DeliveryError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", err, map[string]any{"id": id})
			sendMapsError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
