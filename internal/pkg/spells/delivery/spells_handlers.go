package delivery

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	spellsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells"
	"github.com/gorilla/mux"
)

type SpellsHandler struct {
	usecases spellsinterfaces.SpellsUsecases
}

func NewSpellsHandler(usecases spellsinterfaces.SpellsUsecases) *SpellsHandler {
	return &SpellsHandler{
		usecases: usecases,
	}
}

// GetSpells handles GET /api/reference/spells
func (h *SpellsHandler) GetSpells(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	query := r.URL.Query()

	filter := models.SpellFilterParams{
		Class:  query.Get("class"),
		School: query.Get("school"),
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

	if ritualStr := query.Get("ritual"); ritualStr == "true" {
		ritual := true
		filter.Ritual = &ritual
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

	result, err := h.usecases.GetSpells(ctx, filter)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidSpellLevelErr):
			code = responses.StatusBadRequest
			status = responses.ErrBadRequest
		case errors.Is(err, apperrors.InvalidSpellSchoolErr):
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

// GetSpellByID handles GET /api/reference/spells/{id}
func (h *SpellsHandler) GetSpellByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	id := vars["id"]

	spell, err := h.usecases.GetSpellByID(ctx, id)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.SpellNotFoundErr):
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

	responses.SendOkResponse(w, spell)
}

// GetSpellsByClass handles GET /api/reference/spells/by-class/{className}
func (h *SpellsHandler) GetSpellsByClass(w http.ResponseWriter, r *http.Request) {
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

	spells, err := h.usecases.GetSpellsByClass(ctx, className, level)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidSpellLevelErr):
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

	responses.SendOkResponse(w, spells)
}
