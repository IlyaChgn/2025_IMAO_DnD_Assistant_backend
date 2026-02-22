package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/mocks"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

// --- helpers ---

func validCharacterBase(userID string) *models.CharacterBase {
	return &models.CharacterBase{
		ID:      primitive.NewObjectID(),
		UserID:  userID,
		Version: 1,
		Name:    "Test Character",
		Race:    "Human",
		Classes: []models.ClassEntry{{ClassName: "fighter", Level: 5}},
		AbilityScores: models.AbilityScores{
			Str: 16, Dex: 14, Con: 12, Int: 10, Wis: 13, Cha: 8,
		},
		BaseSpeed: 30,
		Proficiencies: models.Proficiencies{
			Skills:       []string{"athletics"},
			SavingThrows: []models.AbilityType{"STR", "CON"},
		},
	}
}

func validCharacterBaseWithAvatar(userID, avatarURL string) *models.CharacterBase {
	char := validCharacterBase(userID)
	char.Avatar = &models.CharacterAvatar{Url: avatarURL}
	return char
}

// --- TestCreate ---

func TestCreate(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("insert failure")

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockCharacterBaseRepository)
		wantErr error
	}{
		{
			name: "happy path",
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "repo error propagated",
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterBaseRepository(ctrl)
			s3 := mocks.NewMockAvatarS3Manager(ctrl)
			tt.setup(repo)

			uc := NewCharacterBaseUsecases(repo, s3)
			err := uc.Create(context.Background(), validCharacterBase("1"))

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- TestGetByID ---

func TestGetByID(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		id      string
		userID  int
		setup   func(repo *mocks.MockCharacterBaseRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:    "empty id returns InvalidInputError",
			id:      "",
			userID:  1,
			setup:   func(_ *mocks.MockCharacterBaseRepository) {},
			wantErr: apperrors.InvalidInputError,
			wantNil: true,
		},
		{
			name:   "not found returns nil nil",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().GetByID(gomock.Any(), "abc123").Return(nil, nil)
			},
			wantNil: true,
		},
		{
			name:   "wrong owner returns PermissionDeniedError",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				char := validCharacterBase("999") // different user
				repo.EXPECT().GetByID(gomock.Any(), "abc123").Return(char, nil)
			},
			wantErr: apperrors.PermissionDeniedError,
			wantNil: true,
		},
		{
			name:   "happy path returns character",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				char := validCharacterBase("1")
				repo.EXPECT().GetByID(gomock.Any(), "abc123").Return(char, nil)
			},
		},
		{
			name:   "repo error propagated",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().GetByID(gomock.Any(), "abc123").Return(nil, repoErr)
			},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterBaseRepository(ctrl)
			s3 := mocks.NewMockAvatarS3Manager(ctrl)
			tt.setup(repo)

			uc := NewCharacterBaseUsecases(repo, s3)
			result, err := uc.GetByID(context.Background(), tt.id, tt.userID)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}
}

// --- TestGetComputed ---

func TestGetComputed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		id          string
		userID      int
		setup       func(repo *mocks.MockCharacterBaseRepository)
		wantErr     error
		wantNilChar bool
	}{
		{
			name:   "not found returns nil nil nil",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().GetByID(gomock.Any(), "abc123").Return(nil, nil)
			},
			wantNilChar: true,
		},
		{
			name:   "happy path returns char and derived stats",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				char := validCharacterBase("1")
				repo.EXPECT().GetByID(gomock.Any(), "abc123").Return(char, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterBaseRepository(ctrl)
			s3 := mocks.NewMockAvatarS3Manager(ctrl)
			tt.setup(repo)

			uc := NewCharacterBaseUsecases(repo, s3)
			char, derived, err := uc.GetComputed(context.Background(), tt.id, tt.userID)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNilChar {
				assert.Nil(t, char)
				assert.Nil(t, derived)
			} else {
				assert.NotNil(t, char)
				assert.NotNil(t, derived)
			}
		})
	}
}

// --- TestUpdate ---

