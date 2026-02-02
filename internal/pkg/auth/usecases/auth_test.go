package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

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

	existingUser := &models.User{ID: 1, VKID: "vk-123", Name: "Ivan Ivanov", Avatar: "old-avatar"}
	createdUser := &models.User{ID: 2, VKID: "vk-123", Name: "Ivan Ivanov", Avatar: "avatar-url"}
	updatedUser := &models.User{ID: 1, VKID: "vk-123", Name: "Ivan Ivanov", Avatar: "new-avatar"}

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockAuthRepository, vk *mocks.MockVKApi, sess *mocks.MockSessionManager)
		wantErr error
		wantNil bool
	}{
		{
			name: "exchange_code_error",
			setup: func(_ *mocks.MockAuthRepository, vk *mocks.MockVKApi, _ *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(nil, exchangeErr)
			},
			wantErr: exchangeErr,
			wantNil: true,
		},
		{
			name: "invalid_vk_tokens_json",
			setup: func(_ *mocks.MockAuthRepository, vk *mocks.MockVKApi, _ *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return([]byte(`{invalid`), nil)
			},
			wantNil: true,
		},
		{
			name: "get_public_info_error",
			setup: func(_ *mocks.MockAuthRepository, vk *mocks.MockVKApi, _ *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").Return(nil, publicInfoErr)
			},
			wantErr: publicInfoErr,
			wantNil: true,
		},
		{
			name: "invalid_public_info_json",
			setup: func(_ *mocks.MockAuthRepository, vk *mocks.MockVKApi, _ *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").Return([]byte(`{invalid`), nil)
			},
			wantNil: true,
		},
		{
			name: "new_user_create_error",
			setup: func(repo *mocks.MockAuthRepository, vk *mocks.MockVKApi, _ *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").
					Return(validPublicInfoJSON("Ivan", "Ivanov", "avatar-url"), nil)
				repo.EXPECT().CheckUser(gomock.Any(), "vk-123").Return(nil, errors.New("not found"))
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(nil, createErr)
			},
			wantErr: createErr,
			wantNil: true,
		},
		{
			name: "existing_user_update_error",
			setup: func(repo *mocks.MockAuthRepository, vk *mocks.MockVKApi, _ *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").
					Return(validPublicInfoJSON("Ivan", "Ivanov", "new-avatar"), nil)
				repo.EXPECT().CheckUser(gomock.Any(), "vk-123").Return(existingUser, nil)
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(nil, updateErr)
			},
			wantErr: updateErr,
			wantNil: true,
		},
		{
			name: "session_creation_error",
			setup: func(repo *mocks.MockAuthRepository, vk *mocks.MockVKApi, sess *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").
					Return(validPublicInfoJSON("Ivan", "Ivanov", "avatar-url"), nil)
				repo.EXPECT().CheckUser(gomock.Any(), "vk-123").Return(nil, errors.New("not found"))
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(createdUser, nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sessionErr)
			},
			wantErr: sessionErr,
			wantNil: true,
		},
		{
			name: "happy_path_new_user",
			setup: func(repo *mocks.MockAuthRepository, vk *mocks.MockVKApi, sess *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").
					Return(validPublicInfoJSON("Ivan", "Ivanov", "avatar-url"), nil)
				repo.EXPECT().CheckUser(gomock.Any(), "vk-123").Return(nil, errors.New("not found"))
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(createdUser, nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), createdUser.ID, gomock.Any()).Return(nil)
			},
		},
		{
			name: "happy_path_existing_user_updated",
			setup: func(repo *mocks.MockAuthRepository, vk *mocks.MockVKApi, sess *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").
					Return(validPublicInfoJSON("Ivan", "Ivanov", "new-avatar"), nil)
				repo.EXPECT().CheckUser(gomock.Any(), "vk-123").Return(existingUser, nil)
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(updatedUser, nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), updatedUser.ID, gomock.Any()).Return(nil)
			},
		},
		{
			name: "happy_path_no_update_needed",
			setup: func(repo *mocks.MockAuthRepository, vk *mocks.MockVKApi, sess *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").
					Return(validPublicInfoJSON("Ivan", "Ivanov", "old-avatar"), nil)
				repo.EXPECT().CheckUser(gomock.Any(), "vk-123").Return(existingUser, nil)
				// No UpdateUser call expected — gomock will fail if it's called.
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), existingUser.ID, gomock.Any()).Return(nil)
			},
		},
		{
			name: "update_last_login_at_fails_but_login_succeeds",
			setup: func(repo *mocks.MockAuthRepository, vk *mocks.MockVKApi, sess *mocks.MockSessionManager) {
				vk.EXPECT().ExchangeCode(gomock.Any(), gomock.Any()).Return(validVKTokensJSON(), nil)
				vk.EXPECT().GetPublicInfo(gomock.Any(), "id-token").
					Return(validPublicInfoJSON("Ivan", "Ivanov", "avatar-url"), nil)
				repo.EXPECT().CheckUser(gomock.Any(), "vk-123").Return(nil, errors.New("not found"))
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(createdUser, nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), createdUser.ID, gomock.Any()).
					Return(errors.New("db timeout"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockAuthRepository(ctrl)
			vk := mocks.NewMockVKApi(ctrl)
			sess := mocks.NewMockSessionManager(ctrl)

			tt.setup(repo, vk, sess)

			uc := NewAuthUsecases(repo, vk, sess)
			result, err := uc.Login(context.Background(), "session-id",
				&models.LoginRequest{Code: "code"}, time.Hour)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else if tt.wantNil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
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
		setup   func(sess *mocks.MockSessionManager)
		wantErr error
	}{
		{
			name: "happy_path",
			setup: func(sess *mocks.MockSessionManager) {
				sess.EXPECT().RemoveSession(gomock.Any(), "session-id").Return(nil)
			},
		},
		{
			name: "removal_error",
			setup: func(sess *mocks.MockSessionManager) {
				sess.EXPECT().RemoveSession(gomock.Any(), "session-id").Return(removeErr)
			},
			wantErr: removeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sess := mocks.NewMockSessionManager(ctrl)
			tt.setup(sess)

			uc := NewAuthUsecases(
				mocks.NewMockAuthRepository(ctrl),
				mocks.NewMockVKApi(ctrl),
				sess,
			)
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
		setup      func(sess *mocks.MockSessionManager)
		wantAuth   bool
		wantNilUsr bool
	}{
		{
			name: "not_authenticated",
			setup: func(sess *mocks.MockSessionManager) {
				sess.EXPECT().GetSession(gomock.Any(), "session-id").Return(nil, false)
			},
			wantAuth:   false,
			wantNilUsr: true,
		},
		{
			name: "authenticated",
			setup: func(sess *mocks.MockSessionManager) {
				sess.EXPECT().GetSession(gomock.Any(), "session-id").Return(sessionData, true)
			},
			wantAuth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sess := mocks.NewMockSessionManager(ctrl)
			tt.setup(sess)

			uc := NewAuthUsecases(
				mocks.NewMockAuthRepository(ctrl),
				mocks.NewMockVKApi(ctrl),
				sess,
			)
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

	ctrl := gomock.NewController(t)
	sess := mocks.NewMockSessionManager(ctrl)
	sess.EXPECT().GetSession(gomock.Any(), "session-id").Return(sessionData, true)

	uc := NewAuthUsecases(
		mocks.NewMockAuthRepository(ctrl),
		mocks.NewMockVKApi(ctrl),
		sess,
	)

	id := uc.GetUserIDBySessionID(context.Background(), "session-id")
	assert.Equal(t, 42, id)
}
