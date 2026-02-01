package delivery_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeBestiaryUsecases struct {
	listResult  []*models.BestiaryCreature
	listErr     error
	creatureErr error
	addErr      error
	generateErr error
}

func (f *fakeBestiaryUsecases) GetCreaturesList(_ context.Context, _, _ int, _ []models.Order,
	_ models.FilterParams, _ models.SearchParams) ([]*models.BestiaryCreature, error) {
	return f.listResult, f.listErr
}

func (f *fakeBestiaryUsecases) GetCreatureByEngName(_ context.Context, _ string) (*models.Creature, error) {
	return nil, f.creatureErr
}

func (f *fakeBestiaryUsecases) GetUserCreaturesList(_ context.Context, _, _ int, _ []models.Order,
	_ models.FilterParams, _ models.SearchParams, _ int) ([]*models.BestiaryCreature, error) {
	return nil, nil
}

func (f *fakeBestiaryUsecases) GetUserCreatureByEngName(_ context.Context, _ string, _ int) (*models.Creature, error) {
	return nil, nil
}

func (f *fakeBestiaryUsecases) AddGeneratedCreature(_ context.Context, _ models.CreatureInput, _ int) error {
	return f.addErr
}

func (f *fakeBestiaryUsecases) ParseCreatureFromImage(_ context.Context, _ []byte) (*models.Creature, error) {
	return nil, nil
}

func (f *fakeBestiaryUsecases) GenerateCreatureFromDescription(_ context.Context, _ string) (*models.Creature, error) {
	return nil, f.generateErr
}

// --- helpers ---

const ctxUserKey = "test-user-key"

func withUser(r *http.Request, key string, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), key, user)
	return r.WithContext(ctx)
}

// --- tests ---

func TestGetCreaturesList_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewBestiaryHandler(&fakeBestiaryUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodPost, "/api/bestiary", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid json`)))

	rr := httptest.NewRecorder()
	handler.GetCreaturesList(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestGetCreaturesList_StartPosSizeError_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewBestiaryHandler(
		&fakeBestiaryUsecases{listErr: apperrors.StartPosSizeError},
		ctxUserKey,
	)

	body := testhelpers.MustJSON(t, models.BestiaryReq{Start: -1, Size: 10})
	req := httptest.NewRequest(http.MethodPost, "/api/bestiary", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))

	rr := httptest.NewRecorder()
	handler.GetCreaturesList(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrSizeOrPosition, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestGetCreaturesList_GenericError_Returns500(t *testing.T) {
	t.Parallel()

	handler := delivery.NewBestiaryHandler(
		&fakeBestiaryUsecases{listErr: errors.New("unexpected")},
		ctxUserKey,
	)

	body := testhelpers.MustJSON(t, models.BestiaryReq{Start: 0, Size: 10})
	req := httptest.NewRequest(http.MethodPost, "/api/bestiary", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))

	rr := httptest.NewRecorder()
	handler.GetCreaturesList(rr, req)

	assert.Equal(t, responses.StatusInternalServerError, rr.Code)
	assert.Equal(t, responses.ErrInternalServer, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestAddGeneratedCreature_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewBestiaryHandler(&fakeBestiaryUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodPost, "/api/bestiary/generated", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid json`)))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.AddGeneratedCreature(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestSubmitCreatureGenerationPrompt_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewBestiaryHandler(&fakeBestiaryUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodPost, "/api/bestiary/generate", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid json`)))

	rr := httptest.NewRecorder()
	handler.SubmitCreatureGenerationPrompt(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}
