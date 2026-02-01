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

type fakeBestiaryRepo struct {
	listResult     []*models.BestiaryCreature
	listErr        error
	creatureResult *models.Creature
	creatureErr    error
	userListResult []*models.BestiaryCreature
	userListErr    error
	addErr         error
}

func (f *fakeBestiaryRepo) GetCreaturesList(_ context.Context, _, _ int, _ []models.Order,
	_ models.FilterParams, _ models.SearchParams) ([]*models.BestiaryCreature, error) {
	return f.listResult, f.listErr
}

func (f *fakeBestiaryRepo) GetCreatureByEngName(_ context.Context, _ string, _ bool) (*models.Creature, error) {
	return f.creatureResult, f.creatureErr
}

func (f *fakeBestiaryRepo) GetUserCreaturesList(_ context.Context, _, _ int, _ []models.Order,
	_ models.FilterParams, _ models.SearchParams, _ int) ([]*models.BestiaryCreature, error) {
	return f.userListResult, f.userListErr
}

func (f *fakeBestiaryRepo) AddGeneratedCreature(_ context.Context, _ models.Creature) error {
	return f.addErr
}

// --- fake S3 & Gemini (unused by read methods, satisfy constructor) ---

type fakeS3 struct{}

func (f *fakeS3) UploadImage(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

type fakeGemini struct{}

func (f *fakeGemini) GenerateFromImage(_ context.Context, _ []byte) (map[string]interface{}, error) {
	return nil, nil
}

func (f *fakeGemini) GenerateFromDescription(_ context.Context, _ string) (map[string]interface{}, error) {
	return nil, nil
}

// --- tests ---

func TestGetCreaturesList(t *testing.T) {
	t.Parallel()

	expected := []*models.BestiaryCreature{{}}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		start   int
		size    int
		repo    *fakeBestiaryRepo
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			start:   -1,
			size:    10,
			repo:    &fakeBestiaryRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			start:   0,
			size:    0,
			repo:    &fakeBestiaryRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:  "happy path returns list",
			start: 0,
			size:  10,
			repo:  &fakeBestiaryRepo{listResult: expected},
		},
		{
			name:    "repo error is propagated",
			start:   0,
			size:    10,
			repo:    &fakeBestiaryRepo{listErr: repoErr},
			wantErr: repoErr,
		},
		{
			name:    "NoDocsErr is propagated",
			start:   0,
			size:    10,
			repo:    &fakeBestiaryRepo{listErr: apperrors.NoDocsErr},
			wantErr: apperrors.NoDocsErr,
		},
		{
			name:    "UnknownDirectionError is propagated",
			start:   0,
			size:    10,
			repo:    &fakeBestiaryRepo{listErr: apperrors.UnknownDirectionError},
			wantErr: apperrors.UnknownDirectionError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewBestiaryUsecases(tt.repo, &fakeS3{}, &fakeGemini{})
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
		repo    *fakeBestiaryRepo
		wantErr error
		wantNil bool
	}{
		{
			name: "happy path returns creature",
			repo: &fakeBestiaryRepo{creatureResult: expected},
		},
		{
			name:    "repo error is propagated",
			repo:    &fakeBestiaryRepo{creatureErr: repoErr},
			wantErr: repoErr,
			wantNil: true,
		},
		{
			name:    "nil creature returned without error",
			repo:    &fakeBestiaryRepo{creatureResult: nil},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewBestiaryUsecases(tt.repo, &fakeS3{}, &fakeGemini{})
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
		repo    *fakeBestiaryRepo
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			start:   -1,
			size:    10,
			repo:    &fakeBestiaryRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			start:   0,
			size:    0,
			repo:    &fakeBestiaryRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:  "happy path returns list",
			start: 0,
			size:  10,
			repo:  &fakeBestiaryRepo{userListResult: expected},
		},
		{
			name:    "repo error is propagated",
			start:   0,
			size:    10,
			repo:    &fakeBestiaryRepo{userListErr: repoErr},
			wantErr: repoErr,
		},
		{
			name:    "NoDocsErr is propagated",
			start:   0,
			size:    10,
			repo:    &fakeBestiaryRepo{userListErr: apperrors.NoDocsErr},
			wantErr: apperrors.NoDocsErr,
		},
		{
			name:    "UnknownDirectionError is propagated",
			start:   0,
			size:    10,
			repo:    &fakeBestiaryRepo{userListErr: apperrors.UnknownDirectionError},
			wantErr: apperrors.UnknownDirectionError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewBestiaryUsecases(tt.repo, &fakeS3{}, &fakeGemini{})
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
		repo    *fakeBestiaryRepo
		wantErr error
		wantNil bool
	}{
		{
			name:   "happy path returns own creature",
			userID: 1,
			repo:   &fakeBestiaryRepo{creatureResult: ownedCreature},
		},
		{
			name:    "other user's creature returns PermissionDeniedError",
			userID:  1,
			repo:    &fakeBestiaryRepo{creatureResult: otherCreature},
			wantErr: apperrors.PermissionDeniedError,
			wantNil: true,
		},
		{
			name:    "repo error is propagated",
			userID:  1,
			repo:    &fakeBestiaryRepo{creatureErr: repoErr},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewBestiaryUsecases(tt.repo, &fakeS3{}, &fakeGemini{})
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
