package delivery

import (
	"context"

	description "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	descriptionproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery/protobuf"
)

type descriptionGatewayAdapter struct {
	client descriptionproto.DescriptionServiceClient
}

func NewDescriptionGatewayAdapter(client descriptionproto.DescriptionServiceClient) description.DescriptionGateway {
	return &descriptionGatewayAdapter{client: client}
}

func (a *descriptionGatewayAdapter) Describe(ctx context.Context, firstCharID, secondCharID string) (string, error) {
	resp, err := a.client.GenerateDescription(ctx, &descriptionproto.DescriptionRequest{
		FirstCharId:  firstCharID,
		SecondCharId: secondCharID,
	})
	if err != nil {
		return "", err
	}

	return resp.BattleDescription, nil
}
