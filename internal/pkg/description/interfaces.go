package description

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_description.go -package=mocks

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type DescriptionUsecases interface {
	GenerateDescription(ctx context.Context,
		req models.DescriptionGenerationRequest) (models.DescriptionGenerationResponse, error)
}

type DescriptionGateway interface {
	Describe(ctx context.Context, firstCharID, secondCharID string) (string, error)
}
