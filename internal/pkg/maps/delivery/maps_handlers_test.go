package delivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps/usecases"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeMapsUsecases struct {
	createResult *models.MapFull
	createErr    error
	getResult    *models.MapFull
	getErr       error
	updateResult *models.MapFull
	updateErr    error
	deleteErr    error
	listResult   *models.MapsList
	listErr      error
}

func (f *fakeMapsUsecases) CreateMap(_ context.Context, _ int, _ *models.CreateMapRequest) (*models.MapFull, error) {
	return f.createResult, f.createErr
}

func (f *fakeMapsUsecases) GetMapByID(_ context.Context, _ int, _ string) (*models.MapFull, error) {
	return f.getResult, f.getErr
}

func (f *fakeMapsUsecases) UpdateMap(_ context.Context, _ int, _ string, _ *models.UpdateMapRequest) (*models.MapFull, error) {
	return f.updateResult, f.updateErr
}

func (f *fakeMapsUsecases) DeleteMap(_ context.Context, _ int, _ string) error {
	return f.deleteErr
}

func (f *fakeMapsUsecases) ListMaps(_ context.Context, _ int, _, _ int) (*models.MapsList, error) {
	return f.listResult, f.listErr
}

// --- helpers ---

const ctxUserKey = "test-user-key"

func withUser(r *http.Request, key string, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), key, user)
	return r.WithContext(ctx)
}

// decodeMapsError decodes a MapsErrorResponse from the response body and returns the Error code.
func decodeMapsError(t *testing.T, body *bytes.Buffer) string {
	t.Helper()

	var resp models.MapsErrorResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		t.Fatalf("decodeMapsError: failed to decode: %v", err)
	}

	return resp.Error
}

// --- tests ---

func TestCreateMap_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapsHandler(&fakeMapsUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodPost, "/api/maps", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid json`)))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.CreateMap(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "BAD_REQUEST", decodeMapsError(t, rr.Body))
}

func TestCreateMap_ValidationError_Returns422(t *testing.T) {
	t.Parallel()

	valErr := &usecases.ValidationErrorWrapper{
		Errors: []models.ValidationError{
			{Field: "name", Message: "name must be between 1 and 255 characters"},
		},
	}

	handler := delivery.NewMapsHandler(
		&fakeMapsUsecases{createErr: valErr},
		ctxUserKey,
	)

	body, _ := json.Marshal(models.CreateMapRequest{
		Name: "",
		Data: models.MapData{SchemaVersion: 1, WidthUnits: 12, HeightUnits: 12},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/maps", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.CreateMap(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	assert.Equal(t, "BAD_REQUEST", decodeMapsError(t, rr.Body))
}

func TestCreateMap_InternalError_Returns500(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapsHandler(
		&fakeMapsUsecases{createErr: errors.New("unexpected db failure")},
		ctxUserKey,
	)

	body, _ := json.Marshal(models.CreateMapRequest{
		Name: "Test",
		Data: models.MapData{SchemaVersion: 1, WidthUnits: 12, HeightUnits: 12},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/maps", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.CreateMap(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "INTERNAL_ERROR", decodeMapsError(t, rr.Body))
}

func TestListMaps_InternalError_Returns500(t *testing.T) {
	t.Parallel()

	handler := delivery.NewMapsHandler(
		&fakeMapsUsecases{listErr: apperrors.StartPosSizeError},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/maps?start=-1&size=10", nil)
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.ListMaps(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "BAD_REQUEST", decodeMapsError(t, rr.Body))
}
