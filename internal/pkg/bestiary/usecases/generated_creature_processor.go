package usecases

import (
	"context"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"regexp"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
)

type GeneratedCreatureProcessor struct {
	actionProcessor bestiaryinterfaces.ActionProcessorUsecases
}

func NewGeneratedCreatureProcessor(
	actionProcessor bestiaryinterfaces.ActionProcessorUsecases,
) bestiaryinterfaces.GeneratedCreatureProcessorUsecases {
	return &GeneratedCreatureProcessor{
		actionProcessor: actionProcessor,
	}
}

func (processor *GeneratedCreatureProcessor) ValidateAndProcessGeneratedCreature(
	ctx context.Context, c *models.Creature,
) (*models.Creature, error) {
	l := logger.FromContext(ctx)

	if c == nil {
		l.UsecasesError(apperrors.NilCreatureErr, 0, nil)
		return nil, apperrors.NilCreatureErr
	}

	updated := *c

	// 🧠 Попробуем получить LLM-атаки
	attacksLLM, err := processor.actionProcessor.ProcessActions(context.Background(), updated.Actions)
	if err != nil {
		l.UsecasesInfo(fmt.Sprintf("failed to parse LLM actions: %v", err), 0)
		// Ошибка — пропускаем
	} else {
		updated.LLMParsedAttack = attacksLLM
	}

	// Обработка действий
	for i, action := range updated.Actions {
		updated.Actions[i].Value = processActionValue(action.Value)
	}

	return &updated, nil
}

func processActionValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}

	// 1. Оборачиваем первую часть до двоеточия в <p><em>...</em>
	colonIdx := strings.Index(value, ":")
	if colonIdx != -1 {
		prefix := strings.TrimSpace(value[:colonIdx])
		rest := strings.TrimSpace(value[colonIdx+1:])
		value = `<p><em>` + prefix + `:</em> ` + rest
	}

	// 2. Заменяем "+15", "-2", "+0" и т.п. после <em> на <dice-roller>
	reAttackBonus := regexp.MustCompile(`([^\w])([+-]\d{1,2})`)
	value = reAttackBonus.ReplaceAllStringFunc(value, func(match string) string {
		// Извлекаем знак и число
		sign := string(match[len(match)-3])
		num := match[len(match)-2:]
		formula := "к20 " + sign + " " + strings.TrimLeft(num, "+-")
		return `</em> <dice-roller label="Атака" formula="` + formula + `">` + sign + num + `</dice-roller>`
	})

	// 3. Заменяем (3к8 + 8), (3к6) и т.п. на dice-roller (урон)
	reDice := regexp.MustCompile(`\((\d+к\d+(?:\s*[+-]\s*\d+)?)\)`)
	value = reDice.ReplaceAllString(value, `(<dice-roller label="Урон" formula="$1"/>)`)

	return value
}
