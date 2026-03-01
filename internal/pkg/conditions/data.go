package conditions

import "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"

func boolPtr(v bool) *bool { return &v }

// allConditions contains all 15 SRD 5e condition definitions.
//
//nolint:funlen // static reference data table
var allConditions = []models.ConditionDefinition{
	{
		Type:        models.ConditionBlinded,
		Name:        models.Name{Rus: "Ослеплён", Eng: "Blinded"},
		Description: models.Name{Rus: "Не может видеть. Броски атаки с помехой. Атаки против с преимуществом.", Eng: "Can't see. Attack rolls have disadvantage. Attacks against have advantage."},
		Effects: models.ConditionEffects{
			AttackRolls:   "disadvantage",
			BeingAttacked: "advantage",
		},
	},
	{
		Type:        models.ConditionCharmed,
		Name:        models.Name{Rus: "Очарован", Eng: "Charmed"},
		Description: models.Name{Rus: "Не может атаковать очарователя. Очарователь с преимуществом на проверки.", Eng: "Can't attack the charmer. Charmer has advantage on social checks."},
		Effects:     models.ConditionEffects{},
	},
	{
		Type:        models.ConditionDeafened,
		Name:        models.Name{Rus: "Оглох", Eng: "Deafened"},
		Description: models.Name{Rus: "Не может слышать. Автопровал проверок слуха.", Eng: "Can't hear. Auto-fails hearing checks."},
		Effects:     models.ConditionEffects{},
	},
	{
		Type:        models.ConditionExhaustion,
		Name:        models.Name{Rus: "Истощение", Eng: "Exhaustion"},
		Description: models.Name{Rus: "Кумулятивные уровни 1-6 с нарастающими штрафами.", Eng: "Cumulative levels 1-6 with increasing penalties."},
		Effects:     models.ConditionEffects{},
		HasLevels:   true,
		MaxLevel:    6,
	},
	{
		Type:        models.ConditionFrightened,
		Name:        models.Name{Rus: "Испуган", Eng: "Frightened"},
		Description: models.Name{Rus: "Помеха на проверки и атаки, пока источник в поле зрения.", Eng: "Disadvantage on ability checks and attack rolls while source is in line of sight."},
		Effects: models.ConditionEffects{
			AttackRolls:   "disadvantage",
			AbilityChecks: "disadvantage",
		},
	},
	{
		Type:        models.ConditionGrappled,
		Name:        models.Name{Rus: "Схвачен", Eng: "Grappled"},
		Description: models.Name{Rus: "Скорость становится 0.", Eng: "Speed becomes 0."},
		Effects: models.ConditionEffects{
			Speed: "zero",
		},
	},
	{
		Type:        models.ConditionIncapacitated,
		Name:        models.Name{Rus: "Недееспособен", Eng: "Incapacitated"},
		Description: models.Name{Rus: "Не может совершать действия и реакции.", Eng: "Can't take actions or reactions."},
		Effects: models.ConditionEffects{
			CanAct:   boolPtr(false),
			CanReact: boolPtr(false),
		},
	},
	{
		Type:        models.ConditionInvisible,
		Name:        models.Name{Rus: "Невидим", Eng: "Invisible"},
		Description: models.Name{Rus: "Броски атаки с преимуществом. Атаки против с помехой.", Eng: "Attack rolls have advantage. Attacks against have disadvantage."},
		Effects: models.ConditionEffects{
			AttackRolls:   "advantage",
			BeingAttacked: "disadvantage",
		},
	},
	{
		Type:        models.ConditionParalyzed,
		Name:        models.Name{Rus: "Парализован", Eng: "Paralyzed"},
		Description: models.Name{Rus: "Недееспособен, не может двигаться. Автопровал спасов СИЛ/ЛОВ. Ближний бой — автокрит.", Eng: "Incapacitated, can't move or speak. Auto-fail STR/DEX saves. Melee hits auto-crit."},
		Effects: models.ConditionEffects{
			CanMove:       boolPtr(false),
			BeingAttacked: "advantage",
			SavingThrows:  map[string]string{"str": "auto_fail", "dex": "auto_fail"},
			MeleeCrits:    true,
		},
		Implies: []models.ConditionType{models.ConditionIncapacitated},
	},
	{
		Type:        models.ConditionPetrified,
		Name:        models.Name{Rus: "Окаменел", Eng: "Petrified"},
		Description: models.Name{Rus: "Превращён в камень. Недееспособен, не осознаёт. Атаки против с преимуществом.", Eng: "Transformed to stone. Incapacitated, unaware. Attacks against have advantage."},
		Effects: models.ConditionEffects{
			CanMove:       boolPtr(false),
			BeingAttacked: "advantage",
		},
		Implies: []models.ConditionType{models.ConditionIncapacitated},
	},
	{
		Type:        models.ConditionPoisoned,
		Name:        models.Name{Rus: "Отравлен", Eng: "Poisoned"},
		Description: models.Name{Rus: "Помеха на броски атаки и проверки характеристик.", Eng: "Disadvantage on attack rolls and ability checks."},
		Effects: models.ConditionEffects{
			AttackRolls:   "disadvantage",
			AbilityChecks: "disadvantage",
		},
	},
	{
		Type:        models.ConditionProne,
		Name:        models.Name{Rus: "Лежит", Eng: "Prone"},
		Description: models.Name{Rus: "Помеха на атаки. Ближний бой в 5фт с преимуществом, дальний с помехой.", Eng: "Disadvantage on attack rolls. Melee within 5ft has advantage, ranged has disadvantage."},
		Effects: models.ConditionEffects{
			AttackRolls: "disadvantage",
		},
	},
	{
		Type:        models.ConditionRestrained,
		Name:        models.Name{Rus: "Скован", Eng: "Restrained"},
		Description: models.Name{Rus: "Скорость 0. Атаки с помехой. Атаки против с преимуществом. Спасы ЛОВ с помехой.", Eng: "Speed 0. Attack rolls have disadvantage. Attacks against have advantage. DEX saves disadvantage."},
		Effects: models.ConditionEffects{
			Speed:         "zero",
			AttackRolls:   "disadvantage",
			BeingAttacked: "advantage",
			SavingThrows:  map[string]string{"dex": "disadvantage"},
		},
	},
	{
		Type:        models.ConditionStunned,
		Name:        models.Name{Rus: "Оглушён", Eng: "Stunned"},
		Description: models.Name{Rus: "Недееспособен, автопровал спасов СИЛ/ЛОВ. Атаки против с преимуществом.", Eng: "Incapacitated, auto-fail STR/DEX saves. Attacks against have advantage."},
		Effects: models.ConditionEffects{
			BeingAttacked: "advantage",
			SavingThrows:  map[string]string{"str": "auto_fail", "dex": "auto_fail"},
		},
		Implies: []models.ConditionType{models.ConditionIncapacitated},
	},
	{
		Type:        models.ConditionUnconscious,
		Name:        models.Name{Rus: "Без сознания", Eng: "Unconscious"},
		Description: models.Name{Rus: "Недееспособен, роняет предметы, падает. Автопровал спасов СИЛ/ЛОВ. Ближний бой — автокрит.", Eng: "Incapacitated, drops items, falls prone. Auto-fail STR/DEX saves. Melee hits auto-crit."},
		Effects: models.ConditionEffects{
			CanMove:       boolPtr(false),
			DropsItems:    true,
			FallsProne:    true,
			BeingAttacked: "advantage",
			MeleeCrits:    true,
			SavingThrows:  map[string]string{"str": "auto_fail", "dex": "auto_fail"},
		},
		Implies: []models.ConditionType{models.ConditionIncapacitated},
	},
}

// AllConditions returns all 15 SRD 5e condition definitions.
func AllConditions() []models.ConditionDefinition {
	return allConditions
}

// FindByType returns the condition definition for the given type, or nil if not found.
func FindByType(t models.ConditionType) *models.ConditionDefinition {
	for i := range allConditions {
		if allConditions[i].Type == t {
			return &allConditions[i]
		}
	}

	return nil
}
