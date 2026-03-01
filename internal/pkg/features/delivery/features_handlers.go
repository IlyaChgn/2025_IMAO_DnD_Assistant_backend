package delivery

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	featuresinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
)

type FeaturesHandler struct {
	usecases featuresinterfaces.FeaturesUsecases
}

func NewFeaturesHandler(usecases featuresinterfaces.FeaturesUsecases) *FeaturesHandler {
	return &FeaturesHandler{
		usecases: usecases,
	}
}

// GetFeatures handles GET /api/reference/features
func (h *FeaturesHandler) GetFeatures(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	query := r.URL.Query()

	filter := models.FeatureFilterParams{
		Source: query.Get("source"),
		Class:  query.Get("class"),
		Search: query.Get("search"),
	}

	if levelStr := query.Get("level"); levelStr != "" {
		level, err := strconv.Atoi(levelStr)
		if err != nil {
			l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadRequest, err, nil)
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadRequest)
			return
		}
		filter.Level = &level
	}

	if pageStr := query.Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil {
			filter.Page = page
		}
	}

	if sizeStr := query.Get("size"); sizeStr != "" {
		size, err := strconv.Atoi(sizeStr)
		if err == nil {
			filter.Size = size
		}
	}

	result, err := h.usecases.GetFeatures(ctx, filter)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidFeatureSourceErr):
			code = responses.StatusBadRequest
			status = responses.ErrBadRequest
		case errors.Is(err, apperrors.InvalidFeatureLevelErr):
			code = responses.StatusBadRequest
			status = responses.ErrBadRequest
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, nil)
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, result)
}

// GetFeatureByID handles GET /api/reference/features/{id}
func (h *FeaturesHandler) GetFeatureByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	id := vars["id"]

	feature, err := h.usecases.GetFeatureByID(ctx, id)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.FeatureNotFoundErr):
			code = responses.StatusNotFound
			status = responses.ErrNotFound
		case errors.Is(err, apperrors.InvalidIDErr):
			code = responses.StatusBadRequest
			status = responses.ErrInvalidID
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"id": id})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, feature)
}

// GetFeaturesByClass handles GET /api/reference/features/by-class/{className}
func (h *FeaturesHandler) GetFeaturesByClass(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	className := vars["className"]

	var level *int
	if levelStr := r.URL.Query().Get("level"); levelStr != "" {
		lvl, err := strconv.Atoi(levelStr)
		if err != nil {
			l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadRequest, err, nil)
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadRequest)
			return
		}
		level = &lvl
	}

	features, err := h.usecases.GetFeaturesByClass(ctx, className, level)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidFeatureLevelErr):
			code = responses.StatusBadRequest
			status = responses.ErrBadRequest
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"className": className})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, features)
}
