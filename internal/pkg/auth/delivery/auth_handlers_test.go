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
	"github.com/gorilla/mux"
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

func (f *fakeAuthUsecases) Login(_ context.Context, _ string, _ string, _ *models.LoginRequest,
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

// --- helpers ---

const testSessionDuration = 30 * 24 * time.Hour

func newHandler(fake *fakeAuthUsecases, isProd bool) *delivery.AuthHandler {
	return delivery.NewAuthHandler(fake, testSessionDuration, isProd)
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
			fake:       &fakeAuthUsecases{loginResult: &models.User{ID: 1, DisplayName: "Tester"}},
			wantStatus: responses.StatusOk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newHandler(tt.fake, false)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login/vk", nil)
			req.Body = io.NopCloser(bytes.NewReader(tt.body))
			req = mux.SetURLVars(req, map[string]string{"provider": "vk"})

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

			handler := newHandler(tt.fake, false)

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
				checkAuthUser:   &models.User{ID: 1, DisplayName: "Tester"},
				checkAuthIsAuth: true,
			},
			wantStatus: responses.StatusOk,
			wantAuth:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newHandler(tt.fake, false)

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

// --- cookie flag tests ---

func findCookie(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}

	return nil
}

func TestLoginCookieFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		isProd     bool
		wantSecure bool
	}{
		{
			name:       "dev_mode",
			isProd:     false,
			wantSecure: false,
		},
		{
			name:       "prod_mode",
			isProd:     true,
			wantSecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeAuthUsecases{loginResult: &models.User{ID: 1, DisplayName: "Tester"}}
			handler := newHandler(fake, tt.isProd)

			body := testhelpers.MustJSON(t, models.LoginRequest{Code: "code"})
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
			req.Body = io.NopCloser(bytes.NewReader(body))

			rr := httptest.NewRecorder()
			handler.Login(rr, req)

			assert.Equal(t, responses.StatusOk, rr.Code)

			cookie := findCookie(rr, "session_id")
			assert.NotNil(t, cookie)
			assert.True(t, cookie.HttpOnly)
			assert.Equal(t, tt.wantSecure, cookie.Secure)
			assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		})
	}
}

func TestLogoutCookieFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		isProd     bool
		wantSecure bool
	}{
		{
			name:       "dev_mode",
			isProd:     false,
			wantSecure: false,
		},
		{
			name:       "prod_mode",
			isProd:     true,
			wantSecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeAuthUsecases{}
			handler := newHandler(fake, tt.isProd)

			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})

			rr := httptest.NewRecorder()
			handler.Logout(rr, req)

			assert.Equal(t, responses.StatusOk, rr.Code)

			cookie := findCookie(rr, "session_id")
			assert.NotNil(t, cookie)
			assert.True(t, cookie.HttpOnly)
			assert.Equal(t, tt.wantSecure, cookie.Secure)
			assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
			assert.True(t, cookie.Expires.Before(time.Now()), "logout cookie must be expired")
		})
	}
}
