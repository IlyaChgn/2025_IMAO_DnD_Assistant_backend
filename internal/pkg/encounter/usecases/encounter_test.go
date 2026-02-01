package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetEncountersList(t *testing.T) {
	t.Parallel()

	expected := &models.EncountersList{}
	repoErr := errors.New("db failure")

	tests := []struct {
		name       string
		size       int
		start      int
		search     *models.SearchParams
		setup      func(repo *mocks.MockEncounterRepository)
		wantErr    error
		wantNil    bool
		wantResult *models.EncountersList
	}{
		{
			name:    "negative start returns StartPosSizeError",
			size:    10,
			start:   -1,
			search:  &models.SearchParams{},
			setup:   func(_ *mocks.MockEncounterRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			size:    0,
			start:   0,
			search:  &models.SearchParams{},
			setup:   func(_ *mocks.MockEncounterRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:   "happy path without search",
			size:   10,
			start:  0,
			search: &models.SearchParams{},
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().GetEncountersList(gomock.Any(), 10, 0, 1).
					Return(expected, nil)
			},
			wantResult: expected,
		},
		{
			name:   "happy path with search delegates to search method",
			size:   10,
			start:  0,
			search: &models.SearchParams{Value: "dragon"},
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().GetEncountersListWithSearch(gomock.Any(), 10, 0, 1, gomock.Any()).
					Return(expected, nil)
			},
			wantResult: expected,
		},
		{
			name:   "repo error is propagated",
			size:   10,
			start:  0,
			search: &models.SearchParams{},
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().GetEncountersList(gomock.Any(), 10, 0, 1).
					Return(nil, repoErr)
			},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockEncounterRepository(ctrl)
			tt.setup(repo)

			uc := NewEncounterUsecases(repo)
			result, err := uc.GetEncountersList(context.Background(), tt.size, tt.start, 1, tt.search)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

func TestSaveEncounter(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("save failure")

	tests := []struct {
		name      string
		encounter *models.SaveEncounterReq
		setup     func(repo *mocks.MockEncounterRepository)
		wantErr   error
	}{
		{
			name:      "empty name returns InvalidInputError",
			encounter: &models.SaveEncounterReq{Name: ""},
			setup:     func(_ *mocks.MockEncounterRepository) {},
			wantErr:   apperrors.InvalidInputError,
		},
		{
			name:      "name too long returns InvalidInputError",
			encounter: &models.SaveEncounterReq{Name: strings.Repeat("a", 61)},
			setup:     func(_ *mocks.MockEncounterRepository) {},
			wantErr:   apperrors.InvalidInputError,
		},
		{
			name:      "happy path calls repo with non-empty UUID",
			encounter: &models.SaveEncounterReq{Name: "Battle"},
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().SaveEncounter(gomock.Any(), gomock.Any(), gomock.Not(""), 1).
					Return(nil)
			},
		},
		{
			name:      "repo error is propagated",
			encounter: &models.SaveEncounterReq{Name: "Battle"},
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().SaveEncounter(gomock.Any(), gomock.Any(), gomock.Not(""), 1).
					Return(repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockEncounterRepository(ctrl)
			tt.setup(repo)

			uc := NewEncounterUsecases(repo)
			err := uc.SaveEncounter(context.Background(), tt.encounter, 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetEncounterByID(t *testing.T) {
	t.Parallel()

	expectedEncounter := &models.Encounter{Name: "Dungeon Boss"}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockEncounterRepository)
		wantErr error
		wantNil bool
	}{
		{
			name: "no permission returns PermissionDeniedError",
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "enc-1", 1).Return(false)
			},
			wantErr: apperrors.PermissionDeniedError,
			wantNil: true,
		},
		{
			name: "happy path returns encounter",
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "enc-1", 1).Return(true)
				repo.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").Return(expectedEncounter, nil)
			},
		},
		{
			name: "repo error is propagated",
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "enc-1", 1).Return(true)
				repo.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").Return(nil, repoErr)
			},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockEncounterRepository(ctrl)
			tt.setup(repo)

			uc := NewEncounterUsecases(repo)
			result, err := uc.GetEncounterByID(context.Background(), "enc-1", 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, expectedEncounter, result)
			}
		})
	}
}

func TestUpdateEncounter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockEncounterRepository)
		wantErr error
	}{
		{
			name: "no permission returns PermissionDeniedError",
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "enc-1", 1).Return(false)
			},
			wantErr: apperrors.PermissionDeniedError,
		},
		{
			name: "happy path delegates to repo",
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "enc-1", 1).Return(true)
				repo.EXPECT().UpdateEncounter(gomock.Any(), []byte(`{}`), "enc-1").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockEncounterRepository(ctrl)
			tt.setup(repo)

			uc := NewEncounterUsecases(repo)
			err := uc.UpdateEncounter(context.Background(), []byte(`{}`), "enc-1", 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemoveEncounter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockEncounterRepository)
		wantErr error
	}{
		{
			name: "no permission returns PermissionDeniedError",
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "enc-1", 1).Return(false)
			},
			wantErr: apperrors.PermissionDeniedError,
		},
		{
			name: "happy path delegates to repo",
			setup: func(repo *mocks.MockEncounterRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "enc-1", 1).Return(true)
				repo.EXPECT().RemoveEncounter(gomock.Any(), "enc-1").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockEncounterRepository(ctrl)
			tt.setup(repo)

			uc := NewEncounterUsecases(repo)
			err := uc.RemoveEncounter(context.Background(), "enc-1", 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
