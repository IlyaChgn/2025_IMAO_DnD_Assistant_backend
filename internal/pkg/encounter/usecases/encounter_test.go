package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/stretchr/testify/assert"
)

// --- fake repository ---

type fakeEncounterRepo struct {
	// Return values
	listResult      *models.EncountersList
	listErr         error
	encounterResult *models.Encounter
	encounterErr    error
	saveErr         error
	updateErr       error
	removeErr       error
	hasPermission   bool

	// Captured calls
	saveCalled       bool
	saveID           string
	updateCalled     bool
	removeCalled     bool
	listSearchCalled bool
}

func (f *fakeEncounterRepo) GetEncountersList(_ context.Context, _, _, _ int) (*models.EncountersList, error) {
	return f.listResult, f.listErr
}

func (f *fakeEncounterRepo) GetEncountersListWithSearch(_ context.Context, _, _, _ int,
	_ *models.SearchParams) (*models.EncountersList, error) {
	f.listSearchCalled = true
	return f.listResult, f.listErr
}

func (f *fakeEncounterRepo) GetEncounterByID(_ context.Context, _ string) (*models.Encounter, error) {
	return f.encounterResult, f.encounterErr
}

func (f *fakeEncounterRepo) SaveEncounter(_ context.Context, _ *models.SaveEncounterReq, id string, _ int) error {
	f.saveCalled = true
	f.saveID = id
	return f.saveErr
}

func (f *fakeEncounterRepo) UpdateEncounter(_ context.Context, _ []byte, _ string) error {
	f.updateCalled = true
	return f.updateErr
}

func (f *fakeEncounterRepo) RemoveEncounter(_ context.Context, _ string) error {
	f.removeCalled = true
	return f.removeErr
}

func (f *fakeEncounterRepo) CheckPermission(_ context.Context, _ string, _ int) bool {
	return f.hasPermission
}

// --- tests ---

func TestGetEncountersList(t *testing.T) {
	t.Parallel()

	expected := &models.EncountersList{}
	repoErr := errors.New("db failure")

	tests := []struct {
		name       string
		size       int
		start      int
		search     *models.SearchParams
		repo       *fakeEncounterRepo
		wantErr    error
		wantNil    bool
		wantSearch bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			size:    10,
			start:   -1,
			search:  &models.SearchParams{},
			repo:    &fakeEncounterRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			size:    0,
			start:   0,
			search:  &models.SearchParams{},
			repo:    &fakeEncounterRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:   "happy path without search",
			size:   10,
			start:  0,
			search: &models.SearchParams{},
			repo:   &fakeEncounterRepo{listResult: expected},
		},
		{
			name:       "happy path with search delegates to search method",
			size:       10,
			start:      0,
			search:     &models.SearchParams{Value: "dragon"},
			repo:       &fakeEncounterRepo{listResult: expected},
			wantSearch: true,
		},
		{
			name:    "repo error is propagated",
			size:    10,
			start:   0,
			search:  &models.SearchParams{},
			repo:    &fakeEncounterRepo{listErr: repoErr},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewEncounterUsecases(tt.repo)
			result, err := uc.GetEncountersList(context.Background(), tt.size, tt.start, 1, tt.search)

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

			if tt.wantSearch {
				assert.True(t, tt.repo.listSearchCalled, "expected search method to be called")
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
		repo      *fakeEncounterRepo
		wantErr   error
		wantSave  bool
	}{
		{
			name:      "empty name returns InvalidInputError",
			encounter: &models.SaveEncounterReq{Name: ""},
			repo:      &fakeEncounterRepo{},
			wantErr:   apperrors.InvalidInputError,
		},
		{
			name:      "name too long returns InvalidInputError",
			encounter: &models.SaveEncounterReq{Name: strings.Repeat("a", 61)},
			repo:      &fakeEncounterRepo{},
			wantErr:   apperrors.InvalidInputError,
		},
		{
			name:      "happy path calls repo with non-empty UUID",
			encounter: &models.SaveEncounterReq{Name: "Battle"},
			repo:      &fakeEncounterRepo{},
			wantSave:  true,
		},
		{
			name:      "repo error is propagated",
			encounter: &models.SaveEncounterReq{Name: "Battle"},
			repo:      &fakeEncounterRepo{saveErr: repoErr},
			wantErr:   repoErr,
			wantSave:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewEncounterUsecases(tt.repo)
			err := uc.SaveEncounter(context.Background(), tt.encounter, 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantSave, tt.repo.saveCalled)

			if tt.wantSave && tt.wantErr == nil {
				assert.NotEmpty(t, tt.repo.saveID, "expected UUID to be generated")
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
		repo    *fakeEncounterRepo
		wantErr error
		wantNil bool
	}{
		{
			name:    "no permission returns PermissionDeniedError",
			repo:    &fakeEncounterRepo{hasPermission: false},
			wantErr: apperrors.PermissionDeniedError,
			wantNil: true,
		},
		{
			name: "happy path returns encounter",
			repo: &fakeEncounterRepo{hasPermission: true, encounterResult: expectedEncounter},
		},
		{
			name:    "repo error is propagated",
			repo:    &fakeEncounterRepo{hasPermission: true, encounterErr: repoErr},
			wantErr: repoErr,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewEncounterUsecases(tt.repo)
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
		repo    *fakeEncounterRepo
		wantErr error
	}{
		{
			name:    "no permission returns PermissionDeniedError",
			repo:    &fakeEncounterRepo{hasPermission: false},
			wantErr: apperrors.PermissionDeniedError,
		},
		{
			name: "happy path delegates to repo",
			repo: &fakeEncounterRepo{hasPermission: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewEncounterUsecases(tt.repo)
			err := uc.UpdateEncounter(context.Background(), []byte(`{}`), "enc-1", 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.repo.updateCalled)
			}
		})
	}
}

func TestRemoveEncounter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repo    *fakeEncounterRepo
		wantErr error
	}{
		{
			name:    "no permission returns PermissionDeniedError",
			repo:    &fakeEncounterRepo{hasPermission: false},
			wantErr: apperrors.PermissionDeniedError,
		},
		{
			name: "happy path delegates to repo",
			repo: &fakeEncounterRepo{hasPermission: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewEncounterUsecases(tt.repo)
			err := uc.RemoveEncounter(context.Background(), "enc-1", 1)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			} else {
				assert.NoError(t, err)
				assert.True(t, tt.repo.removeCalled)
			}
		})
	}
}
