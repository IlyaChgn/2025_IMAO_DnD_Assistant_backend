package delivery

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	actionproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery/protobuf"
)

type actionProcessorAdapter struct {
	client actionproto.ActionProcessorServiceClient
}

func NewActionProcessorAdapter(
	client actionproto.ActionProcessorServiceClient,
) bestiaryinterface.ActionProcessorGateway {
	return &actionProcessorAdapter{client: client}
}

func (a *actionProcessorAdapter) ProcessActions(ctx context.Context,
	actions []models.Action) (map[string]interface{}, error) {
	var protoActions []*actionproto.Action
	for _, act := range actions {
		protoActions = append(protoActions, &actionproto.Action{
			Name:  act.Name,
			Value: act.Value,
		})
	}

	request := &actionproto.ActionList{
		Actions: protoActions,
	}

	response, err := a.client.ProcessActions(ctx, request)
	if err != nil {
		return nil, err
	}

	return response.AsMap(), nil
}
