package description

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type DescriptionUsecases interface {
	GenerateDescription(ctx context.Context,
		req models.DescriptionGenerationRequest) (models.DescriptionGenerationResponse, error)
}
