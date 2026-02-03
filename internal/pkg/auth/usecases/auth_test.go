package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// --- helpers ---

func validOAuthResult() *models.OAuthResult {
	return &models.OAuthResult{
		ProviderUserID: "vk-123",
		DisplayName:    "Ivan Ivanov",
		AvatarURL:      "avatar-url",
		Email:          "ivan@example.com",
		AccessToken:    "access",
		RefreshToken:   "refresh",
		IDToken:        "id-token",
	}
}

// --- tests ---

func TestLogin(t *testing.T) {
	t.Parallel()

	authErr := errors.New("auth error")
	createErr := errors.New("db create error")
	updateErr := errors.New("db update error")
	sessionErr := errors.New("session create error")
	identityCreateErr := errors.New("identity create error")

	existingUser := &models.User{ID: 1, DisplayName: "Ivan Ivanov", AvatarURL: "old-avatar"}
	createdUser := &models.User{ID: 2, DisplayName: "Ivan Ivanov", AvatarURL: "avatar-url"}
	updatedUser := &models.User{ID: 1, DisplayName: "Ivan Ivanov", AvatarURL: "new-avatar"}

	existingIdentity := &models.UserIdentity{ID: 10, UserID: 1, Provider: "vk", ProviderUserID: "vk-123"}

	tests := []struct {
		name     string
		provider string
		setup    func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, sess *mocks.MockSessionManager)
		wantErr  error
		wantNil  bool
	}{
		{
			name:     "unsupported_provider",
			provider: "unknown",
			setup: func(_ *mocks.MockAuthRepository, _ *mocks.MockIdentityRepository, _ *mocks.MockOAuthProvider, _ *mocks.MockSessionManager) {
			},
			wantErr: apperrors.UnsupportedProviderError,
			wantNil: true,
		},
		{
			name:     "authenticate_error",
			provider: "vk",
			setup: func(_ *mocks.MockAuthRepository, _ *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, _ *mocks.MockSessionManager) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, authErr)
			},
			wantErr: authErr,
			wantNil: true,
		},
		{
			name:     "new_user_create_error",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, _ *mocks.MockSessionManager) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(nil, apperrors.IdentityNotFoundError)
				repo.EXPECT().CreateUserWithIdentity(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, createErr)
			},
			wantErr: createErr,
			wantNil: true,
		},
		{
			name:     "new_user_identity_create_error",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, _ *mocks.MockSessionManager) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(nil, apperrors.IdentityNotFoundError)
				repo.EXPECT().CreateUserWithIdentity(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, identityCreateErr)
			},
			wantErr: identityCreateErr,
			wantNil: true,
		},
		{
			name:     "existing_user_update_error",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, _ *mocks.MockSessionManager) {
				result := validOAuthResult()
				result.AvatarURL = "new-avatar"
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(result, nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(existingIdentity, nil)
				repo.EXPECT().GetUserByID(gomock.Any(), 1).Return(existingUser, nil)
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(nil, updateErr)
			},
			wantErr: updateErr,
			wantNil: true,
		},
		{
			name:     "session_creation_error",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, sess *mocks.MockSessionManager) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(nil, apperrors.IdentityNotFoundError)
				repo.EXPECT().CreateUserWithIdentity(gomock.Any(), gomock.Any(), gomock.Any()).Return(createdUser, nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(sessionErr)
			},
			wantErr: sessionErr,
			wantNil: true,
		},
		{
			name:     "happy_path_new_user",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, sess *mocks.MockSessionManager) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(nil, apperrors.IdentityNotFoundError)
				repo.EXPECT().CreateUserWithIdentity(gomock.Any(), gomock.Any(), gomock.Any()).Return(createdUser, nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), createdUser.ID, gomock.Any()).Return(nil)
			},
		},
		{
			name:     "happy_path_existing_user_updated",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, sess *mocks.MockSessionManager) {
				result := validOAuthResult()
				result.AvatarURL = "new-avatar"
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(result, nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(existingIdentity, nil)
				repo.EXPECT().GetUserByID(gomock.Any(), 1).Return(existingUser, nil)
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(updatedUser, nil)
				idRepo.EXPECT().UpdateLastUsed(gomock.Any(), existingIdentity.ID, gomock.Any()).Return(nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), updatedUser.ID, gomock.Any()).Return(nil)
			},
		},
		{
			name:     "happy_path_no_update_needed",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, sess *mocks.MockSessionManager) {
				result := validOAuthResult()
				result.AvatarURL = "old-avatar"
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(result, nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(existingIdentity, nil)
				repo.EXPECT().GetUserByID(gomock.Any(), 1).Return(existingUser, nil)
				idRepo.EXPECT().UpdateLastUsed(gomock.Any(), existingIdentity.ID, gomock.Any()).Return(nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), existingUser.ID, gomock.Any()).Return(nil)
			},
		},
		{
			name:     "update_last_login_at_fails_but_login_succeeds",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, sess *mocks.MockSessionManager) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(nil, apperrors.IdentityNotFoundError)
				repo.EXPECT().CreateUserWithIdentity(gomock.Any(), gomock.Any(), gomock.Any()).Return(createdUser, nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), createdUser.ID, gomock.Any()).
					Return(errors.New("db timeout"))
			},
		},
		{
			name:     "identity_race_recovery",
			provider: "vk",
			setup: func(repo *mocks.MockAuthRepository, idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider, sess *mocks.MockSessionManager) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				// First FindByProvider: identity not found (race window opens)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(nil, apperrors.IdentityNotFoundError)
				// CreateUserWithIdentity fails: identity was created by concurrent request
				repo.EXPECT().CreateUserWithIdentity(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, apperrors.IdentityAlreadyLinkedError)
				// Retry FindByProvider: identity now exists
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(existingIdentity, nil)
				repo.EXPECT().GetUserByID(gomock.Any(), 1).Return(existingUser, nil)
				sess.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
				repo.EXPECT().UpdateLastLoginAt(gomock.Any(), existingUser.ID, gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockAuthRepository(ctrl)
			idRepo := mocks.NewMockIdentityRepository(ctrl)
			oauth := mocks.NewMockOAuthProvider(ctrl)
			sess := mocks.NewMockSessionManager(ctrl)

			tt.setup(repo, idRepo, oauth, sess)

			providers := map[string]authinterface.OAuthProvider{
				"vk": oauth,
			}

			uc := NewAuthUsecases(repo, idRepo, providers, sess)
			result, err := uc.Login(context.Background(), tt.provider, "session-id",
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
				mocks.NewMockIdentityRepository(ctrl),
				nil,
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
		Provider: "vk",
		User:     models.User{ID: 1, DisplayName: "Tester"},
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
				mocks.NewMockIdentityRepository(ctrl),
				nil,
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
		Provider: "vk",
		User:     models.User{ID: 42},
	}

	ctrl := gomock.NewController(t)
	sess := mocks.NewMockSessionManager(ctrl)
	sess.EXPECT().GetSession(gomock.Any(), "session-id").Return(sessionData, true)

	uc := NewAuthUsecases(
		mocks.NewMockAuthRepository(ctrl),
		mocks.NewMockIdentityRepository(ctrl),
		nil,
		sess,
	)

	id := uc.GetUserIDBySessionID(context.Background(), "session-id")
	assert.Equal(t, 42, id)
}

func TestListIdentities(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")

	tests := []struct {
		name    string
		setup   func(idRepo *mocks.MockIdentityRepository)
		wantErr error
		wantLen int
	}{
		{
			name: "happy_path",
			setup: func(idRepo *mocks.MockIdentityRepository) {
				idRepo.EXPECT().ListByUserID(gomock.Any(), 1).Return([]models.UserIdentity{
					{ID: 1, Provider: "vk"},
					{ID: 2, Provider: "google"},
				}, nil)
			},
			wantLen: 2,
		},
		{
			name: "db_error",
			setup: func(idRepo *mocks.MockIdentityRepository) {
				idRepo.EXPECT().ListByUserID(gomock.Any(), 1).Return(nil, dbErr)
			},
			wantErr: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			idRepo := mocks.NewMockIdentityRepository(ctrl)
			tt.setup(idRepo)

			uc := NewAuthUsecases(
				mocks.NewMockAuthRepository(ctrl),
				idRepo, nil,
				mocks.NewMockSessionManager(ctrl),
			)

			result, err := uc.ListIdentities(context.Background(), 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.wantLen)
			}
		})
	}
}

