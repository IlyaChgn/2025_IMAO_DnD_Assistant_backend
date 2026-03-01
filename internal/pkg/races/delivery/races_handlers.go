package delivery

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	racesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
)

type RacesHandler struct {
	usecases racesinterfaces.RacesUsecases
}

func NewRacesHandler(usecases racesinterfaces.RacesUsecases) *RacesHandler {
	return &RacesHandler{usecases: usecases}
}

// GetRaces handles GET /api/reference/races
func (h *RacesHandler) GetRaces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	query := r.URL.Query()

	filter := models.RaceFilterParams{
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

	result, err := h.usecases.GetRaces(ctx, filter)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		return
	}

	responses.SendOkResponse(w, result)
}

// GetRaceByEngName handles GET /api/reference/races/{engName}
func (h *RacesHandler) GetRaceByEngName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	engName := vars["engName"]

	race, err := h.usecases.GetRaceByEngName(ctx, engName)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.RaceNotFoundErr):
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

	responses.SendOkResponse(w, race)
}
