package usecases

import (
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// parseAttack анализирует список действий и возвращает список атак.
func (uc *bestiaryUsecases) parseAttackList(actions []models.Action) *[]models.Attack {
	var attacks []models.Attack

	for _, action := range actions {
		attack, err := uc.parseAttack(action.Name, action.Value)

		if err != nil {
			log.Printf("error while parsing single attack: %v\n", err)

			continue
		}

		attack.Name = action.Name
		attacks = append(attacks, *attack)
	}

	return &attacks
}

// parseAttack парсит строку и возвращает структуру Attack
func (uc *bestiaryUsecases) parseAttack(attackName, text string) (*models.Attack, error) {
	var attack models.Attack

	attack.Name = attackName

	determined, err := uc.determineAttackType(text)
	if err != nil {
		return nil, err
	}

	attack.Type = determined.Type

	// Парсим бонус на попадание
	reToHit := regexp.MustCompile(`<dice-roller[^>]+>\+(\d+)</dice-roller>`)
	toHitMatches := reToHit.FindStringSubmatch(text)

	if len(toHitMatches) > 1 {
		bonus, err := strconv.Atoi(toHitMatches[1])
		if err != nil {
			return nil, apperrors.ParseHitBonusError
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
			return nil, apperrors.ParseDiceError
		}

		dice := models.DiceType("d" + damageMatches[2])

		bonus, err := strconv.Atoi(damageMatches[3])
		if err != nil {
			return nil, apperrors.ParseDamageBonusError
		}

		attack.DamageBonus = bonus

		// Определяем тип урона
		var damageType models.DamageType
		switch {
		case strings.Contains(text, "колющего урона"):
			damageType = models.Piercing
		case strings.Contains(text, "дробящего урона"):
			damageType = models.Bludgeoning
		case strings.Contains(text, "рубящего урона"):
			damageType = models.Slashing
		default:
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

func (uc *bestiaryUsecases) determineAttackType(text string) (*models.DeterminedAttack, error) {
	attackTypes := []models.AttackType{
		models.MeleeWeaponAttack,
		models.RangedWeaponAttack,
		models.MeleeSpellAttack,
		models.RangedSpellAttack,
		models.MeleeOrRangedWeaponAttack,
		models.MeleeOrRangedSpellAttack,
	}

	normalizedText := utils.NormalizeText(text)
	var determined models.DeterminedAttack

	for _, at := range attackTypes {
		attackDescription := at.String("ru")
		normalizedDescription := utils.NormalizeText(attackDescription)

		if strings.Contains(normalizedText, normalizedDescription) {
			determined.Type = at
			determined.Description = attackDescription

			return &determined, nil
		}
	}

	return &determined, apperrors.UnknownAttackTypeError
}
