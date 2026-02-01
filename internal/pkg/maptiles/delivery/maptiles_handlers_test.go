package delivery_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeMapTilesUsecases struct {
	categories []*models.MapTileCategory
	err        error
}

func (f *fakeMapTilesUsecases) GetCategories(_ context.Context, _ int) ([]*models.MapTileCategory, error) {
	return f.categories, f.err
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
	req = withUser(req, ctxUserKey, &models.User{ID: -1, Name: "Tester"})

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
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.GetCategories(rr, req)

	assert.Equal(t, responses.StatusInternalServerError, rr.Code)
	assert.Equal(t, responses.ErrInternalServer, testhelpers.DecodeErrorResponse(t, rr.Body))
}
