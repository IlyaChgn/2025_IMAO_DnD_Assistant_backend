package server

import (
	"context"
	"errors"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

var errServiceUnavailable = errors.New("service unavailable in test mode")

// stubDescriptionGateway implements description.DescriptionGateway.
type stubDescriptionGateway struct{}

func (stubDescriptionGateway) Describe(_ context.Context, _, _ string) (string, error) {
	return "", errServiceUnavailable
}

// stubActionProcessorGateway implements bestiary.ActionProcessorGateway.
type stubActionProcessorGateway struct{}

func (stubActionProcessorGateway) ProcessActions(_ context.Context, _ []models.Action) (map[string]interface{}, error) {
	return nil, errServiceUnavailable
}

// stubGeminiAPI implements bestiary.GeminiAPI.
type stubGeminiAPI struct{}

func (stubGeminiAPI) GenerateFromImage(_ context.Context, _ []byte) (map[string]interface{}, error) {
	return nil, errServiceUnavailable
}

func (stubGeminiAPI) GenerateFromDescription(_ context.Context, _ string) (map[string]interface{}, error) {
	return nil, errServiceUnavailable
}
