package delivery

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	featsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
)

type FeatsHandler struct {
	usecases featsinterfaces.FeatsUsecases
}

func NewFeatsHandler(usecases featsinterfaces.FeatsUsecases) *FeatsHandler {
	return &FeatsHandler{usecases: usecases}
}

// GetFeats handles GET /api/reference/feats
func (h *FeatsHandler) GetFeats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	query := r.URL.Query()

	filter := models.FeatFilterParams{
		Search: query.Get("search"),
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

	result, err := h.usecases.GetFeats(ctx, filter)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		return
	}

	responses.SendOkResponse(w, result)
}

// GetFeatByEngName handles GET /api/reference/feats/{engName}
func (h *FeatsHandler) GetFeatByEngName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	engName := vars["engName"]

	feat, err := h.usecases.GetFeatByEngName(ctx, engName)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.FeatNotFoundErr):
			code = responses.StatusNotFound
			status = responses.ErrNotFound
		case errors.Is(err, apperrors.InvalidIDErr):
			code = responses.StatusBadRequest
			status = responses.ErrBadRequest
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"engName": engName})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, feat)
}
