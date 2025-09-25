package usecases

import (
	"context"
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	actionproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery/protobuf"
	//"google.golang.org/protobuf/types/known/structpb"
)

type actionProcessorUsecase struct {
	actionClient actionproto.ActionProcessorServiceClient
}

func NewActionProcessorUsecase(
	client actionproto.ActionProcessorServiceClient,
) bestiaryinterface.ActionProcessorUsecases {
	return &actionProcessorUsecase{
		actionClient: client,
	}
}

func (uc *actionProcessorUsecase) ProcessActions(ctx context.Context,
	actions []models.Action) ([]models.AttackLLM, error) {
	l := logger.FromContext(ctx)

	// Преобразуем []models.Action в *actionproto.ActionList
	var protoActions []*actionproto.Action
	for _, a := range actions {
		protoActions = append(protoActions, &actionproto.Action{
			Name:  a.Name,
			Value: a.Value,
		})
	}

	request := &actionproto.ActionList{
		Actions: protoActions,
	}

	response, err := uc.actionClient.ProcessActions(ctx, request)
	if err != nil {
		l.UsecasesError(err, 0, nil)
		return nil, apperrors.ReceivedActionProcessingError
	}

	// Преобразуем protobuf Struct -> map[string]interface{}
	rawMap := response.AsMap()

	// Извлекаем parsed_actions
	rawParsed, ok := rawMap["parsed_actions"]
	if !ok {
		l.UsecasesError(apperrors.ParsedActionsErr, 0, nil)
		return nil, apperrors.ParsedActionsErr
	}

	// Преобразуем в JSON → затем в []AttackLLM
	jsonBytes, err := json.Marshal(rawParsed)
	if err != nil {
		l.UsecasesError(err, 0, nil)
		return nil, err
	}

	var attacks []models.AttackLLM
	if err := json.Unmarshal(jsonBytes, &attacks); err != nil {
		l.UsecasesError(err, 0, nil)
		return nil, err
	}

	return attacks, nil
}
