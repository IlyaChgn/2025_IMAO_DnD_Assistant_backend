package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EncounterFilterParams struct{}

// Condition представляет состояние, которое может быть наложено на существо
type Condition string

// Effect представляет эффект или модификатор, действующий на существо
type Effect struct {
	Name        string         `json:"name" bson:"name"`
	Description string         `json:"description" bson:"description"`
	Duration    int            `json:"duration" bson:"duration"`   // Длительность эффекта в раундах
	Modifiers   map[string]int `json:"modifiers" bson:"modifiers"` // Модификаторы характеристик (например, "ArmorClass": -2)
}

// Encounter представляет энкаунтер в D&D
type Encounter struct {
	ID              primitive.ObjectID   `bson:"_id" json:"id,omitempty"`
	EncounterName   string               `json:"name" bson:"name"`
	Creatures       []*EncounterCreature `json:"creatures" bson:"creatures"`               // Список существ в энкаунтере
	CurrentTurn     int                  `json:"current_turn" bson:"current_turn"`         // Индекс существа, чей сейчас ход
	RoundsCompleted int                  `json:"rounds_completed" bson:"rounds_completed"` // Количество завершенных раундов
}

type EncounterRaw struct {
	EncounterName   string               `json:"name" bson:"name"`
	Creatures       []*EncounterCreature `json:"creatures" bson:"creatures"`               // Список существ в энкаунтере
	CurrentTurn     int                  `json:"current_turn" bson:"current_turn"`         // Индекс существа, чей сейчас ход
	RoundsCompleted int                  `json:"rounds_completed" bson:"rounds_completed"` // Количество завершенных раундов
}

type EncounterShort struct {
	ID            primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	EncounterName string             `json:"name" bson:"name"`
}

// EncounterCreature представляет существо в контексте энкаунтера
type EncounterCreature struct {
	CreatureID    *string     `json:"creature" bson:"creature"`
	Initiative    int         `json:"initiative" bson:"initiative"`       // Инициатива существа
	CurrentHP     int         `json:"current_hp" bson:"current_hp"`       // Текущее HP существа
	Conditions    []Condition `json:"conditions" bson:"conditions"`       // Состояния, наложенные на существо
	Effects       []Effect    `json:"effects" bson:"effects"`             // Эффекты и модификаторы, действующие на существо
	ArmorClass    int         `json:"armor_class" bson:"armor_class"`     // Актуальный класс защиты
	Concentration bool        `json:"concentration" bson:"concentration"` // Наличие концентрации
	ReactionUsed  bool        `json:"reaction_used" bson:"reaction_used"` // Использована ли реакция в этом раунде
	Team          int         `json:"team" bson:"team"`                   // Команда, к которой относится существо (0 - нейтральные, 1 - команда 1, 2 - команда 2 и т.д.)
}

type EncounterReq struct {
	Start  int                   `json:"start"`
	Size   int                   `json:"size"`
	Search SearchParams          `json:"search"`
	Order  []Order               `json:"order"`
	Filter EncounterFilterParams `json:"filter"`
}
