package models

// ItemModifierTarget represents the target of a modifier effect.
// Discriminated union via the Type field.
type ItemModifierTarget struct {
	Type       string `json:"type" bson:"type"`
	Ability    string `json:"ability,omitempty" bson:"ability,omitempty"`
	Skill      string `json:"skill,omitempty" bson:"skill,omitempty"`
	Movement   string `json:"movement,omitempty" bson:"movement,omitempty"`
	DamageType string `json:"damageType,omitempty" bson:"damageType,omitempty"`
}

// ItemModifierCondition represents when a modifier is active.
type ItemModifierCondition string

const (
	ModifierConditionAlways         ItemModifierCondition = "always"
	ModifierConditionWhileEquipped  ItemModifierCondition = "while_equipped"
	ModifierConditionWhileAttuned   ItemModifierCondition = "while_attuned"
	ModifierConditionInCombat       ItemModifierCondition = "in_combat"
	ModifierConditionWearingNoArmor ItemModifierCondition = "wearing_no_armor"
)

// ItemModifierDef represents a modifier that an item applies to a character.
type ItemModifierDef struct {
	ID        string                `json:"id" bson:"id"`
	Source    string                `json:"source" bson:"source"`
	Target    ItemModifierTarget    `json:"target" bson:"target"`
	Operation string                `json:"operation" bson:"operation"`
	Value     float64               `json:"value" bson:"value"`
	Priority  int                   `json:"priority" bson:"priority"`
	Tag       string                `json:"tag,omitempty" bson:"tag,omitempty"`
	Condition ItemModifierCondition `json:"condition,omitempty" bson:"condition,omitempty"`
	Unique    bool                  `json:"unique,omitempty" bson:"unique,omitempty"`
}
