package models

// CreatureRuntimeState represents the combat state of a creature instance.
// This is stored per-participant in an encounter, separate from the creature template.
// Frontend Participant type should embed or reference this for automation features.
type CreatureRuntimeState struct {
	// Hit Points
	CurrentHP int `json:"currentHP" bson:"currentHP"`
	MaxHP     int `json:"maxHP" bson:"maxHP"`
	TempHP    int `json:"tempHP,omitempty" bson:"tempHP,omitempty"`

	// Active conditions (frightened, poisoned, etc.)
	Conditions []ActiveCondition `json:"conditions,omitempty" bson:"conditions,omitempty"`

	// Resource tracking (spell slots, ability uses, legendary actions)
	Resources ResourceState `json:"resources,omitempty" bson:"resources,omitempty"`

	// Concentration tracking
	Concentration *ConcentrationState `json:"concentration,omitempty" bson:"concentration,omitempty"`

	// Death saves (for player characters at 0 HP)
	DeathSaves *DeathSaveState `json:"deathSaves,omitempty" bson:"deathSaves,omitempty"`

	// Temporary effects that modify stats (e.g., Bless, Bane, Haste)
	StatModifiers []StatModifier `json:"statModifiers,omitempty" bson:"statModifiers,omitempty"`
}

// ActiveCondition represents a condition currently affecting a creature.
type ActiveCondition struct {
	ID        string        `json:"id" bson:"id"` // unique instance ID
	Condition ConditionType `json:"condition" bson:"condition"`
	SourceID  string        `json:"sourceID,omitempty" bson:"sourceID,omitempty"` // who applied it

	// Duration tracking
	Duration     DurationType `json:"duration" bson:"duration"`
	RoundsLeft   int          `json:"roundsLeft,omitempty" bson:"roundsLeft,omitempty"`     // for round-based
	EndsOnTurn   string       `json:"endsOnTurn,omitempty" bson:"endsOnTurn,omitempty"`     // "start" or "end"
	TurnEntityID string       `json:"turnEntityID,omitempty" bson:"turnEntityID,omitempty"` // whose turn triggers end

	// Save to end
	SaveToEnd *SaveToEndCondition `json:"saveToEnd,omitempty" bson:"saveToEnd,omitempty"`

	// Escape (for grapple/restrain)
	EscapeDC   int    `json:"escapeDC,omitempty" bson:"escapeDC,omitempty"`
	EscapeType string `json:"escapeType,omitempty" bson:"escapeType,omitempty"` // "STR", "DEX", "Athletics_vs_Acrobatics"

	// Extra data (exhaustion level, custom description)
	Level       int    `json:"level,omitempty" bson:"level,omitempty"` // for exhaustion (1-6)
	Description string `json:"description,omitempty" bson:"description,omitempty"`
}

// DurationType defines how a condition's duration is tracked.
type DurationType string

const (
	DurationRounds        DurationType = "rounds"        // N rounds
	DurationUntilTurn     DurationType = "until_turn"    // until start/end of someone's turn
	DurationUntilSave     DurationType = "until_save"    // until successful save
	DurationPermanent     DurationType = "permanent"     // until removed (dispel, rest, etc.)
	DurationConcentration DurationType = "concentration" // until caster loses concentration
)

// SaveToEndCondition describes a repeating save to end a condition.
type SaveToEndCondition struct {
	Ability AbilityType `json:"ability" bson:"ability"` // DEX, CON, WIS
	DC      int         `json:"dc" bson:"dc"`
	Timing  string      `json:"timing" bson:"timing"` // "end_of_turn", "start_of_turn"
}

// ResourceState tracks all expendable resources.
type ResourceState struct {
	// Spell slots: level (1-9) -> slots remaining
	SpellSlots map[int]int `json:"spellSlots,omitempty" bson:"spellSlots,omitempty"`

	// Ability uses: action ID -> uses remaining
	AbilityUses map[string]int `json:"abilityUses,omitempty" bson:"abilityUses,omitempty"`

	// Legendary actions remaining (refreshes at start of creature's turn)
	LegendaryActions int `json:"legendaryActions,omitempty" bson:"legendaryActions,omitempty"`

	// Legendary resistances remaining
	LegendaryResistances int `json:"legendaryResistances,omitempty" bson:"legendaryResistances,omitempty"`

	// Lair action availability (true if not used this round)
	LairActionAvailable bool `json:"lairActionAvailable,omitempty" bson:"lairActionAvailable,omitempty"`

	// Recharge tracking: action ID -> true if recharged and ready
	RechargeReady map[string]bool `json:"rechargeReady,omitempty" bson:"rechargeReady,omitempty"`

	// Reaction used this round
	ReactionUsed bool `json:"reactionUsed,omitempty" bson:"reactionUsed,omitempty"`

	// Bonus action used this turn (for some class features)
	BonusActionUsed bool `json:"bonusActionUsed,omitempty" bson:"bonusActionUsed,omitempty"`
}