func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockCharacterBaseRepository)
		wantErr error
	}{
		{
			name: "happy path",
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().Update(gomock.Any(), gomock.Any(), 1).Return(nil)
			},
		},
		{
			name: "version conflict",
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().Update(gomock.Any(), gomock.Any(), 1).Return(apperrors.VersionConflictErr)
			},
			wantErr: apperrors.VersionConflictErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterBaseRepository(ctrl)
			s3 := mocks.NewMockAvatarS3Manager(ctrl)
			tt.setup(repo)

			uc := NewCharacterBaseUsecases(repo, s3)
			char := validCharacterBase("1")
			err := uc.Update(context.Background(), char, 1, 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- TestDelete ---

func TestDelete(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("delete failure")

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockCharacterBaseRepository)
		wantErr error
	}{
		{
			name: "happy path",
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().Delete(gomock.Any(), "abc123", "1").Return(nil)
			},
		},
		{
			name: "repo error propagated",
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().Delete(gomock.Any(), "abc123", "1").Return(repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterBaseRepository(ctrl)
			s3 := mocks.NewMockAvatarS3Manager(ctrl)
			tt.setup(repo)

			uc := NewCharacterBaseUsecases(repo, s3)
			err := uc.Delete(context.Background(), "abc123", 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- TestList ---

func TestList(t *testing.T) {
	t.Parallel()

	expected := []*models.CharacterBase{validCharacterBase("1")}

	tests := []struct {
		name      string
		page      int
		size      int
		setup     func(repo *mocks.MockCharacterBaseRepository)
		wantErr   error
		wantTotal int64
	}{
		{
			name:    "negative page returns StartPosSizeError",
			page:    -1,
			size:    10,
			setup:   func(_ *mocks.MockCharacterBaseRepository) {},
			wantErr: apperrors.StartPosSizeError,
		},
		{
			name:    "zero size returns StartPosSizeError",
			page:    0,
			size:    0,
			setup:   func(_ *mocks.MockCharacterBaseRepository) {},
			wantErr: apperrors.StartPosSizeError,
		},
		{
			name: "happy path returns list and total",
			page: 0,
			size: 20,
			setup: func(repo *mocks.MockCharacterBaseRepository) {
				repo.EXPECT().List(gomock.Any(), "1", 0, 20, "").Return(expected, int64(1), nil)
			},
			wantTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterBaseRepository(ctrl)
			s3 := mocks.NewMockAvatarS3Manager(ctrl)
			tt.setup(repo)

			uc := NewCharacterBaseUsecases(repo, s3)
			result, total, err := uc.List(context.Background(), 1, tt.page, tt.size, "")

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantTotal, total)
			}
		})
	}
}

// --- TestUploadAvatar ---

func TestUploadAvatar(t *testing.T) {
	t.Parallel()

	smallFile := make([]byte, 100)
	largeFile := make([]byte, maxAvatarSize+1)
	s3Err := errors.New("s3 failure")
	dbErr := errors.New("db failure")
	const testID = "507f1f77bcf86cd799439011"
	const oldAvatarURL = "https://encounterium.ru/character-avatars/old-avatar.webp"

	tests := []struct {
		name    string
		id      string
		userID  int
		data    []byte
		setup   func(repo *mocks.MockCharacterBaseRepository, s3 *mocks.MockAvatarS3Manager)
		wantErr error
	}{
		{
			name:   "file too large returns AvatarTooLargeErr",
			id:     testID,
			userID: 1,
			data:   largeFile,
			setup: func(_ *mocks.MockCharacterBaseRepository, _ *mocks.MockAvatarS3Manager) {
			},
			wantErr: apperrors.AvatarTooLargeErr,
		},
		{
			name:   "character not found returns CharacterNotFoundErr",
			id:     testID,
			userID: 1,
			data:   smallFile,
			setup: func(repo *mocks.MockCharacterBaseRepository, _ *mocks.MockAvatarS3Manager) {
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(nil, nil)
			},
			wantErr: apperrors.CharacterNotFoundErr,
		},
		{
			name:   "S3 upload fails returns AvatarUploadErr",
			id:     testID,
			userID: 1,
			data:   smallFile,
			setup: func(repo *mocks.MockCharacterBaseRepository, s3 *mocks.MockAvatarS3Manager) {
				char := validCharacterBase("1")
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(char, nil)
				s3.EXPECT().UploadAvatar(gomock.Any(), smallFile, gomock.Any()).Return("", s3Err)
			},
			wantErr: apperrors.AvatarUploadErr,
		},
		{
			name:   "DB update fails cleans up S3 and returns error",
			id:     testID,
			userID: 1,
			data:   smallFile,
			setup: func(repo *mocks.MockCharacterBaseRepository, s3 *mocks.MockAvatarS3Manager) {
				char := validCharacterBase("1")
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(char, nil)
				s3.EXPECT().UploadAvatar(gomock.Any(), smallFile, gomock.Any()).Return("https://example.com/new.webp", nil)
				repo.EXPECT().UpdateAvatarURL(gomock.Any(), testID, "1", "https://example.com/new.webp").Return(dbErr)
				// Verify cleanup: S3 delete is called for the new object
				s3.EXPECT().DeleteAvatar(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: dbErr,
		},
		{
			name:   "happy path without old avatar",
			id:     testID,
			userID: 1,
			data:   smallFile,
			setup: func(repo *mocks.MockCharacterBaseRepository, s3 *mocks.MockAvatarS3Manager) {
				char := validCharacterBase("1") // no Avatar field
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(char, nil)
				s3.EXPECT().UploadAvatar(gomock.Any(), smallFile, gomock.Any()).Return("https://example.com/new.webp", nil)
				repo.EXPECT().UpdateAvatarURL(gomock.Any(), testID, "1", "https://example.com/new.webp").Return(nil)
				// No S3 delete expected — no old avatar
			},
		},
		{
			name:   "happy path replaces old avatar",
			id:     testID,
			userID: 1,
			data:   smallFile,
			setup: func(repo *mocks.MockCharacterBaseRepository, s3 *mocks.MockAvatarS3Manager) {
				char := validCharacterBaseWithAvatar("1", oldAvatarURL)
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(char, nil)
				s3.EXPECT().UploadAvatar(gomock.Any(), smallFile, gomock.Any()).Return("https://example.com/new.webp", nil)
				repo.EXPECT().UpdateAvatarURL(gomock.Any(), testID, "1", "https://example.com/new.webp").Return(nil)
				// Verify old avatar S3 object deleted
				s3.EXPECT().DeleteAvatar(gomock.Any(), "old-avatar.webp").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterBaseRepository(ctrl)
			s3 := mocks.NewMockAvatarS3Manager(ctrl)
			tt.setup(repo, s3)

			uc := NewCharacterBaseUsecases(repo, s3)
			url, err := uc.UploadAvatar(context.Background(), tt.id, tt.userID, tt.data)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, url)
			}
		})
	}
}

