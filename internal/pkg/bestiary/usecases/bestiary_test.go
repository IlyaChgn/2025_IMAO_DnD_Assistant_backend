package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetCreaturesList(t *testing.T) {
	t.Parallel()

	expected := []*models.BestiaryCreature{{}}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		start   int
		size    int
		setup   func(repo *mocks.MockBestiaryRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			start:   -1,
			size:    10,
			setup:   func(_ *mocks.MockBestiaryRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			start:   0,
			size:    0,
			setup:   func(_ *mocks.MockBestiaryRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:  "happy path returns list",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreaturesList(gomock.Any(), 10, 0, gomock.Any(),
					models.FilterParams{}, models.SearchParams{}).Return(expected, nil)
			},
		},
		{
			name:  "repo error is propagated",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreaturesList(gomock.Any(), 10, 0, gomock.Any(),
					models.FilterParams{}, models.SearchParams{}).Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name:  "NoDocsErr is propagated",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreaturesList(gomock.Any(), 10, 0, gomock.Any(),
					models.FilterParams{}, models.SearchParams{}).Return(nil, apperrors.NoDocsErr)
			},
			wantErr: apperrors.NoDocsErr,
		},
		{
			name:  "UnknownDirectionError is propagated",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreaturesList(gomock.Any(), 10, 0, gomock.Any(),
					models.FilterParams{}, models.SearchParams{}).Return(nil, apperrors.UnknownDirectionError)
			},
			wantErr: apperrors.UnknownDirectionError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockBestiaryRepository(ctrl)
			s3 := mocks.NewMockBestiaryS3Manager(ctrl)
			gemini := mocks.NewMockGeminiAPI(ctrl)
			tt.setup(repo)

			uc := NewBestiaryUsecases(repo, s3, gemini)
			result, err := uc.GetCreaturesList(context.Background(), tt.size, tt.start,
				nil, models.FilterParams{}, models.SearchParams{})

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

func TestGetCreatureByEngName(t *testing.T) {
	t.Parallel()

	expected := &models.Creature{UserID: "public"}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockBestiaryRepository)
		wantErr error
		wantNil bool
	}{
		{
			name: "happy path returns creature",
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreatureByEngName(gomock.Any(), "goblin", false).Return(expected, nil)
			},
		},
		{
			name: "repo error is propagated",
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreatureByEngName(gomock.Any(), "goblin", false).Return(nil, repoErr)
			},
			wantErr: repoErr,
			wantNil: true,
		},
		{
			name: "nil creature returned without error",
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreatureByEngName(gomock.Any(), "goblin", false).Return(nil, nil)
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockBestiaryRepository(ctrl)
			s3 := mocks.NewMockBestiaryS3Manager(ctrl)
			gemini := mocks.NewMockGeminiAPI(ctrl)
			tt.setup(repo)

			uc := NewBestiaryUsecases(repo, s3, gemini)
			result, err := uc.GetCreatureByEngName(context.Background(), "goblin")

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
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

func TestGetUserCreaturesList(t *testing.T) {
	t.Parallel()

	expected := []*models.BestiaryCreature{{}}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		start   int
		size    int
		setup   func(repo *mocks.MockBestiaryRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			start:   -1,
			size:    10,
			setup:   func(_ *mocks.MockBestiaryRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			start:   0,
			size:    0,
			setup:   func(_ *mocks.MockBestiaryRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:  "happy path returns list",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetUserCreaturesList(gomock.Any(), 10, 0, gomock.Any(),
					models.FilterParams{}, models.SearchParams{}, 1).Return(expected, nil)
			},
		},
		{
			name:  "repo error is propagated",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetUserCreaturesList(gomock.Any(), 10, 0, gomock.Any(),
					models.FilterParams{}, models.SearchParams{}, 1).Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name:  "NoDocsErr is propagated",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetUserCreaturesList(gomock.Any(), 10, 0, gomock.Any(),
					models.FilterParams{}, models.SearchParams{}, 1).Return(nil, apperrors.NoDocsErr)
			},
			wantErr: apperrors.NoDocsErr,
		},
		{
			name:  "UnknownDirectionError is propagated",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetUserCreaturesList(gomock.Any(), 10, 0, gomock.Any(),
					models.FilterParams{}, models.SearchParams{}, 1).Return(nil, apperrors.UnknownDirectionError)
			},
			wantErr: apperrors.UnknownDirectionError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockBestiaryRepository(ctrl)
			s3 := mocks.NewMockBestiaryS3Manager(ctrl)
			gemini := mocks.NewMockGeminiAPI(ctrl)
			tt.setup(repo)

			uc := NewBestiaryUsecases(repo, s3, gemini)
			result, err := uc.GetUserCreaturesList(context.Background(), tt.size, tt.start,
				nil, models.FilterParams{}, models.SearchParams{}, 1)

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

func TestGetUserCreatureByEngName(t *testing.T) {
	t.Parallel()

	ownedCreature := &models.Creature{UserID: "1"}
	otherCreature := &models.Creature{UserID: "999"}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		userID  int
		setup   func(repo *mocks.MockBestiaryRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:   "happy path returns own creature",
			userID: 1,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreatureByEngName(gomock.Any(), "goblin", true).Return(ownedCreature, nil)
			},
		},
		{
			name:   "other user's creature returns PermissionDeniedError",
			userID: 1,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreatureByEngName(gomock.Any(), "goblin", true).Return(otherCreature, nil)
			},
			wantErr: apperrors.PermissionDeniedError,
			wantNil: true,
		},
		{
			name:   "repo error is propagated",
			userID: 1,
			setup: func(repo *mocks.MockBestiaryRepository) {
				repo.EXPECT().GetCreatureByEngName(gomock.Any(), "goblin", true).Return(nil, repoErr)
			},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockBestiaryRepository(ctrl)
			s3 := mocks.NewMockBestiaryS3Manager(ctrl)
			gemini := mocks.NewMockGeminiAPI(ctrl)
			tt.setup(repo)

			uc := NewBestiaryUsecases(repo, s3, gemini)
			result, err := uc.GetUserCreatureByEngName(context.Background(), "goblin", tt.userID)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
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
