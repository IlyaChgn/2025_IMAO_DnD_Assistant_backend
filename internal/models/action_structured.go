package models

// ActionCategory defines when an action can be used in combat.
type ActionCategory string

const (
	ActionCategoryAction    ActionCategory = "action"
	ActionCategoryBonus     ActionCategory = "bonus_action"
	ActionCategoryReaction  ActionCategory = "reaction"
	ActionCategoryLegendary ActionCategory = "legendary"
	ActionCategoryLair      ActionCategory = "lair"
	ActionCategoryFree      ActionCategory = "free" // e.g., dropping an item
)

// StructuredAction represents a machine-readable creature action for automation.
// It coexists with the text-based Action/BonusAction/Reaction types for backward compatibility.
type StructuredAction struct {
	ID          string         `json:"id" bson:"id"`
	Name        string         `json:"name" bson:"name"`
	Description string         `json:"description" bson:"description"` // original text for display
	Category    ActionCategory `json:"category" bson:"category"`

	// Attack roll actions (melee/ranged weapon or spell attacks)
	Attack *AttackRollData `json:"attack,omitempty" bson:"attack,omitempty"`

	// Saving throw actions (breath weapons, spells, abilities)
	SavingThrow *SavingThrowData `json:"savingThrow,omitempty" bson:"savingThrow,omitempty"`

	// Resource management
	Uses     *UsesData     `json:"uses,omitempty" bson:"uses,omitempty"`
	Recharge *RechargeData `json:"recharge,omitempty" bson:"recharge,omitempty"`

	// Legendary action cost (1-3)
	LegendaryCost int `json:"legendaryCost,omitempty" bson:"legendaryCost,omitempty"`

	// Effects applied on hit/fail
	Effects []ActionEffect `json:"effects,omitempty" bson:"effects,omitempty"`
}

// AttackRollData describes an attack that requires a d20 roll vs AC.
type AttackRollData struct {
	Type    AttackRollType `json:"type" bson:"type"`
	Bonus   int            `json:"bonus" bson:"bonus"`                     // +7 to hit
	Reach   int            `json:"reach,omitempty" bson:"reach,omitempty"` // melee reach in feet
	Range   *RangeData     `json:"range,omitempty" bson:"range,omitempty"` // ranged attack distances
	Targets int            `json:"targets" bson:"targets"`                 // number of targets (usually 1)
	Damage  []DamageRoll   `json:"damage" bson:"damage"`
}

// AttackRollType categorizes the attack for rules purposes.
type AttackRollType string

const (
	AttackRollMeleeWeapon  AttackRollType = "melee_weapon"
	AttackRollRangedWeapon AttackRollType = "ranged_weapon"
	AttackRollMeleeSpell   AttackRollType = "melee_spell"
	AttackRollRangedSpell  AttackRollType = "ranged_spell"
)

// RangeData describes normal and long range for ranged attacks.
type RangeData struct {
	Normal int `json:"normal" bson:"normal"`                 // normal range in feet
	Long   int `json:"long,omitempty" bson:"long,omitempty"` // long range (disadvantage)
}

// DamageRoll describes a single damage component.
type DamageRoll struct {
	DiceCount  int    `json:"diceCount" bson:"diceCount"`             // 2
	DiceType   string `json:"diceType" bson:"diceType"`               // "d6"
	Bonus      int    `json:"bonus,omitempty" bson:"bonus,omitempty"` // +4
	DamageType string `json:"damageType" bson:"damageType"`           // "slashing", "fire", etc.

	// Conditional damage (e.g., "extra 2d6 fire on crit", "extra 1d8 vs undead")
	Condition string `json:"condition,omitempty" bson:"condition,omitempty"`
}

// SavingThrowData describes an effect that requires a saving throw.
type SavingThrowData struct {
	Ability   AbilityType   `json:"ability" bson:"ability"` // DEX, CON, WIS, etc.
	DC        int           `json:"dc" bson:"dc"`
	OnFail    string        `json:"onFail" bson:"onFail"`       // "full damage", "half damage and prone"
	OnSuccess string        `json:"onSuccess" bson:"onSuccess"` // "half damage", "no effect"
	Damage    []DamageRoll  `json:"damage,omitempty" bson:"damage,omitempty"`
	Area      *AreaOfEffect `json:"area,omitempty" bson:"area,omitempty"`
}

// AbilityType represents the six D&D ability scores.
type AbilityType string

const (
	AbilitySTR AbilityType = "STR"
	AbilityDEX AbilityType = "DEX"
	AbilityCON AbilityType = "CON"
	AbilityINT AbilityType = "INT"
	AbilityWIS AbilityType = "WIS"
	AbilityCHA AbilityType = "CHA"
)

// AreaOfEffect describes the shape and size of an area effect.
type AreaOfEffect struct {
	Shape  AreaShape `json:"shape" bson:"shape"`
	Size   int       `json:"size" bson:"size"`                         // radius/length in feet
	Width  int       `json:"width,omitempty" bson:"width,omitempty"`   // for lines
	Origin string    `json:"origin,omitempty" bson:"origin,omitempty"` // "self", "point"
}

