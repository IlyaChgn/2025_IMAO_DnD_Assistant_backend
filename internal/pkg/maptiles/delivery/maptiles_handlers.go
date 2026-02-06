package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	maptilesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
)

type MapTilesHandler struct {
	usecases   maptilesinterfaces.MapTilesUsecases
	ctxUserKey string
}

func NewMapTilesHandler(usecases maptilesinterfaces.MapTilesUsecases, ctxUserKey string) *MapTilesHandler {
	return &MapTilesHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
	}
}

// GET /api/map-tiles/categories
func (h *MapTilesHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	user := ctx.Value(h.ctxUserKey).(*models.User)
	userID := user.ID

	list, err := h.usecases.GetCategories(ctx, userID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			l.DeliveryInfo(ctx, "empty data", nil)
			responses.SendOkResponse(w, nil)
			return
		case errors.Is(err, apperrors.InvalidUserIDError):
			code = responses.StatusBadRequest
			status = responses.ErrInvalidID
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, nil)
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, list)
}

// GET /api/map-tiles/walkability/{tileId}
func (h *MapTilesHandler) GetWalkabilityByTileID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	tileID := vars["tileId"]

	walkability, err := h.usecases.GetWalkabilityByTileID(ctx, tileID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			code = responses.StatusNotFound
			status = responses.ErrNotFound
		case errors.Is(err, apperrors.InvalidTileIDError):
			code = responses.StatusBadRequest
			status = responses.ErrInvalidID
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"tileId": tileID})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, walkability)
}

// GET /api/map-tiles/walkability?setId=...
func (h *MapTilesHandler) GetWalkabilityBySetID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	setID := r.URL.Query().Get("setId")

	walkabilities, err := h.usecases.GetWalkabilityBySetID(ctx, setID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			code = responses.StatusNotFound
			status = responses.ErrNotFound
		case errors.Is(err, apperrors.InvalidSetIDError):
			code = responses.StatusBadRequest
			status = responses.ErrInvalidID
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"setId": setID})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, walkabilities)
}

// PUT /api/map-tiles/walkability/{tileId}
func (h *MapTilesHandler) UpsertWalkability(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	tileID := vars["tileId"]

	var req models.TileWalkability
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	// tileId из URL имеет приоритет
	req.TileID = tileID

	if err := h.usecases.UpsertWalkability(ctx, &req); err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidTileIDError):
			code = responses.StatusBadRequest
			status = responses.ErrInvalidID
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"tileId": tileID})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, nil)
}
