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
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeDescriptionUsecases struct {
	result models.DescriptionGenerationResponse
	err    error
}

func (f *fakeDescriptionUsecases) GenerateDescription(_ context.Context,
	_ models.DescriptionGenerationRequest) (models.DescriptionGenerationResponse, error) {
	return f.result, f.err
}

// --- tests ---

func TestGenerateDescription_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewDescriptionHandler(&fakeDescriptionUsecases{})

	req := httptest.NewRequest(http.MethodPost, "/api/battle/description", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid json`)))

	rr := httptest.NewRecorder()
	handler.GenerateDescription(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestGenerateDescription_UsecaseError_Returns500(t *testing.T) {
	t.Parallel()

	handler := delivery.NewDescriptionHandler(
		&fakeDescriptionUsecases{err: errors.New("grpc failure")},
	)

	body := testhelpers.MustJSON(t, models.DescriptionGenerationRequest{
		FirstCharID:  "char-1",
		SecondCharID: "char-2",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/battle/description", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))

	rr := httptest.NewRecorder()
	handler.GenerateDescription(rr, req)

	assert.Equal(t, responses.StatusInternalServerError, rr.Code)
	assert.Equal(t, responses.ErrInternalServer, testhelpers.DecodeErrorResponse(t, rr.Body))
}
