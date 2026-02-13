package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type InventoryHandler struct {
	usecases   itemsinterfaces.InventoryUsecases
	ctxUserKey string
}

func NewInventoryHandler(usecases itemsinterfaces.InventoryUsecases, ctxUserKey string) *InventoryHandler {
	return &InventoryHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
	}
}

// GetContainers handles GET /api/inventory/containers
func (h *InventoryHandler) GetContainers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	query := r.URL.Query()
	filter := models.ContainerFilterParams{
		EncounterID: query.Get("encounterId"),
		OwnerID:     query.Get("ownerId"),
		Kind:        models.ContainerKind(query.Get("kind")),
	}

	result, err := h.usecases.GetContainers(ctx, filter)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		return
	}

	responses.SendOkResponse(w, result)
}

// GetContainer handles GET /api/inventory/containers/{id}
func (h *InventoryHandler) GetContainer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	id := vars["id"]

	container, err := h.usecases.GetContainer(ctx, id)
	if err != nil {
		code, status := mapInventoryError(err)
		l.DeliveryError(ctx, code, status, err, map[string]any{"id": id})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, container)
}

// CreateContainer handles POST /api/inventory/containers
func (h *InventoryHandler) CreateContainer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var container models.InventoryContainer
	if err := json.NewDecoder(r.Body).Decode(&container); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	created, err := h.usecases.CreateContainer(ctx, &container)
	if err != nil {
		code, status := mapInventoryError(err)
		l.DeliveryError(ctx, code, status, err, nil)
		responses.SendErrResponse(w, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// DeleteContainer handles DELETE /api/inventory/containers/{id}
func (h *InventoryHandler) DeleteContainer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	id := vars["id"]

	user := ctx.Value(h.ctxUserKey).(*models.User)

	err := h.usecases.DeleteContainer(ctx, id, user.ID)
	if err != nil {
		code, status := mapInventoryError(err)
		l.DeliveryError(ctx, code, status, err, map[string]any{"id": id})
		responses.SendErrResponse(w, code, status)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExecuteCommand handles POST /api/inventory/commands
func (h *InventoryHandler) ExecuteCommand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var req models.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	result, err := h.usecases.ExecuteCommand(ctx, &req, user.ID)
	if err != nil {
		code, status := mapInventoryError(err)
		l.DeliveryError(ctx, code, status, err, map[string]any{"containerId": req.ContainerID, "command": req.Command.Type})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, result)
}

// GenerateLoot handles POST /api/inventory/generate-loot
func (h *InventoryHandler) GenerateLoot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var req models.GenerateLootRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	result, err := h.usecases.GenerateLoot(ctx, &req, user.ID)
	if err != nil {
		code, status := mapInventoryError(err)
		l.DeliveryError(ctx, code, status, err, map[string]any{"cr": req.CR})
		responses.SendErrResponse(w, code, status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func mapInventoryError(err error) (int, string) {
	switch {
	case errors.Is(err, apperrors.ContainerNotFoundErr),
		errors.Is(err, apperrors.ItemNotFoundErr):
		return responses.StatusNotFound, responses.ErrNotFound
	case errors.Is(err, apperrors.VersionConflictErr):
		return responses.StatusConflict, responses.ErrVersionConflict
	case errors.Is(err, apperrors.InvalidCommandErr),
		errors.Is(err, apperrors.InvalidCommandTypeErr),
		errors.Is(err, apperrors.InvalidContainerKindErr),
		errors.Is(err, apperrors.InvalidLayoutTypeErr),
		errors.Is(err, apperrors.EmptyContainerNameErr),
		errors.Is(err, apperrors.ItemNotStackableErr),
		errors.Is(err, apperrors.InsufficientQuantityErr),
		errors.Is(err, apperrors.ItemNotEquippedErr),
		errors.Is(err, apperrors.ItemNotConsumableErr),
		errors.Is(err, apperrors.NegativeCoinsErr),
		errors.Is(err, apperrors.ItemNotInContainerErr),
		errors.Is(err, apperrors.InvalidIDErr),
		errors.Is(err, apperrors.MissingEncounterIDErr),
		errors.Is(err, apperrors.InvalidCRErr):
		return responses.StatusBadRequest, responses.ErrInvalidCommand
	case errors.Is(err, apperrors.ContainerFullErr),
		errors.Is(err, apperrors.SlotOccupiedErr):
		return responses.StatusBadRequest, responses.ErrBadRequest
	default:
		return responses.StatusInternalServerError, responses.ErrInternalServer
	}
}
