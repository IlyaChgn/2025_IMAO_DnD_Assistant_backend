package usecases

import (
	"context"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"regexp"
	"strconv"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
)

type bestiaryUsecases struct {
	repo bestiaryinterface.BestiaryRepository
}

func NewBestiaryUsecases(repo bestiaryinterface.BestiaryRepository) bestiaryinterface.BestiaryUsecases {
	return &bestiaryUsecases{
		repo: repo,
	}
}

func (uc *bestiaryUsecases) GetCreaturesList(ctx context.Context, size, start int, order []models.Order,
	filter models.FilterParams, search models.SearchParams) ([]*models.BestiaryCreature, error) {
	if start < 0 || size <= 0 {
		return nil, apperrors.StartPosSizeError
	}

	return uc.repo.GetCreaturesList(ctx, size, start, order, filter, search)
}

func determineAttackType(text string) (models.AttackType, string, error) {
	// Список всех типов атак
	attackTypes := []models.AttackType{
		models.MeleeWeaponAttack,
		models.RangedWeaponAttack,
		models.MeleeSpellAttack,
		models.RangedSpellAttack,
		models.MeleeOrRangedWeaponAttack,
		models.MeleeOrRangedSpellAttack,
	}

	// Нормализуем текст: удаляем специальные символы и приводим к нижнему регистру
	normalizedText := utils.NormalizeText(text)

	// Перебираем все типы атак и проверяем, содержится ли их описание в тексте
	for _, at := range attackTypes {
		// Получаем строковое представление типа атаки на русском
		attackDescription := at.String("ru")
		// Нормализуем описание атаки
		normalizedDescription := utils.NormalizeText(attackDescription)

		// Проверяем, содержится ли нормализованное описание в нормализованном тексте
		if strings.Contains(normalizedText, normalizedDescription) {
			// Возвращаем тип атаки и его название
			return at, attackDescription, nil
		}
	}

	// Если тип атаки не найден, возвращаем ошибку
	return 0, "", fmt.Errorf("неизвестный тип атаки")
}

// parseAttack парсит строку и возвращает структуру Attack
func parseAttack(attackName, text string) (*models.Attack, error) {
	var attack models.Attack

	attack.Name = attackName

	attackType, attackName, err := determineAttackType(text)
	if err != nil {
		return nil, err
	}

	attack.Type = attackType

	// Парсим бонус на попадание
	reToHit := regexp.MustCompile(`<dice-roller[^>]+>\+(\d+)</dice-roller>`)
	toHitMatches := reToHit.FindStringSubmatch(text)
	if len(toHitMatches) > 1 {
		bonus, err := strconv.Atoi(toHitMatches[1])
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга бонуса на попадание: %v", err)
		}
		attack.ToHitBonus = bonus
	}

	// Парсим досягаемость (для рукопашной атаки)
	reReach := regexp.MustCompile(`досягаемость (\d+ фт\.)`)
	reachMatches := reReach.FindStringSubmatch(text)
	if len(reachMatches) > 1 {
		attack.Reach = reachMatches[1]
	}

	// Парсим дистанцию (для дальнобойной атаки)
	reRange := regexp.MustCompile(`дистанция (\d+)\/(\d+) фт\.`)
	rangeMatches := reRange.FindStringSubmatch(text)
	if len(rangeMatches) > 2 {
		attack.EffectiveRange = rangeMatches[1] + " фт."
		attack.MaxRange = rangeMatches[2] + " фт."
	}

	// Парсим цель
	attack.Target = models.SingleTarget

	// Парсим урон
	reDamage := regexp.MustCompile(`<dice-roller[^>]+formula="(\d+)к(\d+) \+ (\d+)"`)
	damageMatches := reDamage.FindStringSubmatch(text)
	if len(damageMatches) > 3 {

		count, err := strconv.Atoi(damageMatches[1])
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга количества костей: %v", err)
		}
		dice := models.DiceType("d" + damageMatches[2])
		bonus, err := strconv.Atoi(damageMatches[3])
		if err != nil {
			return nil, fmt.Errorf("ошибка парсинга бонуса урона: %v", err)
		}
		attack.DamageBonus = bonus

		// Определяем тип урона
		var damageType models.DamageType
		if strings.Contains(text, "колющего урона") {
			damageType = models.Piercing
		} else if strings.Contains(text, "дробящего урона") {
			damageType = models.Bludgeoning
		} else if strings.Contains(text, "рубящего урона") {
			damageType = models.Slashing
		} else {
			return nil, fmt.Errorf("неизвестный тип урона")
		}

		attack.Damage = []models.Damage{
			{
				Dice:       dice,
				Count:      count,
				DamageType: damageType,
			},
		}
	}

	// Парсим боеприпасы (если есть)
	reAmmo := regexp.MustCompile(`носит с собой (\d+) болтов для арбалета`)
	ammoMatches := reAmmo.FindStringSubmatch(text)
	if len(ammoMatches) > 1 {
		attack.Ammo = ammoMatches[1] + " болтов для арбалета"
	}

	return &attack, nil
}

// parseAttack анализирует список действий и возвращает список атак.
func parseAttackList(actions []models.Action) ([]models.Attack, error) {
	var attacks []models.Attack

	for _, action := range actions {
		// Пример логики: если имя действия содержит "Attack", считаем его атакой.

		attack, err := parseAttack(action.Name, action.Value)

		if err != nil {
			fmt.Printf("error while parsing single attack: %v", err)

			continue
		}

		attack.Name = action.Name

		attacks = append(attacks, *attack)

	}

	return attacks, nil
}

func (uc *bestiaryUsecases) GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error) {

	creature, err := uc.repo.GetCreatureByEngName(ctx, engName)

	if err != nil {
		fmt.Printf("mongo error most likely: %v", err)

		return nil, err
	}

	attacks, err := parseAttackList(creature.Actions)

	if err != nil {
		fmt.Printf("error whole parsing attack lisk: %v", err)
	}

	creature.Attacks = attacks

	return creature, nil
}
