package models

type AttackType int

const (
	MeleeWeaponAttack AttackType = iota
	RangedWeaponAttack
	MeleeSpellAttack
	RangedSpellAttack
	MeleeOrRangedWeaponAttack
	MeleeOrRangedSpellAttack
)

type TargetType int

const (
	SingleTarget        TargetType = iota // Одна цель
	Cone                                  // Конус
	Cube                                  // Куб
	Sphere                                // Сфера
	Cylinder                              // Цилиндр
	Line                                  // Линия
	Self                                  // Сам на себя
	Touch                                 // Касание
	MultipleTargets                       // Несколько целей
	Object                                // Объект
	Point                                 // Точка в пространстве
	AllCreaturesInRange                   // Все существа в радиусе
	AllEnemiesInRange                     // Все враги в радиусе
	AllAlliesInRange                      // Все союзники в радиусе
)

type DamageType int

const (
	Acid DamageType = iota
	Bludgeoning
	Cold
	Fire
	Force
	Lightning
	Necrotic
	Piercing
	Poison
	Psychic
	Radiant
	Slashing
	Thunder
)

type DiceType string

const (
	D4   DiceType = "d4"
	D6   DiceType = "d6"
	D8   DiceType = "d8"
	D10  DiceType = "d10"
	D12  DiceType = "d12"
	D20  DiceType = "d20"
	D100 DiceType = "d100"
)

// Damage описывает одну кость урона
type Damage struct {
	Dice       DiceType   `json:"dice" bson:"dice"`             // Тип кости
	Count      int        `json:"count" bson:"count"`           // Количество костей
	DamageType DamageType `json:"damageType" bson:"damageType"` // Тип урона
}

type DeterminedAttack struct {
	Type        AttackType
	Description string
}

/////////////////////////// LLM PARSED ATTACK ////////////////////////////////////////

type DamageLLM struct {
	Dice  string `bson:"dice" json:"dice"`
	Count int    `bson:"count" json:"count"`
	Type  string `bson:"type" json:"type"`
	Bonus int    `bson:"bonus" json:"bonus"`
}

type AdditionalEffectLLM struct {
	Damage    *DamageLLM `bson:"damage,omitempty" json:"damage,omitempty"`
	Condition string     `bson:"condition,omitempty" json:"condition,omitempty"`
	EscapeDC  int        `bson:"escape_dc,omitempty" json:"escapeDc,omitempty"`
}

type MultiAttackLLM struct {
	Type  string `bson:"type" json:"type"`
	Count int    `bson:"count" json:"count"`
}

type AreaAttackLLM struct {
	Shape     string `bson:"shape,omitempty" json:"shape,omitempty"`
	Recharge  string `bson:"recharge,omitempty" json:"recharge,omitempty"`
	SaveDC    int    `bson:"save_dc,omitempty" json:"saveDc,omitempty"`
	SaveType  string `bson:"save_type,omitempty" json:"saveType,omitempty"`
	OnFail    string `bson:"on_fail,omitempty" json:"onSail,omitempty"`
	OnSuccess string `bson:"on_success,omitempty" json:"onSuccess,omitempty"`
}

type AttackLLM struct {
	Name              string                `bson:"name" json:"name"`
	Type              string                `bson:"type,omitempty" json:"type,omitempty"` // melee, ranged, area и т.д.
	AttackBonus       string                `bson:"attack_bonus,omitempty" json:"attackBonus,omitempty"`
	Reach             string                `bson:"reach,omitempty" json:"reach,omitempty"` // для ближних атак
	Range             string                `bson:"range,omitempty" json:"range,omitempty"` // для дальних атак
	Target            string                `bson:"target,omitempty" json:"target,omitempty"`
	Damage            *DamageLLM            `bson:"damage,omitempty" json:"damage,omitempty"`
	Attacks           []MultiAttackLLM      `bson:"attacks,omitempty" json:"attacks,omitempty"` // для мультиатак
	AdditionalEffects []AdditionalEffectLLM `bson:"additional_effects,omitempty" json:"additionalEffects,omitempty"`
	Area              *AreaAttackLLM        `bson:"area,omitempty" json:"area,omitempty"`   // для зональных атак
	Shape             string                `bson:"shape,omitempty" json:"shape,omitempty"` // альт. вариант для area
	Recharge          string                `bson:"recharge,omitempty" json:"recharge,omitempty"`
	SaveDC            int                   `bson:"save_dc,omitempty" json:"saveDc,omitempty"`
	SaveType          string                `bson:"save_type,omitempty" json:"saveType,omitempty"`
	OnFail            string                `bson:"on_fail,omitempty" json:"onFail,omitempty"`
	OnSuccess         string                `bson:"on_success,omitempty" json:"onSuccess,omitempty"`
}

