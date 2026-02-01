package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	categories []*models.MapTileCategory
	err        error
}

func (m *mockRepo) GetCategories(_ context.Context, _ int) ([]*models.MapTileCategory, error) {
	return m.categories, m.err
}

func TestGetCategories_NegativeUserID(t *testing.T) {
	uc := NewMapTilesUsecases(&mockRepo{})

	result, err := uc.GetCategories(context.Background(), -1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, apperrors.InvalidUserIDError))
}

func TestGetCategories_ValidUserID(t *testing.T) {
	expected := []*models.MapTileCategory{
		{Name: "terrain"},
	}

	uc := NewMapTilesUsecases(&mockRepo{categories: expected})

	result, err := uc.GetCategories(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}
