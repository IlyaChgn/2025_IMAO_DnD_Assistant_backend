package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/stretchr/testify/assert"
)

// --- fake repository ---

type fakeMapTilesRepo struct {
	categories []*models.MapTileCategory
	err        error
}

func (f *fakeMapTilesRepo) GetCategories(_ context.Context, _ int) ([]*models.MapTileCategory, error) {
	return f.categories, f.err
}

// --- tests ---

func TestGetCategories(t *testing.T) {
	t.Parallel()

	expected := []*models.MapTileCategory{{Name: "terrain"}}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		userID  int
		repo    *fakeMapTilesRepo
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative userID returns InvalidUserIDError",
			userID:  -1,
			repo:    &fakeMapTilesRepo{},
			wantErr: apperrors.InvalidUserIDError,
			wantNil: true,
		},
		{
			name:   "happy path returns categories",
			userID: 1,
			repo:   &fakeMapTilesRepo{categories: expected},
		},
		{
			name:    "repo error is propagated",
			userID:  1,
			repo:    &fakeMapTilesRepo{err: repoErr},
			wantErr: repoErr,
			wantNil: true,
		},
		{
			name:   "zero userID is valid",
			userID: 0,
			repo:   &fakeMapTilesRepo{categories: expected},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewMapTilesUsecases(tt.repo)
			result, err := uc.GetCategories(context.Background(), tt.userID)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, expected, result)
			}
		})
	}
}
