package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type ItemsHandler struct {
	usecases   itemsinterfaces.ItemUsecases
	ctxUserKey string
}

func NewItemsHandler(usecases itemsinterfaces.ItemUsecases, ctxUserKey string) *ItemsHandler {
	return &ItemsHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
	}
}

// GetItems handles GET /api/items/definitions
func (h *ItemsHandler) GetItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	query := r.URL.Query()

	filter := models.ItemFilterParams{
		Category: models.ItemCategory(query.Get("category")),
		Rarity:   models.ItemRarity(query.Get("rarity")),
		Search:   query.Get("search"),
	}

	if tagsStr := query.Get("tags"); tagsStr != "" {
		filter.Tags = strings.Split(tagsStr, ",")
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

	result, err := h.usecases.GetItems(ctx, filter)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidItemCategoryErr),
			errors.Is(err, apperrors.InvalidItemRarityErr):
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

// GetItemByEngName handles GET /api/items/definitions/{engName}
func (h *ItemsHandler) GetItemByEngName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	engName := vars["engName"]

	item, err := h.usecases.GetItemByEngName(ctx, engName)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.ItemNotFoundErr):
			code = responses.StatusNotFound
			status = responses.ErrNotFound
		case errors.Is(err, apperrors.InvalidIDErr):
			code = responses.StatusBadRequest
			status = responses.ErrInvalidID
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"engName": engName})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, item)
}

// CreateItem handles POST /api/items/definitions
func (h *ItemsHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var item models.ItemDefinition
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	created, err := h.usecases.CreateItem(ctx, &item, user.ID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.InvalidItemCategoryErr),
			errors.Is(err, apperrors.InvalidItemRarityErr):
			code = responses.StatusBadRequest
			status = responses.ErrBadRequest
		case errors.Is(err, apperrors.DuplicateEngNameErr):
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// UpdateItem handles PUT /api/items/definitions/{id}
func (h *ItemsHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	id := vars["id"]

	var item models.ItemDefinition
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	objID, err := parseObjectID(id)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrInvalidID, err, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		return
	}
	item.ID = objID

	user := ctx.Value(h.ctxUserKey).(*models.User)

	updated, err := h.usecases.UpdateItem(ctx, &item, user.ID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.ItemNotFoundErr):
			code = responses.StatusNotFound
			status = responses.ErrNotFound
		case errors.Is(err, apperrors.ItemNotCustomErr),
			errors.Is(err, apperrors.ItemNotOwnedErr):
			code = responses.StatusForbidden
			status = responses.ErrForbidden
		case errors.Is(err, apperrors.InvalidItemCategoryErr),
			errors.Is(err, apperrors.InvalidItemRarityErr),
			errors.Is(err, apperrors.DuplicateEngNameErr),
			errors.Is(err, apperrors.InvalidIDErr):
			code = responses.StatusBadRequest
			status = responses.ErrBadRequest
		default:
			code = responses.StatusInternalServerError
			status = responses.ErrInternalServer
		}

		l.DeliveryError(ctx, code, status, err, map[string]any{"id": id})
		responses.SendErrResponse(w, code, status)
		return
	}

	responses.SendOkResponse(w, updated)
}

// DeleteItem handles DELETE /api/items/definitions/{id}
func (h *ItemsHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	vars := mux.Vars(r)
	id := vars["id"]

	user := ctx.Value(h.ctxUserKey).(*models.User)

	err := h.usecases.DeleteItem(ctx, id, user.ID)
	if err != nil {
		var code int
		var status string

		switch {
		case errors.Is(err, apperrors.ItemNotFoundErr):
			code = responses.StatusNotFound
			status = responses.ErrNotFound
		case errors.Is(err, apperrors.ItemNotCustomErr),
			errors.Is(err, apperrors.ItemNotOwnedErr):
			code = responses.StatusForbidden
			status = responses.ErrForbidden
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

	w.WriteHeader(http.StatusNoContent)
}

func parseObjectID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}
