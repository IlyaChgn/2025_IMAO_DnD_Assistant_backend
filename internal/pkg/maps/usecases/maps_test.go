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

type fakeMapsRepo struct {
	hasPermission bool
	createResult  *models.MapFull
	createErr     error
	getResult     *models.MapFull
	getErr        error
	updateResult  *models.MapFull
	updateErr     error
	deleteErr     error
	listResult    *models.MapsList
	listErr       error

	createCalled bool
	getCalled    bool
	updateCalled bool
	deleteCalled bool
	listCalled   bool
}

func (f *fakeMapsRepo) CheckPermission(_ context.Context, _ string, _ int) bool {
	return f.hasPermission
}

func (f *fakeMapsRepo) CreateMap(_ context.Context, _ int, _ string, _ []byte) (*models.MapFull, error) {
	f.createCalled = true
	return f.createResult, f.createErr
}

func (f *fakeMapsRepo) GetMapByID(_ context.Context, _ int, _ string) (*models.MapFull, error) {
	f.getCalled = true
	return f.getResult, f.getErr
}

func (f *fakeMapsRepo) UpdateMap(_ context.Context, _ int, _ string, _ string, _ []byte) (*models.MapFull, error) {
	f.updateCalled = true
	return f.updateResult, f.updateErr
}

func (f *fakeMapsRepo) DeleteMap(_ context.Context, _ int, _ string) error {
	f.deleteCalled = true
	return f.deleteErr
}

func (f *fakeMapsRepo) ListMaps(_ context.Context, _ int, _, _ int) (*models.MapsList, error) {
	f.listCalled = true
	return f.listResult, f.listErr
}

// --- helpers ---

func validMapData() models.MapData {
	return models.MapData{
		SchemaVersion: 1,
		WidthUnits:    12,
		HeightUnits:   12,
		Placements:    []models.Placement{},
	}
}

// --- tests ---

func TestListMaps(t *testing.T) {
	t.Parallel()

	expected := &models.MapsList{Maps: []models.MapMetadata{}, Total: 0}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		start   int
		size    int
		repo    *fakeMapsRepo
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			start:   -1,
			size:    10,
			repo:    &fakeMapsRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			start:   0,
			size:    0,
			repo:    &fakeMapsRepo{},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:  "happy path returns list",
			start: 0,
			size:  10,
			repo:  &fakeMapsRepo{listResult: expected},
		},
		{
			name:    "repo error is propagated",
			start:   0,
			size:    10,
			repo:    &fakeMapsRepo{listErr: repoErr},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewMapsUsecases(tt.repo)
			result, err := uc.ListMaps(context.Background(), 1, tt.start, tt.size)

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

func TestCreateMap(t *testing.T) {
	t.Parallel()

	expectedMap := &models.MapFull{ID: "uuid-1", Name: "Battle Map"}
	repoErr := errors.New("insert failure")

	tests := []struct {
		name       string
		req        *models.CreateMapRequest
		repo       *fakeMapsRepo
		wantCreate bool
		wantValErr bool // expect ValidationErrorWrapper
		wantErr    error
	}{
		{
			name: "validation fails on empty name",
			req: &models.CreateMapRequest{
				Name: "",
				Data: validMapData(),
			},
			repo:       &fakeMapsRepo{},
			wantValErr: true,
		},
		{
			name: "validation fails on invalid dimensions",
			req: &models.CreateMapRequest{
				Name: "Test",
				Data: models.MapData{
					SchemaVersion: 1,
					WidthUnits:    7, // not a multiple of 6
					HeightUnits:   12,
				},
			},
			repo:       &fakeMapsRepo{},
			wantValErr: true,
		},
		{
			name: "happy path calls repo and returns result",
			req: &models.CreateMapRequest{
				Name: "Battle Map",
				Data: validMapData(),
			},
			repo:       &fakeMapsRepo{createResult: expectedMap},
			wantCreate: true,
		},
		{
			name: "repo error is propagated",
			req: &models.CreateMapRequest{
				Name: "Battle Map",
				Data: validMapData(),
			},
			repo:       &fakeMapsRepo{createErr: repoErr},
			wantCreate: true,
			wantErr:    repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewMapsUsecases(tt.repo)
			result, err := uc.CreateMap(context.Background(), 1, tt.req)

			if tt.wantValErr {
				var valErr *ValidationErrorWrapper
				assert.True(t, errors.As(err, &valErr), "expected ValidationErrorWrapper, got %T: %v", err, err)
				assert.NotEmpty(t, valErr.Errors)
				assert.Nil(t, result)
			} else if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedMap, result)
			}

			assert.Equal(t, tt.wantCreate, tt.repo.createCalled)
		})
	}
}

