package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/mocks"
	middleware "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const testCtxUserKey = "user"

func TestLoginRequiredMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cookie      *http.Cookie
		setup       func(uc *mocks.MockAuthUsecases)
		wantCode    int
		wantErrCode string
		wantNext    bool
	}{
		{
			name:        "no_cookie",
			cookie:      nil,
			setup:       func(_ *mocks.MockAuthUsecases) {},
			wantCode:    responses.StatusUnauthorized,
			wantErrCode: responses.ErrNotAuthorized,
		},
		{
			name:   "not_authenticated",
			cookie: &http.Cookie{Name: "session_id", Value: "bad-session"},
			setup: func(uc *mocks.MockAuthUsecases) {
				uc.EXPECT().CheckAuth(gomock.Any(), "bad-session").Return(nil, false)
			},
			wantCode:    responses.StatusUnauthorized,
			wantErrCode: responses.ErrNotAuthorized,
		},
		{
			name:   "active_user",
			cookie: &http.Cookie{Name: "session_id", Value: "good-session"},
			setup: func(uc *mocks.MockAuthUsecases) {
				uc.EXPECT().CheckAuth(gomock.Any(), "good-session").
					Return(&models.User{ID: 1, Name: "Tester", Status: "active"}, true)
			},
			wantCode: http.StatusOK,
			wantNext: true,
		},
		{
			name:   "empty_status_old_session",
			cookie: &http.Cookie{Name: "session_id", Value: "old-session"},
			setup: func(uc *mocks.MockAuthUsecases) {
				uc.EXPECT().CheckAuth(gomock.Any(), "old-session").
					Return(&models.User{ID: 1, Name: "Tester", Status: ""}, true)
			},
			wantCode: http.StatusOK,
			wantNext: true,
		},
		{
			name:   "banned_user",
			cookie: &http.Cookie{Name: "session_id", Value: "banned-session"},
			setup: func(uc *mocks.MockAuthUsecases) {
				uc.EXPECT().CheckAuth(gomock.Any(), "banned-session").
					Return(&models.User{ID: 2, Name: "Banned", Status: "banned"}, true)
			},
			wantCode:    responses.StatusForbidden,
			wantErrCode: responses.ErrUserInactive,
		},
		{
			name:   "deleted_user",
			cookie: &http.Cookie{Name: "session_id", Value: "deleted-session"},
			setup: func(uc *mocks.MockAuthUsecases) {
				uc.EXPECT().CheckAuth(gomock.Any(), "deleted-session").
					Return(&models.User{ID: 3, Name: "Deleted", Status: "deleted"}, true)
			},
			wantCode:    responses.StatusForbidden,
			wantErrCode: responses.ErrUserInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mocks.NewMockAuthUsecases(ctrl)
			tt.setup(uc)

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			mw := middleware.LoginRequiredMiddleware(uc, testCtxUserKey)
			handler := mw(next)

			req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantCode, rr.Code)
			assert.Equal(t, tt.wantNext, nextCalled)

			if tt.wantErrCode != "" {
				assert.Equal(t, tt.wantErrCode, testhelpers.DecodeErrorResponse(t, rr.Body))
			}
		})
	}
}
