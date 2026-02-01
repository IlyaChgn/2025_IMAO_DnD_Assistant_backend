package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/stretchr/testify/assert"
)

// --- fakes ---

type fakeAuthRepo struct {
	checkUserResult *models.User
	checkUserErr    error
	createResult    *models.User
	createErr       error
	updateResult    *models.User
	updateErr       error
}

func (f *fakeAuthRepo) CheckUser(_ context.Context, _ string) (*models.User, error) {
	return f.checkUserResult, f.checkUserErr
}

func (f *fakeAuthRepo) CreateUser(_ context.Context, _ *models.User) (*models.User, error) {
	return f.createResult, f.createErr
}

func (f *fakeAuthRepo) UpdateUser(_ context.Context, _ *models.User) (*models.User, error) {
	return f.updateResult, f.updateErr
}

type fakeVKApi struct {
	exchangeResult []byte
	exchangeErr    error
	publicResult   []byte
	publicErr      error
}

func (f *fakeVKApi) ExchangeCode(_ context.Context, _ *models.LoginRequest) ([]byte, error) {
	return f.exchangeResult, f.exchangeErr
}

func (f *fakeVKApi) GetPublicInfo(_ context.Context, _ string) ([]byte, error) {
	return f.publicResult, f.publicErr
}

type fakeSessionManager struct {
	createErr    error
	removeErr    error
	getResult    *models.FullSessionData
	getIsAuth    bool
	createCalled bool
	removeCalled bool
}

func (f *fakeSessionManager) CreateSession(_ context.Context, _ string, _ *models.FullSessionData,
	_ time.Duration) error {
	f.createCalled = true
	return f.createErr
}

func (f *fakeSessionManager) RemoveSession(_ context.Context, _ string) error {
	f.removeCalled = true
	return f.removeErr
}

func (f *fakeSessionManager) GetSession(_ context.Context, _ string) (*models.FullSessionData, bool) {
	return f.getResult, f.getIsAuth
}

// --- helpers ---

func validVKTokensJSON() []byte {
	data, _ := json.Marshal(models.VKTokensData{
		AccessToken:  "access",
		RefreshToken: "refresh",
		IDToken:      "id-token",
	})
	return data
}

func validPublicInfoJSON(firstName, lastName, avatar string) []byte {
	data, _ := json.Marshal(models.PublicInfo{
		User: models.UserPublicInfo{
			UserID:    "vk-123",
			FirstName: firstName,
			LastName:  lastName,
			Avatar:    avatar,
		},
	})
	return data
}

// --- tests ---

