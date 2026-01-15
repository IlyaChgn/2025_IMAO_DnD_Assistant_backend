package delivery

import (
	"errors"
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	maptilesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
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
