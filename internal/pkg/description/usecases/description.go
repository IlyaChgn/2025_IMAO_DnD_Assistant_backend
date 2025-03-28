package usecases

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	descriptioninterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	descriptionproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery/protobuf"
)

type descriptionUsecase struct {
	descriptionClient descriptionproto.DescriptionServiceClient
}

func NewDescriptionUsecase(
	descriptionClient descriptionproto.DescriptionServiceClient) descriptioninterfaces.DescriptionUsecases {
	return &descriptionUsecase{
		descriptionClient: descriptionClient,
	}
}

func (uc *descriptionUsecase) GenerateDescription(ctx context.Context,
	req models.DescriptionGenerationRequest) (models.DescriptionGenerationResponse, error) {
	descriptionRequest := descriptionproto.DescriptionRequest{
		FirstCharId:  req.FirstCharID,
		SecondCharId: req.SecondCharID,
	}

	descriptionResponse, err := uc.descriptionClient.GenerateDescription(ctx, &descriptionRequest)
	if err != nil {
		return models.DescriptionGenerationResponse{}, apperrors.ReceivedDescriptionError
	}

	return models.DescriptionGenerationResponse{
		BattleDescription: descriptionResponse.BattleDescription,
	}, nil
}
