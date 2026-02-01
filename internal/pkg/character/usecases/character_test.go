package usecases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/stretchr/testify/assert"
)

// --- fake repository ---

type fakeCharacterRepo struct {
	listResult      []*models.CharacterShort
	listErr         error
	characterResult *models.Character
	characterErr    error
	addErr          error

	addCalled bool
}

func (f *fakeCharacterRepo) GetCharactersList(_ context.Context, _, _, _ int,
	_ models.SearchParams) ([]*models.CharacterShort, error) {
	return f.listResult, f.listErr
}

func (f *fakeCharacterRepo) GetCharacterByMongoId(_ context.Context, _ string) (*models.Character, error) {
	return f.characterResult, f.characterErr
}

func (f *fakeCharacterRepo) AddCharacter(_ context.Context, _ models.CharacterRaw, _ int) error {
	f.addCalled = true
	return f.addErr
}

// --- fake multipart.File ---

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
		repo    *fakeCharacterRepo
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			size:    10,
			start:   -1,
			repo:    &fakeCharacterRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			size:    0,
			start:   0,
			repo:    &fakeCharacterRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:  "happy path returns list",
			size:  10,
			start: 0,
			repo:  &fakeCharacterRepo{listResult: expected},
		},
		{
			name:    "repo error is propagated",
			size:    10,
			start:   0,
			repo:    &fakeCharacterRepo{listErr: repoErr},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewCharacterUsecases(tt.repo)
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
		repo    *fakeCharacterRepo
		wantErr error
		wantNil bool
	}{
		{
			name:    "empty id returns InvalidInputError",
			id:      "",
			userID:  1,
			repo:    &fakeCharacterRepo{},
			wantErr: apperrors.InvalidInputError,
			wantNil: true,
		},
		{
			name:   "happy path returns own character",
			id:     "abc123",
			userID: 1,
			repo:   &fakeCharacterRepo{characterResult: ownedChar},
		},
		{
			name:   "public character accessible by any user",
			id:     "abc123",
			userID: 42,
			repo:   &fakeCharacterRepo{characterResult: publicChar},
		},
		{
			name:    "other user's character returns PermissionDeniedError",
			id:      "abc123",
			userID:  1,
			repo:    &fakeCharacterRepo{characterResult: otherUserChar},
			wantErr: apperrors.PermissionDeniedError,
			wantNil: true,
		},
		{
			name:    "repo error is propagated",
			id:      "abc123",
			userID:  1,
			repo:    &fakeCharacterRepo{characterErr: repoErr},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewCharacterUsecases(tt.repo)
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
		repo     *fakeCharacterRepo
		wantErr  error
		wantAdd  bool
	}{
		{
			name:     "invalid JSON returns InvalidJSONError",
			fileData: []byte(`{not valid json`),
			repo:     &fakeCharacterRepo{},
			wantErr:  apperrors.InvalidJSONError,
		},
		{
			name:     "empty data field returns InvalidInputError",
			fileData: []byte(emptyDataJSON),
			repo:     &fakeCharacterRepo{},
			wantErr:  apperrors.InvalidInputError,
		},
		{
			name:     "happy path calls repo",
			fileData: []byte(validJSON),
			repo:     &fakeCharacterRepo{},
			wantAdd:  true,
		},
		{
			name:     "repo error is propagated",
			fileData: []byte(validJSON),
			repo:     &fakeCharacterRepo{addErr: repoErr},
			wantErr:  repoErr,
			wantAdd:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewCharacterUsecases(tt.repo)
			err := uc.AddCharacter(context.Background(), newFakeFile(tt.fileData), 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantAdd, tt.repo.addCalled)
		})
	}
}