// AreaShape defines the geometric shape of an area effect.
type AreaShape string

const (
	AreaShapeCone     AreaShape = "cone"
	AreaShapeCube     AreaShape = "cube"
	AreaShapeCylinder AreaShape = "cylinder"
	AreaShapeLine     AreaShape = "line"
	AreaShapeSphere   AreaShape = "sphere"
)

// UsesData tracks limited-use abilities.
type UsesData struct {
	Max int         `json:"max" bson:"max"`
	Per RestoreType `json:"per" bson:"per"` // "day", "short_rest", "long_rest"
}

// RestoreType defines when uses are restored.
type RestoreType string

const (
	RestorePerDay    RestoreType = "day"
	RestoreShortRest RestoreType = "short_rest"
	RestoreLongRest  RestoreType = "long_rest"
)

// RechargeData tracks abilities that recharge on a roll (e.g., "Recharge 5-6").
type RechargeData struct {
	MinRoll int `json:"minRoll" bson:"minRoll"` // 5 for "Recharge 5-6", 6 for "Recharge 6"
}

// ActionEffect describes an effect applied by an action.
type ActionEffect struct {
	// Condition effects (grappled, poisoned, frightened, etc.)
	Condition *ConditionEffect `json:"condition,omitempty" bson:"condition,omitempty"`

	// Ongoing damage (e.g., "takes 1d6 fire damage at the start of each turn")
	OngoingDamage *OngoingDamageEffect `json:"ongoingDamage,omitempty" bson:"ongoingDamage,omitempty"`

	// Movement effects (push, pull, teleport)
	Movement *MovementEffect `json:"movement,omitempty" bson:"movement,omitempty"`

	// Healing
	Healing *HealingEffect `json:"healing,omitempty" bson:"healing,omitempty"`

	// Description for non-standard effects that don't fit other categories
	// Examples: "скорость уменьшается на 10 футов", "заражена синегнилью", "проклята ликантропией"
	Description string `json:"description,omitempty" bson:"description,omitempty"`
}

// ConditionEffect applies a D&D condition to a target.
type ConditionEffect struct {
	Condition  ConditionType `json:"condition" bson:"condition"`
	Duration   string        `json:"duration" bson:"duration"`                         // "1 minute", "until end of next turn"
	SaveEnds   bool          `json:"saveEnds,omitempty" bson:"saveEnds,omitempty"`     // can repeat save at end of turn
	EscapeDC   int           `json:"escapeDC,omitempty" bson:"escapeDC,omitempty"`     // for grapple/restrain
	EscapeType string        `json:"escapeType,omitempty" bson:"escapeType,omitempty"` // "STR", "DEX", "STR_or_DEX"
}

// ConditionType represents standard D&D 5e conditions.
type ConditionType string

const (
	ConditionBlinded       ConditionType = "blinded"
	ConditionCharmed       ConditionType = "charmed"
	ConditionDeafened      ConditionType = "deafened"
	ConditionFrightened    ConditionType = "frightened"
	ConditionGrappled      ConditionType = "grappled"
	ConditionIncapacitated ConditionType = "incapacitated"
	ConditionInvisible     ConditionType = "invisible"
	ConditionParalyzed     ConditionType = "paralyzed"
	ConditionPetrified     ConditionType = "petrified"
	ConditionPoisoned      ConditionType = "poisoned"
	ConditionProne         ConditionType = "prone"
	ConditionRestrained    ConditionType = "restrained"
	ConditionStunned       ConditionType = "stunned"
	ConditionUnconscious   ConditionType = "unconscious"
	ConditionExhaustion    ConditionType = "exhaustion"
)

// OngoingDamageEffect represents damage that occurs over time.
type OngoingDamageEffect struct {
	Damage   DamageRoll  `json:"damage" bson:"damage"`
	Trigger  string      `json:"trigger" bson:"trigger"` // "start_of_turn", "end_of_turn"
	Duration string      `json:"duration" bson:"duration"`
	SaveEnds bool        `json:"saveEnds,omitempty" bson:"saveEnds,omitempty"`
	SaveType AbilityType `json:"saveType,omitempty" bson:"saveType,omitempty"`
	SaveDC   int         `json:"saveDC,omitempty" bson:"saveDC,omitempty"`
}

// MovementEffect represents forced movement.
type MovementEffect struct {
	Type     MovementEffectType `json:"type" bson:"type"`
	Distance int                `json:"distance" bson:"distance"` // in feet
}

// MovementEffectType categorizes forced movement.
type MovementEffectType string

const (
	MovementPush      MovementEffectType = "push"
	MovementPull      MovementEffectType = "pull"
	MovementTeleport  MovementEffectType = "teleport"
	MovementKnockdown MovementEffectType = "knockdown" // results in prone
)

// HealingEffect represents healing from an action.
type HealingEffect struct {
	DiceCount int    `json:"diceCount" bson:"diceCount"`
	DiceType  string `json:"diceType" bson:"diceType"`
	Bonus     int    `json:"bonus,omitempty" bson:"bonus,omitempty"`
}