func TestGetMapByID(t *testing.T) {
	t.Parallel()

	expectedMap := &models.MapFull{ID: "uuid-1", Name: "Dungeon"}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		repo    *fakeMapsRepo
		wantErr error
		wantNil bool
	}{
		{
			name:    "no permission returns MapPermissionDenied",
			repo:    &fakeMapsRepo{hasPermission: false},
			wantErr: apperrors.MapPermissionDenied,
			wantNil: true,
		},
		{
			name: "happy path returns map",
			repo: &fakeMapsRepo{hasPermission: true, getResult: expectedMap},
		},
		{
			name:    "repo error is propagated",
			repo:    &fakeMapsRepo{hasPermission: true, getErr: repoErr},
			wantErr: repoErr,
			wantNil: true,
		},
		{
			name:    "map not found is propagated",
			repo:    &fakeMapsRepo{hasPermission: true, getErr: apperrors.MapNotFoundError},
			wantErr: apperrors.MapNotFoundError,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewMapsUsecases(tt.repo)
			result, err := uc.GetMapByID(context.Background(), 1, "map-id")

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else if tt.wantErr == nil {
				assert.Equal(t, expectedMap, result)
			}
		})
	}
}

func TestUpdateMap(t *testing.T) {
	t.Parallel()

	expectedMap := &models.MapFull{ID: "uuid-1", Name: "Updated"}
	repoErr := errors.New("db failure")

	tests := []struct {
		name       string
		req        *models.UpdateMapRequest
		repo       *fakeMapsRepo
		wantUpdate bool
		wantValErr bool
		wantErr    error
	}{
		{
			name: "no permission returns MapPermissionDenied",
			req: &models.UpdateMapRequest{
				Name: "Updated",
				Data: validMapData(),
			},
			repo:    &fakeMapsRepo{hasPermission: false},
			wantErr: apperrors.MapPermissionDenied,
		},
		{
			name: "validation fails after permission check",
			req: &models.UpdateMapRequest{
				Name: "",
				Data: validMapData(),
			},
			repo:       &fakeMapsRepo{hasPermission: true},
			wantValErr: true,
		},
		{
			name: "happy path calls repo",
			req: &models.UpdateMapRequest{
				Name: "Updated",
				Data: validMapData(),
			},
			repo:       &fakeMapsRepo{hasPermission: true, updateResult: expectedMap},
			wantUpdate: true,
		},
		{
			name: "repo error is propagated",
			req: &models.UpdateMapRequest{
				Name: "Updated",
				Data: validMapData(),
			},
			repo:       &fakeMapsRepo{hasPermission: true, updateErr: repoErr},
			wantUpdate: true,
			wantErr:    repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewMapsUsecases(tt.repo)
			result, err := uc.UpdateMap(context.Background(), 1, "map-id", tt.req)

			if tt.wantValErr {
				var valErr *ValidationErrorWrapper
				assert.True(t, errors.As(err, &valErr), "expected ValidationErrorWrapper, got %T: %v", err, err)
				assert.Nil(t, result)
			} else if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedMap, result)
			}

			assert.Equal(t, tt.wantUpdate, tt.repo.updateCalled)
		})
	}
}

func TestDeleteMap(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db failure")

	tests := []struct {
		name       string
		repo       *fakeMapsRepo
		wantDelete bool
		wantErr    error
	}{
		{
			name:    "no permission returns MapPermissionDenied",
			repo:    &fakeMapsRepo{hasPermission: false},
			wantErr: apperrors.MapPermissionDenied,
		},
		{
			name:       "happy path calls repo",
			repo:       &fakeMapsRepo{hasPermission: true},
			wantDelete: true,
		},
		{
			name:       "repo error is propagated",
			repo:       &fakeMapsRepo{hasPermission: true, deleteErr: repoErr},
			wantDelete: true,
			wantErr:    repoErr,
		},
		{
			name:       "map not found is propagated",
			repo:       &fakeMapsRepo{hasPermission: true, deleteErr: apperrors.MapNotFoundError},
			wantDelete: true,
			wantErr:    apperrors.MapNotFoundError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewMapsUsecases(tt.repo)
			err := uc.DeleteMap(context.Background(), 1, "map-id")

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantDelete, tt.repo.deleteCalled)
		})
	}
}