// ConcentrationState tracks what a creature is concentrating on.
type ConcentrationState struct {
	EffectName   string   `json:"effectName" bson:"effectName"`                   // "Hold Person", "Bless"
	EffectID     string   `json:"effectID,omitempty" bson:"effectID,omitempty"`   // reference to spell/ability
	TargetIDs    []string `json:"targetIDs,omitempty" bson:"targetIDs,omitempty"` // affected creatures
	RoundsActive int      `json:"roundsActive,omitempty" bson:"roundsActive,omitempty"`
	MaxDuration  int      `json:"maxDuration,omitempty" bson:"maxDuration,omitempty"` // in rounds, 0 = unlimited
}

// DeathSaveState tracks death saving throws for creatures at 0 HP.
type DeathSaveState struct {
	Successes int  `json:"successes" bson:"successes"` // 0-3
	Failures  int  `json:"failures" bson:"failures"`   // 0-3
	Stable    bool `json:"stable" bson:"stable"`       // true if stabilized
}

// StatModifier represents a temporary modification to a creature's stats.
type StatModifier struct {
	ID         string           `json:"id" bson:"id"`
	Name       string           `json:"name" bson:"name"` // "Bless", "Shield of Faith"
	SourceID   string           `json:"sourceID,omitempty" bson:"sourceID,omitempty"`
	Modifiers  []ModifierEffect `json:"modifiers" bson:"modifiers"`
	Duration   DurationType     `json:"duration" bson:"duration"`
	RoundsLeft int              `json:"roundsLeft,omitempty" bson:"roundsLeft,omitempty"`
}

// ModifierEffect describes a single stat modification.
type ModifierEffect struct {
	Target    ModifierTarget `json:"target" bson:"target"`       // what is modified
	Operation ModifierOp     `json:"operation" bson:"operation"` // how it's modified
	Value     int            `json:"value,omitempty" bson:"value,omitempty"`
	DiceBonus string         `json:"diceBonus,omitempty" bson:"diceBonus,omitempty"` // "1d4" for Bless
}

// ModifierTarget specifies what stat is being modified.
type ModifierTarget string

const (
	ModTargetAC            ModifierTarget = "ac"
	ModTargetAttackRolls   ModifierTarget = "attack_rolls"
	ModTargetSavingThrows  ModifierTarget = "saving_throws"
	ModTargetAbilityChecks ModifierTarget = "ability_checks"
	ModTargetDamage        ModifierTarget = "damage"
	ModTargetSpeed         ModifierTarget = "speed"
	ModTargetHP            ModifierTarget = "hp" // temp HP or max HP
	ModTargetSTR           ModifierTarget = "str"
	ModTargetDEX           ModifierTarget = "dex"
	ModTargetCON           ModifierTarget = "con"
	ModTargetINT           ModifierTarget = "int"
	ModTargetWIS           ModifierTarget = "wis"
	ModTargetCHA           ModifierTarget = "cha"
)

// ModifierOp defines how the modifier is applied.
type ModifierOp string

const (
	ModOpAdd          ModifierOp = "add"       // +2 to AC
	ModOpMultiply     ModifierOp = "multiply"  // speed * 2 (Haste)
	ModOpSet          ModifierOp = "set"       // set speed to 0
	ModOpAdvantage    ModifierOp = "advantage" // advantage on rolls
	ModOpDisadvantage ModifierOp = "disadvantage"
	ModOpDiceBonus    ModifierOp = "dice_bonus" // +1d4 (Bless, Guidance)
)

// ParticipantFull extends the basic Participant with runtime state.
// This is the complete representation for automation-enabled encounters.
type ParticipantFull struct {
	// Basic fields (matching frontend Participant)
	CreatureID  string            `json:"_id" bson:"_id"` // reference to creature template
	InstanceID  string            `json:"id" bson:"id"`   // unique instance in this encounter
	Initiative  int               `json:"initiative" bson:"initiative"`
	CellsCoords *CellsCoordinates `json:"cellsCoords,omitempty" bson:"cellsCoords,omitempty"`

	// Display overrides (for when DM renames a goblin to "Goblin Chief")
	DisplayName string `json:"displayName,omitempty" bson:"displayName,omitempty"`

	// Ownership for per-player fog/visibility
	OwnerID string `json:"ownerID,omitempty" bson:"ownerID,omitempty"` // player who controls this

	// Runtime combat state
	RuntimeState CreatureRuntimeState `json:"runtimeState,omitempty" bson:"runtimeState,omitempty"`

	// Is this a player character or NPC/monster?
	IsPlayerCharacter bool `json:"isPlayerCharacter,omitempty" bson:"isPlayerCharacter,omitempty"`

	// Hidden from players (for ambushes, invisible enemies)
	Hidden bool `json:"hidden,omitempty" bson:"hidden,omitempty"`
}

// CellsCoordinates represents position on the battle map grid.
type CellsCoordinates struct {
	CellsX int `json:"cellsX" bson:"cellsX"`
	CellsY int `json:"cellsY" bson:"cellsY"`
}
