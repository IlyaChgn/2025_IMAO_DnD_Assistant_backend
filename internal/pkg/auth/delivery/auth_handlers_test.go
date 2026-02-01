package delivery_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeAuthUsecases struct {
	loginResult     *models.User
	loginErr        error
	logoutErr       error
	checkAuthUser   *models.User
	checkAuthIsAuth bool
}

func (f *fakeAuthUsecases) Login(_ context.Context, _ string, _ *models.LoginRequest,
	_ time.Duration) (*models.User, error) {
	return f.loginResult, f.loginErr
}

func (f *fakeAuthUsecases) Logout(_ context.Context, _ string) error {
	return f.logoutErr
}

func (f *fakeAuthUsecases) CheckAuth(_ context.Context, _ string) (*models.User, bool) {
	return f.checkAuthUser, f.checkAuthIsAuth
}

func (f *fakeAuthUsecases) GetUserIDBySessionID(_ context.Context, _ string) int {
	return 0
}

// --- tests ---

func TestLogin_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewAuthHandler(&fakeAuthUsecases{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid json`)))

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestLogin_VKApiError_Returns500(t *testing.T) {
	t.Parallel()

	handler := delivery.NewAuthHandler(
		&fakeAuthUsecases{loginErr: apperrors.VKApiError},
	)

	body := testhelpers.MustJSON(t, models.LoginRequest{Code: "code"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	assert.Equal(t, responses.StatusInternalServerError, rr.Code)
	assert.Equal(t, responses.ErrVKServer, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestLogin_GenericError_Returns500(t *testing.T) {
	t.Parallel()

	handler := delivery.NewAuthHandler(
		&fakeAuthUsecases{loginErr: errors.New("unexpected")},
	)

	body := testhelpers.MustJSON(t, models.LoginRequest{Code: "code"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	assert.Equal(t, responses.StatusInternalServerError, rr.Code)
	assert.Equal(t, responses.ErrInternalServer, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestCheckAuth_NoCookie_ReturnsNotAuth(t *testing.T) {
	t.Parallel()

	handler := delivery.NewAuthHandler(&fakeAuthUsecases{})

	req := httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	// no session_id cookie

	rr := httptest.NewRecorder()
	handler.CheckAuth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
