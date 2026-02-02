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

func TestLogin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        []byte
		fake        *fakeAuthUsecases
		wantStatus  int
		wantErrCode string
	}{
		{
			name:        "bad_json",
			body:        []byte(`{invalid json`),
			fake:        &fakeAuthUsecases{},
			wantStatus:  responses.StatusBadRequest,
			wantErrCode: responses.ErrBadJSON,
		},
		{
			name:        "vk_api_error",
			body:        testhelpers.MustJSON(t, models.LoginRequest{Code: "code"}),
			fake:        &fakeAuthUsecases{loginErr: apperrors.VKApiError},
			wantStatus:  responses.StatusInternalServerError,
			wantErrCode: responses.ErrVKServer,
		},
		{
			name:        "generic_error",
			body:        testhelpers.MustJSON(t, models.LoginRequest{Code: "code"}),
			fake:        &fakeAuthUsecases{loginErr: errors.New("unexpected")},
			wantStatus:  responses.StatusInternalServerError,
			wantErrCode: responses.ErrInternalServer,
		},
		{
			name:       "happy_path",
			body:       testhelpers.MustJSON(t, models.LoginRequest{Code: "code"}),
			fake:       &fakeAuthUsecases{loginResult: &models.User{ID: 1, Name: "Tester"}},
			wantStatus: responses.StatusOk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := delivery.NewAuthHandler(tt.fake)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
			req.Body = io.NopCloser(bytes.NewReader(tt.body))

			rr := httptest.NewRecorder()
			handler.Login(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErrCode != "" {
				assert.Equal(t, tt.wantErrCode, testhelpers.DecodeErrorResponse(t, rr.Body))
			} else {
				var resp models.AuthResponse
				testhelpers.DecodeJSON(t, rr.Body, &resp)
				assert.True(t, resp.IsAuth)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fake        *fakeAuthUsecases
		wantStatus  int
		wantErrCode string
	}{
		{
			name:        "error",
			fake:        &fakeAuthUsecases{logoutErr: errors.New("redis error")},
			wantStatus:  responses.StatusInternalServerError,
			wantErrCode: responses.ErrInternalServer,
		},
		{
			name:       "happy_path",
			fake:       &fakeAuthUsecases{},
			wantStatus: responses.StatusOk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := delivery.NewAuthHandler(tt.fake)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})

			rr := httptest.NewRecorder()
			handler.Logout(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantErrCode != "" {
				assert.Equal(t, tt.wantErrCode, testhelpers.DecodeErrorResponse(t, rr.Body))
			} else {
				var resp models.AuthResponse
				testhelpers.DecodeJSON(t, rr.Body, &resp)
				assert.False(t, resp.IsAuth)
			}
		})
	}
}

func TestCheckAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cookie     *http.Cookie
		fake       *fakeAuthUsecases
		wantStatus int
		wantAuth   bool
	}{
		{
			name:       "no_cookie",
			cookie:     nil,
			fake:       &fakeAuthUsecases{},
			wantStatus: responses.StatusOk,
			wantAuth:   false,
		},
		{
			name:       "not_authenticated",
			cookie:     &http.Cookie{Name: "session_id", Value: "test-session"},
			fake:       &fakeAuthUsecases{checkAuthIsAuth: false},
			wantStatus: responses.StatusOk,
			wantAuth:   false,
		},
		{
			name:   "authenticated",
			cookie: &http.Cookie{Name: "session_id", Value: "test-session"},
			fake: &fakeAuthUsecases{
				checkAuthUser:   &models.User{ID: 1, Name: "Tester"},
				checkAuthIsAuth: true,
			},
			wantStatus: responses.StatusOk,
			wantAuth:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := delivery.NewAuthHandler(tt.fake)

			req := httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			rr := httptest.NewRecorder()
			handler.CheckAuth(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)

			var resp models.AuthResponse
			testhelpers.DecodeJSON(t, rr.Body, &resp)
			assert.Equal(t, tt.wantAuth, resp.IsAuth)
		})
	}
}
