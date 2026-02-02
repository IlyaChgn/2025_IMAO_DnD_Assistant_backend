package delivery_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeCharacterUsecases struct {
	listResult      []*models.CharacterShort
	listErr         error
	characterResult *models.Character
	characterErr    error
	addErr          error
}

func (f *fakeCharacterUsecases) GetCharactersList(_ context.Context, _, _, _ int,
	_ models.SearchParams) ([]*models.CharacterShort, error) {
	return f.listResult, f.listErr
}

func (f *fakeCharacterUsecases) GetCharacterByMongoId(_ context.Context, _ string, _ int) (*models.Character, error) {
	return f.characterResult, f.characterErr
}

func (f *fakeCharacterUsecases) AddCharacter(_ context.Context, _ multipart.File, _ int) error {
	return f.addErr
}

const ctxUserKey = "test-user-key"

func withUser(r *http.Request, key string, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), key, user)
	return r.WithContext(ctx)
}

func TestGetCharactersList_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewCharacterHandler(
		&fakeCharacterUsecases{},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/character", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid`)))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, DisplayName: "Tester"})

	rr := httptest.NewRecorder()
	handler.GetCharactersList(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestGetCharactersList_StartPosSizeError_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewCharacterHandler(
		&fakeCharacterUsecases{listErr: apperrors.StartPosSizeError},
		ctxUserKey,
	)

	body := testhelpers.MustJSON(t, models.CharacterReq{
		Start: -1,
		Size:  10,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/character", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, DisplayName: "Tester"})

	rr := httptest.NewRecorder()
	handler.GetCharactersList(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrSizeOrPosition, testhelpers.DecodeErrorResponse(t, rr.Body))
}
