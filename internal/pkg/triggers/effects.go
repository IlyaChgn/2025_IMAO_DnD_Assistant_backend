package triggers

import (
	"fmt"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
)

// Effect executor param contracts (diverges from design doc § 5.4):
//
//	deal_damage:      dice (string, e.g. "2d6"), damageType (string, e.g. "fire")
//	heal:             dice (string) OR amount (int) — one required
//	apply_condition:  condition (string), duration (string, optional)
//	remove_condition: conditions ([]string, non-empty)
//	grant_temp_hp:    dice (string) OR amount (int) — one required

func executeDealDamage(params map[string]interface{}, source string) (models.TriggerResult, error) {
	diceExpr, err := getString(params, "dice")
	if err != nil {
		return models.TriggerResult{}, err
	}
	damageType, err := getString(params, "damageType")
	if err != nil {
		return models.TriggerResult{}, err
	}

	roll, err := dice.Roll(diceExpr)
	if err != nil {
		return models.TriggerResult{}, fmt.Errorf("deal_damage dice roll: %w", err)
	}

	return models.TriggerResult{
		EffectType:  models.EffectDealDamage,
		Description: fmt.Sprintf("%s deals %d %s damage", source, roll.Total, damageType),
		DamageResult: &models.TriggerDamageResult{
			Dice:       diceExpr,
			DamageType: damageType,
			Rolls:      roll.Rolls,
			Total:      roll.Total,
		},
	}, nil
}

func executeHeal(params map[string]interface{}, source string) (models.TriggerResult, error) {
	diceExpr := getStringOptional(params, "dice")

	// Flat amount fallback (e.g. Ring of Regeneration: amount=1)
	if diceExpr == "" {
		amount, err := getInt(params, "amount")
		if err != nil {
			return models.TriggerResult{}, fmt.Errorf("heal requires 'dice' or 'amount' param")
		}
		return models.TriggerResult{
			EffectType:  models.EffectHeal,
			Description: fmt.Sprintf("%s heals %d HP", source, amount),
			HealResult: &models.TriggerHealResult{
				Total: amount,
			},
		}, nil
	}

	roll, err := dice.Roll(diceExpr)
	if err != nil {
		return models.TriggerResult{}, fmt.Errorf("heal dice roll: %w", err)
	}

	return models.TriggerResult{
		EffectType:  models.EffectHeal,
		Description: fmt.Sprintf("%s heals %d HP", source, roll.Total),
		HealResult: &models.TriggerHealResult{
			Dice:  diceExpr,
			Rolls: roll.Rolls,
			Total: roll.Total,
		},
	}, nil
}

func executeApplyCondition(params map[string]interface{}, source string) (models.TriggerResult, error) {
	condition, err := getString(params, "condition")
	if err != nil {
		return models.TriggerResult{}, err
	}
	duration := getStringOptional(params, "duration")

	desc := fmt.Sprintf("%s applies %s", source, condition)
	if duration != "" {
		desc += fmt.Sprintf(" for %s", duration)
	}

	return models.TriggerResult{
		EffectType:  models.EffectApplyCondition,
		Description: desc,
		ConditionResult: &models.TriggerConditionResult{
			Action:    "apply",
			Condition: condition,
			Duration:  duration,
		},
	}, nil
}

func executeRemoveCondition(params map[string]interface{}, source string) (models.TriggerResult, error) {
	conditions, err := getStringSlice(params, "conditions")
	if err != nil {
		return models.TriggerResult{}, err
	}

	return models.TriggerResult{
		EffectType:  models.EffectRemoveCondition,
		Description: fmt.Sprintf("%s removes conditions: %s", source, strings.Join(conditions, ", ")),
		ConditionResult: &models.TriggerConditionResult{
			Action:     "remove",
			Conditions: conditions,
		},
	}, nil
}

func executeGrantTempHP(params map[string]interface{}, source string) (models.TriggerResult, error) {
	diceExpr := getStringOptional(params, "dice")

	// Flat amount fallback (e.g. Armor of Resistance: amount=10)
	if diceExpr == "" {
		amount, err := getInt(params, "amount")
		if err != nil {
			return models.TriggerResult{}, fmt.Errorf("grant_temp_hp requires 'dice' or 'amount' param")
		}
		return models.TriggerResult{
			EffectType:  models.EffectGrantTempHP,
			Description: fmt.Sprintf("%s grants %d temporary HP", source, amount),
			TempHPResult: &models.TriggerTempHPResult{
				Amount: amount,
			},
		}, nil
	}

	roll, err := dice.Roll(diceExpr)
	if err != nil {
		return models.TriggerResult{}, fmt.Errorf("grant_temp_hp dice roll: %w", err)
	}

	return models.TriggerResult{
		EffectType:  models.EffectGrantTempHP,
		Description: fmt.Sprintf("%s grants %d temporary HP", source, roll.Total),
		TempHPResult: &models.TriggerTempHPResult{
			Amount: roll.Total,
			Dice:   diceExpr,
			Rolls:  roll.Rolls,
		},
	}, nil
}