func TestLinkIdentity(t *testing.T) {
	t.Parallel()

	authErr := errors.New("auth error")
	createErr := errors.New("db create error")

	tests := []struct {
		name     string
		provider string
		setup    func(idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider)
		wantErr  error
	}{
		{
			name:     "unsupported_provider",
			provider: "unknown",
			setup:    func(_ *mocks.MockIdentityRepository, _ *mocks.MockOAuthProvider) {},
			wantErr:  apperrors.UnsupportedProviderError,
		},
		{
			name:     "authenticate_error",
			provider: "vk",
			setup: func(_ *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, authErr)
			},
			wantErr: authErr,
		},
		{
			name:     "already_linked_to_other_user",
			provider: "vk",
			setup: func(idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(&models.UserIdentity{ID: 10, UserID: 99}, nil)
			},
			wantErr: apperrors.IdentityAlreadyLinkedError,
		},
		{
			name:     "already_linked_to_same_user",
			provider: "vk",
			setup: func(idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(&models.UserIdentity{ID: 10, UserID: 1}, nil)
			},
		},
		{
			name:     "create_error",
			provider: "vk",
			setup: func(idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(nil, apperrors.IdentityNotFoundError)
				idRepo.EXPECT().CreateIdentity(gomock.Any(), gomock.Any()).Return(createErr)
			},
			wantErr: createErr,
		},
		{
			name:     "happy_path",
			provider: "vk",
			setup: func(idRepo *mocks.MockIdentityRepository, oauth *mocks.MockOAuthProvider) {
				oauth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(validOAuthResult(), nil)
				idRepo.EXPECT().FindByProvider(gomock.Any(), "vk", "vk-123").
					Return(nil, apperrors.IdentityNotFoundError)
				idRepo.EXPECT().CreateIdentity(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			idRepo := mocks.NewMockIdentityRepository(ctrl)
			oauth := mocks.NewMockOAuthProvider(ctrl)

			tt.setup(idRepo, oauth)

			providers := map[string]authinterface.OAuthProvider{"vk": oauth}
			uc := NewAuthUsecases(
				mocks.NewMockAuthRepository(ctrl),
				idRepo, providers,
				mocks.NewMockSessionManager(ctrl),
			)

			err := uc.LinkIdentity(context.Background(), 1, tt.provider, &models.LoginRequest{Code: "code"})

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnlinkIdentity(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	deleteErr := errors.New("delete error")

	tests := []struct {
		name    string
		setup   func(idRepo *mocks.MockIdentityRepository)
		wantErr error
	}{
		{
			name: "list_error",
			setup: func(idRepo *mocks.MockIdentityRepository) {
				idRepo.EXPECT().ListByUserID(gomock.Any(), 1).Return(nil, dbErr)
			},
			wantErr: dbErr,
		},
		{
			name: "last_identity",
			setup: func(idRepo *mocks.MockIdentityRepository) {
				idRepo.EXPECT().ListByUserID(gomock.Any(), 1).Return([]models.UserIdentity{
					{ID: 1, Provider: "vk"},
				}, nil)
			},
			wantErr: apperrors.LastIdentityError,
		},
		{
			name: "delete_error",
			setup: func(idRepo *mocks.MockIdentityRepository) {
				idRepo.EXPECT().ListByUserID(gomock.Any(), 1).Return([]models.UserIdentity{
					{ID: 1, Provider: "vk"},
					{ID: 2, Provider: "google"},
				}, nil)
				idRepo.EXPECT().DeleteByUserAndProvider(gomock.Any(), 1, "vk").Return(deleteErr)
			},
			wantErr: deleteErr,
		},
		{
			name: "happy_path",
			setup: func(idRepo *mocks.MockIdentityRepository) {
				idRepo.EXPECT().ListByUserID(gomock.Any(), 1).Return([]models.UserIdentity{
					{ID: 1, Provider: "vk"},
					{ID: 2, Provider: "google"},
				}, nil)
				idRepo.EXPECT().DeleteByUserAndProvider(gomock.Any(), 1, "vk").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			idRepo := mocks.NewMockIdentityRepository(ctrl)
			tt.setup(idRepo)

			uc := NewAuthUsecases(
				mocks.NewMockAuthRepository(ctrl),
				idRepo, nil,
				mocks.NewMockSessionManager(ctrl),
			)

			err := uc.UnlinkIdentity(context.Background(), 1, "vk")

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
