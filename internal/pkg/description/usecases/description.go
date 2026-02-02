package usecases

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	descriptioninterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type descriptionUsecase struct {
	gateway descriptioninterfaces.DescriptionGateway
}

func NewDescriptionUsecase(gateway descriptioninterfaces.DescriptionGateway) descriptioninterfaces.DescriptionUsecases {
	return &descriptionUsecase{
		gateway: gateway,
	}
}

func (uc *descriptionUsecase) GenerateDescription(ctx context.Context,
	req models.DescriptionGenerationRequest) (models.DescriptionGenerationResponse, error) {
	l := logger.FromContext(ctx)

	battleDescription, err := uc.gateway.Describe(ctx, req.FirstCharID, req.SecondCharID)
	if err != nil {
		l.UsecasesError(err, 0, nil)
		return models.DescriptionGenerationResponse{}, apperrors.ReceivedDescriptionError
	}

	return models.DescriptionGenerationResponse{
		BattleDescription: battleDescription,
	}, nil
}
