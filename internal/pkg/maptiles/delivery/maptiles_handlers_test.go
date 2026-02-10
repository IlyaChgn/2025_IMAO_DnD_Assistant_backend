package delivery_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeMapTilesUsecases struct {
	categories    []*models.MapTileCategory
	walkability   *models.TileWalkability
	walkabilities []*models.TileWalkability
	err           error
}

func (f *fakeMapTilesUsecases) GetCategories(_ context.Context, _ int) ([]*models.MapTileCategory, error) {
	return f.categories, f.err
}

func (f *fakeMapTilesUsecases) GetWalkabilityByTileID(_ context.Context, _ string) (*models.TileWalkability, error) {
	return f.walkability, f.err
}

func (f *fakeMapTilesUsecases) GetWalkabilityBySetID(_ context.Context, _ string) ([]*models.TileWalkability, error) {
	return f.walkabilities, f.err
}

func (f *fakeMapTilesUsecases) UpsertWalkability(_ context.Context, _ *models.TileWalkability) error {
	return f.err
}

func (f *fakeMapTilesUsecases) AddTile(_ context.Context, _ string, _ *models.MapTile) error {
	return f.err
}

func (f *fakeMapTilesUsecases) UpdateTile(_ context.Context, _ string, _ *models.MapTile) error {
	return f.err
}

func (f *fakeMapTilesUsecases) DeleteTile(_ context.Context, _ string, _ string) error {
	return f.err
}

const ctxUserKey = "test-user-key"

func withUser(r *http.Request, key string, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), key, user)
	return r.WithContext(ctx)
}

func TestGetCategories_InvalidUserID_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(
		&fakeMapTilesUsecases{err: apperrors.InvalidUserIDError},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/map-tiles/categories", nil)
	req = withUser(req, ctxUserKey, &models.User{ID: -1, DisplayName: "Tester"})

	rr := httptest.NewRecorder()
	handler.GetCategories(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrInvalidID, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestGetCategories_InternalError_Returns500(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(
		&fakeMapTilesUsecases{err: errors.New("unexpected db failure")},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/map-tiles/categories", nil)
	req = withUser(req, ctxUserKey, &models.User{ID: 1, DisplayName: "Tester"})

	rr := httptest.NewRecorder()
	handler.GetCategories(rr, req)

	assert.Equal(t, responses.StatusInternalServerError, rr.Code)
	assert.Equal(t, responses.ErrInternalServer, testhelpers.DecodeErrorResponse(t, rr.Body))
}

// --- AddTile tests ---

func TestAddTile_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(&fakeMapTilesUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodPost, "/api/map-tiles", strings.NewReader("{bad json"))
	rr := httptest.NewRecorder()
	handler.AddTile(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestAddTile_InvalidID_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(
		&fakeMapTilesUsecases{err: apperrors.InvalidIDErr},
		ctxUserKey,
	)

	body := `{"categoryId":"","tile":{"id":"t1","name":"Tile","imageUrl":"img.png"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/map-tiles", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.AddTile(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrInvalidID, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestAddTile_Success_Returns200(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(&fakeMapTilesUsecases{}, ctxUserKey)

	body := `{"categoryId":"cat1","tile":{"id":"t1","name":"Tile","imageUrl":"img.png"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/map-tiles", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.AddTile(rr, req)

	assert.Equal(t, responses.StatusOk, rr.Code)
}

// --- UpdateTile tests ---

func TestUpdateTile_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(&fakeMapTilesUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodPut, "/api/map-tiles/t1", strings.NewReader("{bad"))
	req = mux.SetURLVars(req, map[string]string{"tileId": "t1"})
	rr := httptest.NewRecorder()
	handler.UpdateTile(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestUpdateTile_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(
		&fakeMapTilesUsecases{err: apperrors.NoDocsErr},
		ctxUserKey,
	)

	body := `{"categoryId":"cat1","tile":{"id":"t1","name":"Tile","imageUrl":"img.png"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/map-tiles/t1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"tileId": "t1"})
	rr := httptest.NewRecorder()
	handler.UpdateTile(rr, req)

	assert.Equal(t, responses.StatusNotFound, rr.Code)
	assert.Equal(t, responses.ErrNotFound, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestUpdateTile_Success_Returns200(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(&fakeMapTilesUsecases{}, ctxUserKey)

	body := `{"categoryId":"cat1","tile":{"id":"old","name":"Tile","imageUrl":"img.png"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/map-tiles/t1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"tileId": "t1"})
	rr := httptest.NewRecorder()
	handler.UpdateTile(rr, req)

	assert.Equal(t, responses.StatusOk, rr.Code)
}

// --- DeleteTile tests ---

func TestDeleteTile_InvalidID_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(
		&fakeMapTilesUsecases{err: apperrors.InvalidIDErr},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodDelete, "/api/map-tiles/t1?categoryId=", nil)
	req = mux.SetURLVars(req, map[string]string{"tileId": "t1"})
	rr := httptest.NewRecorder()
	handler.DeleteTile(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrInvalidID, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestDeleteTile_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(
		&fakeMapTilesUsecases{err: apperrors.NoDocsErr},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodDelete, "/api/map-tiles/t1?categoryId=cat1", nil)
	req = mux.SetURLVars(req, map[string]string{"tileId": "t1"})
	rr := httptest.NewRecorder()
	handler.DeleteTile(rr, req)

	assert.Equal(t, responses.StatusNotFound, rr.Code)
	assert.Equal(t, responses.ErrNotFound, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestDeleteTile_Success_Returns200(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapTilesHandler(&fakeMapTilesUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodDelete, "/api/map-tiles/t1?categoryId=cat1", nil)
	req = mux.SetURLVars(req, map[string]string{"tileId": "t1"})
	rr := httptest.NewRecorder()
	handler.DeleteTile(rr, req)

	assert.Equal(t, responses.StatusOk, rr.Code)
}
