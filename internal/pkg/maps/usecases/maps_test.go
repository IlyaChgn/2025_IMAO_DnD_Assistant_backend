package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

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
		setup   func(repo *mocks.MockMapsRepository)
		wantErr error
		wantNil bool
	}{
		{
			name:    "negative start returns StartPosSizeError",
			start:   -1,
			size:    10,
			setup:   func(_ *mocks.MockMapsRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:    "zero size returns StartPosSizeError",
			start:   0,
			size:    0,
			setup:   func(_ *mocks.MockMapsRepository) {},
			wantErr: apperrors.StartPosSizeError,
			wantNil: true,
		},
		{
			name:  "happy path returns list",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().ListMaps(gomock.Any(), 1, 0, 10).Return(expected, nil)
			},
		},
		{
			name:  "repo error is propagated",
			start: 0,
			size:  10,
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().ListMaps(gomock.Any(), 1, 0, 10).Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMapsRepository(ctrl)
			tt.setup(repo)

			uc := NewMapsUsecases(repo)
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
		setup      func(repo *mocks.MockMapsRepository)
		wantValErr bool
		wantErr    error
	}{
		{
			name: "validation fails on empty name",
			req: &models.CreateMapRequest{
				Name: "",
				Data: validMapData(),
			},
			setup:      func(_ *mocks.MockMapsRepository) {},
			wantValErr: true,
		},
		{
			name: "validation fails on invalid dimensions",
			req: &models.CreateMapRequest{
				Name: "Test",
				Data: models.MapData{
					SchemaVersion: 1,
					WidthUnits:    7,
					HeightUnits:   12,
				},
			},
			setup:      func(_ *mocks.MockMapsRepository) {},
			wantValErr: true,
		},
		{
			name: "happy path calls repo and returns result",
			req: &models.CreateMapRequest{
				Name: "Battle Map",
				Data: validMapData(),
			},
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CreateMap(gomock.Any(), 1, "Battle Map", gomock.Any()).
					Return(expectedMap, nil)
			},
		},
		{
			name: "repo error is propagated",
			req: &models.CreateMapRequest{
				Name: "Battle Map",
				Data: validMapData(),
			},
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CreateMap(gomock.Any(), 1, "Battle Map", gomock.Any()).
					Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMapsRepository(ctrl)
			tt.setup(repo)

			uc := NewMapsUsecases(repo)
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
		})
	}
}

func TestGetMapByID(t *testing.T) {
	t.Parallel()

	expectedMap := &models.MapFull{ID: "uuid-1", Name: "Dungeon"}
	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockMapsRepository)
		wantErr error
		wantNil bool
	}{
		{
			name: "no permission returns MapPermissionDenied",
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(false)
			},
			wantErr: apperrors.MapPermissionDenied,
			wantNil: true,
		},
		{
			name: "happy path returns map",
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
				repo.EXPECT().GetMapByID(gomock.Any(), 1, "map-id").Return(expectedMap, nil)
			},
		},
		{
			name: "repo error is propagated",
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
				repo.EXPECT().GetMapByID(gomock.Any(), 1, "map-id").Return(nil, repoErr)
			},
			wantErr: repoErr,
			wantNil: true,
		},
		{
			name: "map not found is propagated",
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
				repo.EXPECT().GetMapByID(gomock.Any(), 1, "map-id").Return(nil, apperrors.MapNotFoundError)
			},
			wantErr: apperrors.MapNotFoundError,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMapsRepository(ctrl)
			tt.setup(repo)

			uc := NewMapsUsecases(repo)
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
		setup      func(repo *mocks.MockMapsRepository)
		wantValErr bool
		wantErr    error
	}{
		{
			name: "no permission returns MapPermissionDenied",
			req: &models.UpdateMapRequest{
				Name: "Updated",
				Data: validMapData(),
			},
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(false)
			},
			wantErr: apperrors.MapPermissionDenied,
		},
		{
			name: "validation fails after permission check",
			req: &models.UpdateMapRequest{
				Name: "",
				Data: validMapData(),
			},
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
			},
			wantValErr: true,
		},
		{
			name: "happy path calls repo",
			req: &models.UpdateMapRequest{
				Name: "Updated",
				Data: validMapData(),
			},
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
				repo.EXPECT().UpdateMap(gomock.Any(), 1, "map-id", "Updated", gomock.Any()).
					Return(expectedMap, nil)
			},
		},
		{
			name: "repo error is propagated",
			req: &models.UpdateMapRequest{
				Name: "Updated",
				Data: validMapData(),
			},
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
				repo.EXPECT().UpdateMap(gomock.Any(), 1, "map-id", "Updated", gomock.Any()).
					Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMapsRepository(ctrl)
			tt.setup(repo)

			uc := NewMapsUsecases(repo)
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
		})
	}
}

func TestDeleteMap(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		setup   func(repo *mocks.MockMapsRepository)
		wantErr error
	}{
		{
			name: "no permission returns MapPermissionDenied",
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(false)
			},
			wantErr: apperrors.MapPermissionDenied,
		},
		{
			name: "happy path calls repo",
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
				repo.EXPECT().DeleteMap(gomock.Any(), 1, "map-id").Return(nil)
			},
		},
		{
			name: "repo error is propagated",
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
				repo.EXPECT().DeleteMap(gomock.Any(), 1, "map-id").Return(repoErr)
			},
			wantErr: repoErr,
		},
		{
			name: "map not found is propagated",
			setup: func(repo *mocks.MockMapsRepository) {
				repo.EXPECT().CheckPermission(gomock.Any(), "map-id", 1).Return(true)
				repo.EXPECT().DeleteMap(gomock.Any(), 1, "map-id").Return(apperrors.MapNotFoundError)
			},
			wantErr: apperrors.MapNotFoundError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mocks.NewMockMapsRepository(ctrl)
			tt.setup(repo)

			uc := NewMapsUsecases(repo)
			err := uc.DeleteMap(context.Background(), 1, "map-id")

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
