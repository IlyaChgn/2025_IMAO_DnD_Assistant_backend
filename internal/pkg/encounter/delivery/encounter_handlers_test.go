package delivery_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeEncounterUsecases struct {
	saveErr error
}

func (f *fakeEncounterUsecases) GetEncountersList(_ context.Context, _, _, _ int,
	_ *models.SearchParams) (*models.EncountersList, error) {
	return nil, nil
}

func (f *fakeEncounterUsecases) GetEncounterByID(_ context.Context, _ string, _ int) (*models.Encounter, error) {
	return nil, nil
}

func (f *fakeEncounterUsecases) SaveEncounter(_ context.Context, _ *models.SaveEncounterReq, _ int) error {
	return f.saveErr
}

func (f *fakeEncounterUsecases) UpdateEncounter(_ context.Context, _ []byte, _ string, _ int) error {
	return nil
}

func (f *fakeEncounterUsecases) RemoveEncounter(_ context.Context, _ string, _ int) error {
	return nil
}

// ctxUserKey must match the key used by the handler to extract the user from context.
const ctxUserKey = "test-user-key"

// withUser injects a models.User into request context with the given key.
func withUser(r *http.Request, key string, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), key, user)
	return r.WithContext(ctx)
}

func TestSaveEncounter_InvalidInput_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewEncounterHandler(
		&fakeEncounterUsecases{saveErr: apperrors.InvalidInputError},
		ctxUserKey,
	)

	body := testhelpers.MustJSON(t, models.SaveEncounterReq{
		Name: "", // triggers InvalidInputError in usecase
	})

	req := httptest.NewRequest(http.MethodPost, "/api/encounter", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.SaveEncounter(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrWrongEncounterName, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestSaveEncounter_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewEncounterHandler(
		&fakeEncounterUsecases{},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/encounter", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid json`)))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.SaveEncounter(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}
