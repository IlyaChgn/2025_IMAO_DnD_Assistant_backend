package usecases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// --- fake multipart.File (not a domain interface — keep as hand-written fake) ---

type fakeFile struct {
	io.ReadSeeker
}

func (f *fakeFile) ReadAt(p []byte, off int64) (int, error) {
	if seeker, ok := f.ReadSeeker.(io.ReaderAt); ok {
		return seeker.ReadAt(p, off)
	}
	return 0, errors.New("ReadAt not supported")
}

func (f *fakeFile) Close() error { return nil }

func newFakeFile(data []byte) *fakeFile {
	return &fakeFile{ReadSeeker: bytes.NewReader(data)}
}

// --- tests ---

func TestGetCharactersList(t *testing.T) {
	t.Parallel()

	expected := []*models.CharacterShort{{}}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		size    int
		start   int
		setup   func(repo *mocks.MockCharacterRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			size:    10,
			start:   -1,
			setup:   func(_ *mocks.MockCharacterRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			size:    0,
			start:   0,
			setup:   func(_ *mocks.MockCharacterRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:  "happy path returns list",
			size:  10,
			start: 0,
			setup: func(repo *mocks.MockCharacterRepository) {
				repo.EXPECT().GetCharactersList(gomock.Any(), 10, 0, 1, models.SearchParams{}).
					Return(expected, nil)
			},
		},
		{
			name:  "repo error is propagated",
			size:  10,
			start: 0,
			setup: func(repo *mocks.MockCharacterRepository) {
				repo.EXPECT().GetCharactersList(gomock.Any(), 10, 0, 1, models.SearchParams{}).
					Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterRepository(ctrl)
			tt.setup(repo)

			uc := NewCharacterUsecases(repo)
			result, err := uc.GetCharactersList(context.Background(), tt.size, tt.start, 1, models.SearchParams{})

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else if tt.wantErr == nil {
				assert.Equal(t, expected, result)
			}
		})
	}
}

func TestGetCharacterByMongoId(t *testing.T) {
	t.Parallel()

	ownedChar := &models.Character{UserID: "1"}
	publicChar := &models.Character{UserID: "*"}
	otherUserChar := &models.Character{UserID: "999"}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		id      string
		userID  int
		setup   func(repo *mocks.MockCharacterRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:    "empty id returns InvalidInputError",
			id:      "",
			userID:  1,
			setup:   func(_ *mocks.MockCharacterRepository) {},
			wantErr: apperrors.InvalidInputError,
			wantNil: true,
		},
		{
			name:   "happy path returns own character",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterRepository) {
				repo.EXPECT().GetCharacterByMongoId(gomock.Any(), "abc123").Return(ownedChar, nil)
			},
		},
		{
			name:   "public character accessible by any user",
			id:     "abc123",
			userID: 42,
			setup: func(repo *mocks.MockCharacterRepository) {
				repo.EXPECT().GetCharacterByMongoId(gomock.Any(), "abc123").Return(publicChar, nil)
			},
		},
		{
			name:   "other user's character returns PermissionDeniedError",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterRepository) {
				repo.EXPECT().GetCharacterByMongoId(gomock.Any(), "abc123").Return(otherUserChar, nil)
			},
			wantErr: apperrors.PermissionDeniedError,
			wantNil: true,
		},
		{
			name:   "repo error is propagated",
			id:     "abc123",
			userID: 1,
			setup: func(repo *mocks.MockCharacterRepository) {
				repo.EXPECT().GetCharacterByMongoId(gomock.Any(), "abc123").Return(nil, repoErr)
			},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterRepository(ctrl)
			tt.setup(repo)

			uc := NewCharacterUsecases(repo)
			result, err := uc.GetCharacterByMongoId(context.Background(), tt.id, tt.userID)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else if tt.wantErr == nil {
				assert.NotNil(t, result)
			}
		})
	}
}

func TestAddCharacter(t *testing.T) {
	t.Parallel()

	validJSON := `{"data":"some-data","jsonType":"character","version":"1"}`
	emptyDataJSON := `{"data":"","jsonType":"character","version":"1"}`
	repoErr := errors.New("insert failure")

	tests := []struct {
		name     string
		fileData []byte
		setup    func(repo *mocks.MockCharacterRepository)
		wantErr  error
	}{
		{
			name:     "invalid JSON returns InvalidJSONError",
			fileData: []byte(`{not valid json`),
			setup:    func(_ *mocks.MockCharacterRepository) {},
			wantErr:  apperrors.InvalidJSONError,
		},
		{
			name:     "empty data field returns InvalidInputError",
			fileData: []byte(emptyDataJSON),
			setup:    func(_ *mocks.MockCharacterRepository) {},
			wantErr:  apperrors.InvalidInputError,
		},
		{
			name:     "happy path calls repo",
			fileData: []byte(validJSON),
			setup: func(repo *mocks.MockCharacterRepository) {
				repo.EXPECT().AddCharacter(gomock.Any(), gomock.Any(), 1).Return(nil)
			},
		},
		{
			name:     "repo error is propagated",
			fileData: []byte(validJSON),
			setup: func(repo *mocks.MockCharacterRepository) {
				repo.EXPECT().AddCharacter(gomock.Any(), gomock.Any(), 1).Return(repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockCharacterRepository(ctrl)
			tt.setup(repo)

			uc := NewCharacterUsecases(repo)
			err := uc.AddCharacter(context.Background(), newFakeFile(tt.fileData), 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
