package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dungeongen"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dungeongen/usecases"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

// DungeonGenHandler handles HTTP requests for dungeon generation.
type DungeonGenHandler struct {
	usecases dungeongen.DungeonGenUsecases
}

// NewDungeonGenHandler creates a new DungeonGenHandler.
func NewDungeonGenHandler(uc dungeongen.DungeonGenUsecases) *DungeonGenHandler {
	return &DungeonGenHandler{usecases: uc}
}

// GenerateDungeon handles POST /api/dungeons/generate.
func (h *DungeonGenHandler) GenerateDungeon(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var req dungeongen.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	resp, err := h.usecases.GenerateDungeon(ctx, &req)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, usecases.ErrInvalidSize),
			errors.Is(err, usecases.ErrInvalidPartyLevel),
			errors.Is(err, usecases.ErrInvalidPartySize),
			errors.Is(err, usecases.ErrInvalidDifficulty),
			errors.Is(err, usecases.ErrInvalidTheme):
			code = responses.StatusBadRequest
			status = responses.ErrBadRequest
		case errors.Is(err, usecases.ErrNoTileMetadata):
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, nil)
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, resp)
}