// --- TestDeleteAvatar ---

func TestDeleteAvatar(t *testing.T) {
	t.Parallel()

	s3Err := errors.New("s3 failure")
	repoErr := errors.New("db failure")
	const testID = "507f1f77bcf86cd799439011"
	const avatarURL = "https://encounterium.ru/character-avatars/test-avatar.webp"

	tests := []struct {
		name    string
		id      string
		userID  int
		setup   func(repo *mocks.MockCharacterBaseRepository, s3 *mocks.MockAvatarS3Manager)
		wantErr error
	}{
		{
			name:   "character not found returns CharacterNotFoundErr",
			id:     testID,
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository, _ *mocks.MockAvatarS3Manager) {
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(nil, nil)
			},
			wantErr: apperrors.CharacterNotFoundErr,
		},
		{
			name:   "char with avatar S3 ok",
			id:     testID,
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository, s3 *mocks.MockAvatarS3Manager) {
				char := validCharacterBaseWithAvatar("1", avatarURL)
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(char, nil)
				s3.EXPECT().DeleteAvatar(gomock.Any(), "test-avatar.webp").Return(nil)
				repo.EXPECT().ClearAvatar(gomock.Any(), testID, "1").Return(nil)
			},
		},
		{
			name:   "char with avatar S3 fails continues to clear DB",
			id:     testID,
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository, s3 *mocks.MockAvatarS3Manager) {
				char := validCharacterBaseWithAvatar("1", avatarURL)
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(char, nil)
				s3.EXPECT().DeleteAvatar(gomock.Any(), "test-avatar.webp").Return(s3Err)
				// ClearAvatar still called despite S3 error
				repo.EXPECT().ClearAvatar(gomock.Any(), testID, "1").Return(nil)
			},
		},
		{
			name:   "char without avatar skips S3 and clears DB",
			id:     testID,
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository, _ *mocks.MockAvatarS3Manager) {
				char := validCharacterBase("1") // no Avatar
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(char, nil)
				// No S3 delete expected
				repo.EXPECT().ClearAvatar(gomock.Any(), testID, "1").Return(nil)
			},
		},
		{
			name:   "ClearAvatar repo error propagated",
			id:     testID,
			userID: 1,
			setup: func(repo *mocks.MockCharacterBaseRepository, _ *mocks.MockAvatarS3Manager) {
				char := validCharacterBase("1")
				repo.EXPECT().GetByID(gomock.Any(), testID).Return(char, nil)
				repo.EXPECT().ClearAvatar(gomock.Any(), testID, "1").Return(repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterBaseRepository(ctrl)
			s3 := mocks.NewMockAvatarS3Manager(ctrl)
			tt.setup(repo, s3)

			uc := NewCharacterBaseUsecases(repo, s3)
			err := uc.DeleteAvatar(context.Background(), tt.id, tt.userID)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
