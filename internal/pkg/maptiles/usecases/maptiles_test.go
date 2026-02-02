package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetCategories(t *testing.T) {
	t.Parallel()

	expected := []*models.MapTileCategory{{Name: "terrain"}}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		userID  int
		setup   func(repo *mocks.MockMapTilesRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative userID returns InvalidUserIDError",
			userID:  -1,
			setup:   func(_ *mocks.MockMapTilesRepository) {},
			wantErr: apperrors.InvalidUserIDError,
			wantNil: true,
		},
		{
			name:   "happy path returns categories",
			userID: 1,
			setup: func(repo *mocks.MockMapTilesRepository) {
				repo.EXPECT().GetCategories(gomock.Any(), 1).Return(expected, nil)
			},
		},
		{
			name:   "repo error is propagated",
			userID: 1,
			setup: func(repo *mocks.MockMapTilesRepository) {
				repo.EXPECT().GetCategories(gomock.Any(), 1).Return(nil, repoErr)
			},
			wantErr: repoErr,
			wantNil: true,
		},
		{
			name:   "zero userID is valid",
			userID: 0,
			setup: func(repo *mocks.MockMapTilesRepository) {
				repo.EXPECT().GetCategories(gomock.Any(), 0).Return(expected, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMapTilesRepository(ctrl)
			tt.setup(repo)

			uc := NewMapTilesUsecases(repo)
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
