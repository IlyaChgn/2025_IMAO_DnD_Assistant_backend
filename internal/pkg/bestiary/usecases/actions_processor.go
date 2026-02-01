package usecases

import (
	"context"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type actionProcessorUsecase struct {
	gateway bestiaryinterface.ActionProcessorGateway
}

func NewActionProcessorUsecase(
	gateway bestiaryinterface.ActionProcessorGateway,
) bestiaryinterface.ActionProcessorUsecases {
	return &actionProcessorUsecase{
		gateway: gateway,
	}
}

func (uc *actionProcessorUsecase) ProcessActions(ctx context.Context,
	actions []models.Action) ([]models.AttackLLM, error) {
	l := logger.FromContext(ctx)

	rawMap, err := uc.gateway.ProcessActions(ctx, actions)
	if err != nil {
		l.UsecasesError(err, 0, nil)
		return nil, apperrors.ReceivedActionProcessingError
	}

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