func TestLogin(t *testing.T) {
	t.Parallel()

	exchangeErr := errors.New("vk exchange error")
	publicInfoErr := errors.New("vk public info error")
	createErr := errors.New("db create error")
	updateErr := errors.New("db update error")
	sessionErr := errors.New("session create error")
	checkUserErr := apperrors.UserDoesNotExistError

	existingUser := &models.User{ID: 1, VKID: "vk-123", Name: "Ivan Ivanov", Avatar: "old-avatar"}
	createdUser := &models.User{ID: 2, VKID: "vk-123", Name: "Ivan Ivanov", Avatar: "avatar-url"}
	updatedUser := &models.User{ID: 1, VKID: "vk-123", Name: "Ivan Ivanov", Avatar: "new-avatar"}

	tests := []struct {
		name    string
		vk      *fakeVKApi
		repo    *fakeAuthRepo
		sess    *fakeSessionManager
		wantErr error
		wantNil bool
	}{
		{
			name:    "ExchangeCode error is propagated",
			vk:      &fakeVKApi{exchangeErr: exchangeErr},
			repo:    &fakeAuthRepo{},
			sess:    &fakeSessionManager{},
			wantErr: exchangeErr,
			wantNil: true,
		},
		{
			name:    "invalid VK tokens JSON returns error",
			vk:      &fakeVKApi{exchangeResult: []byte(`{invalid`)},
			repo:    &fakeAuthRepo{},
			sess:    &fakeSessionManager{},
			wantNil: true,
		},
		{
			name: "GetPublicInfo error is propagated",
			vk: &fakeVKApi{
				exchangeResult: validVKTokensJSON(),
				publicErr:      publicInfoErr,
			},
			repo:    &fakeAuthRepo{},
			sess:    &fakeSessionManager{},
			wantErr: publicInfoErr,
			wantNil: true,
		},
		{
			name: "invalid public info JSON returns error",
			vk: &fakeVKApi{
				exchangeResult: validVKTokensJSON(),
				publicResult:   []byte(`{invalid`),
			},
			repo:    &fakeAuthRepo{},
			sess:    &fakeSessionManager{},
			wantNil: true,
		},
		{
			name: "new user: CreateUser error is propagated",
			vk: &fakeVKApi{
				exchangeResult: validVKTokensJSON(),
				publicResult:   validPublicInfoJSON("Ivan", "Ivanov", "avatar-url"),
			},
			repo:    &fakeAuthRepo{checkUserErr: checkUserErr, createErr: createErr},
			sess:    &fakeSessionManager{},
			wantErr: createErr,
			wantNil: true,
		},
		{
			name: "existing user: UpdateUser error is propagated",
			vk: &fakeVKApi{
				exchangeResult: validVKTokensJSON(),
				publicResult:   validPublicInfoJSON("Ivan", "Ivanov", "new-avatar"),
			},
			repo:    &fakeAuthRepo{checkUserResult: existingUser, updateErr: updateErr},
			sess:    &fakeSessionManager{},
			wantErr: updateErr,
			wantNil: true,
		},
		{
			name: "session creation error is propagated",
			vk: &fakeVKApi{
				exchangeResult: validVKTokensJSON(),
				publicResult:   validPublicInfoJSON("Ivan", "Ivanov", "avatar-url"),
			},
			repo:    &fakeAuthRepo{checkUserErr: checkUserErr, createResult: createdUser},
			sess:    &fakeSessionManager{createErr: sessionErr},
			wantErr: sessionErr,
			wantNil: true,
		},
		{
			name: "happy path: new user created and session stored",
			vk: &fakeVKApi{
				exchangeResult: validVKTokensJSON(),
				publicResult:   validPublicInfoJSON("Ivan", "Ivanov", "avatar-url"),
			},
			repo: &fakeAuthRepo{checkUserErr: checkUserErr, createResult: createdUser},
			sess: &fakeSessionManager{},
		},
		{
			name: "happy path: existing user updated",
			vk: &fakeVKApi{
				exchangeResult: validVKTokensJSON(),
				publicResult:   validPublicInfoJSON("Ivan", "Ivanov", "new-avatar"),
			},
			repo: &fakeAuthRepo{checkUserResult: existingUser, updateResult: updatedUser},
			sess: &fakeSessionManager{},
		},
		{
			name: "happy path: existing user no update needed",
			vk: &fakeVKApi{
				exchangeResult: validVKTokensJSON(),
				publicResult:   validPublicInfoJSON("Ivan", "Ivanov", "old-avatar"),
			},
			repo: &fakeAuthRepo{checkUserResult: existingUser},
			sess: &fakeSessionManager{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewAuthUsecases(tt.repo, tt.vk, tt.sess)
			result, err := uc.Login(context.Background(), "session-id",
				&models.LoginRequest{Code: "code"}, time.Hour)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else if tt.wantNil {
				// error occurred but we don't check specific type (e.g., JSON unmarshal)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, tt.sess.createCalled, "session should be created")
			}

			if tt.wantNil {
				assert.Nil(t, result)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()

	removeErr := errors.New("redis error")

	tests := []struct {
		name    string
		sess    *fakeSessionManager
		wantErr error
	}{
		{
			name: "happy path",
			sess: &fakeSessionManager{},
		},
		{
			name:    "session removal error is propagated",
			sess:    &fakeSessionManager{removeErr: removeErr},
			wantErr: removeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewAuthUsecases(&fakeAuthRepo{}, &fakeVKApi{}, tt.sess)
			err := uc.Logout(context.Background(), "session-id")

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckAuth(t *testing.T) {
	t.Parallel()

	sessionData := &models.FullSessionData{
		User: models.User{ID: 1, Name: "Tester"},
	}

	tests := []struct {
		name       string
		sess       *fakeSessionManager
		wantAuth   bool
		wantNilUsr bool
	}{
		{
			name:       "not authenticated returns nil user and false",
			sess:       &fakeSessionManager{getIsAuth: false},
			wantAuth:   false,
			wantNilUsr: true,
		},
		{
			name:     "authenticated returns user and true",
			sess:     &fakeSessionManager{getResult: sessionData, getIsAuth: true},
			wantAuth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewAuthUsecases(&fakeAuthRepo{}, &fakeVKApi{}, tt.sess)
			user, isAuth := uc.CheckAuth(context.Background(), "session-id")

			assert.Equal(t, tt.wantAuth, isAuth)

			if tt.wantNilUsr {
				assert.Nil(t, user)
			} else {
				assert.NotNil(t, user)
				assert.Equal(t, sessionData.User.ID, user.ID)
			}
		})
	}
}

func TestGetUserIDBySessionID(t *testing.T) {
	t.Parallel()

	sessionData := &models.FullSessionData{
		User: models.User{ID: 42},
	}

	sess := &fakeSessionManager{getResult: sessionData, getIsAuth: true}
	uc := NewAuthUsecases(&fakeAuthRepo{}, &fakeVKApi{}, sess)

	id := uc.GetUserIDBySessionID(context.Background(), "session-id")
	assert.Equal(t, 42, id)
}