func (at AttackType) String(lang string) string {
	switch at {
	case MeleeWeaponAttack:
		if lang == "ru" {
			return "Рукопашная атака оружием"
		}
		return "Melee weapon attack"
	case RangedWeaponAttack:
		if lang == "ru" {
			return "Дальнобойная атака оружием"
		}
		return "Ranged weapon attack"
	case MeleeSpellAttack:
		if lang == "ru" {
			return "Рукопашная атака заклинанием"
		}
		return "Melee spell attack"
	case RangedSpellAttack:
		if lang == "ru" {
			return "Дальнобойная атака заклинанием"
		}
		return "Ranged spell attack"
	case MeleeOrRangedWeaponAttack:
		if lang == "ru" {
			return "Рукопашная или дальнобойная атака оружием"
		}
		return "Melee or ranged weapon attack"
	case MeleeOrRangedSpellAttack:
		if lang == "ru" {
			return "Рукопашная или дальнобойная атака заклинанием"
		}
		return "Melee or ranged spell attack"
	default:
		if lang == "ru" {
			return "Неизвестный тип атаки"
		}
		return "Unknown attack type"
	}
}

func (tt TargetType) String(lang string) string {
	switch tt {
	case SingleTarget:
		if lang == "ru" {
			return "одна цель"
		}
		return "single target"
	case Cone:
		if lang == "ru" {
			return "конус"
		}
		return "cone"
	case Cube:
		if lang == "ru" {
			return "куб"
		}
		return "cube"
	case Sphere:
		if lang == "ru" {
			return "сфера"
		}
		return "sphere"
	case Cylinder:
		if lang == "ru" {
			return "цилиндр"
		}
		return "cylinder"
	case Line:
		if lang == "ru" {
			return "линия"
		}
		return "line"
	case Self:
		if lang == "ru" {
			return "сам на себя"
		}
		return "self"
	case Touch:
		if lang == "ru" {
			return "касание"
		}
		return "touch"
	case MultipleTargets:
		if lang == "ru" {
			return "несколько целей"
		}
		return "multiple targets"
	case Object:
		if lang == "ru" {
			return "объект"
		}
		return "object"
	case Point:
		if lang == "ru" {
			return "точка в пространстве"
		}
		return "point"
	case AllCreaturesInRange:
		if lang == "ru" {
			return "все существа в радиусе"
		}
		return "all creatures in range"
	case AllEnemiesInRange:
		if lang == "ru" {
			return "все враги в радиусе"
		}
		return "all enemies in range"
	case AllAlliesInRange:
		if lang == "ru" {
			return "все союзники в радиусе"
		}
		return "all allies in range"
	default:
		if lang == "ru" {
			return "неизвестная цель"
		}
		return "unknown target"
	}
}

func (dt DamageType) String(lang string) string {
	switch dt {
	case Acid:
		if lang == "ru" {
			return "кислотный"
		}
		return "acid"
	case Bludgeoning:
		if lang == "ru" {
			return "дробящий"
		}
		return "bludgeoning"
	case Cold:
		if lang == "ru" {
			return "холод"
		}
		return "cold"
	case Fire:
		if lang == "ru" {
			return "огненный"
		}
		return "fire"
	case Force:
		if lang == "ru" {
			return "силовой"
		}
		return "force"
	case Lightning:
		if lang == "ru" {
			return "молния"
		}
		return "lightning"
	case Necrotic:
		if lang == "ru" {
			return "некротический"
		}
		return "necrotic"
	case Piercing:
		if lang == "ru" {
			return "колющий"
		}
		return "piercing"
	case Poison:
		if lang == "ru" {
			return "ядовитый"
		}
		return "poison"
	case Psychic:
		if lang == "ru" {
			return "психический"
		}
		return "psychic"
	case Radiant:
		if lang == "ru" {
			return "светящийся"
		}
		return "radiant"
	case Slashing:
		if lang == "ru" {
			return "рубящий"
		}
		return "slashing"
	case Thunder:
		if lang == "ru" {
			return "громовой"
		}
		return "thunder"
	default:
		if lang == "ru" {
			return "неизвестный тип урона"
		}
		return "unknown damage type"
	}
}
