package models

// Spellcasting represents a creature's spellcasting ability (from statblock).
// This is the template - actual spell slot usage is tracked in CreatureRuntimeState.Resources.
type Spellcasting struct {
	// Spellcasting ability (INT for wizards, WIS for clerics, CHA for sorcerers)
	Ability AbilityType `json:"ability" bson:"ability"`

	// Spell save DC = 8 + proficiency + ability modifier
	SpellSaveDC int `json:"spellSaveDC" bson:"spellSaveDC"`

	// Spell attack modifier = proficiency + ability modifier
	SpellAttackBonus int `json:"spellAttackBonus" bson:"spellAttackBonus"`

	// Spell slots by level (1-9). Value = number of slots at that level.
	// Example: {1: 4, 2: 3, 3: 2} = 4 first-level, 3 second-level, 2 third-level
	SpellSlots map[int]int `json:"spellSlots,omitempty" bson:"spellSlots,omitempty"`

	// Caster level (affects cantrip scaling, some spell effects)
	CasterLevel int `json:"casterLevel,omitempty" bson:"casterLevel,omitempty"`

	// Spells known/prepared, organized by level (0 = cantrips)
	// Key is spell level, value is list of spells
	SpellsByLevel map[int][]SpellKnown `json:"spellsByLevel,omitempty" bson:"spellsByLevel,omitempty"`

	// Flat list alternative (for simpler creatures)
	Spells []SpellKnown `json:"spells,omitempty" bson:"spells,omitempty"`

	// Spellcasting class/type description (for display)
	Description string `json:"description,omitempty" bson:"description,omitempty"`
}

// InnateSpellcasting represents innate spellcasting abilities (racial, monster).
// These don't use spell slots - they have their own usage limits.
type InnateSpellcasting struct {
	// Spellcasting ability
	Ability AbilityType `json:"ability" bson:"ability"`

	// Spell save DC
	SpellSaveDC int `json:"spellSaveDC" bson:"spellSaveDC"`

	// Spell attack bonus (if any)
	SpellAttackBonus int `json:"spellAttackBonus,omitempty" bson:"spellAttackBonus,omitempty"`

	// At will spells (unlimited uses)
	AtWill []SpellKnown `json:"atWill,omitempty" bson:"atWill,omitempty"`

	// Limited use spells, grouped by uses per day
	// Key is uses per day (3, 2, 1), value is list of spells
	PerDay map[int][]SpellKnown `json:"perDay,omitempty" bson:"perDay,omitempty"`

	// Special note (e.g., "requires no material components")
	Note string `json:"note,omitempty" bson:"note,omitempty"`
}

// SpellKnown represents a spell in a creature's spell list.
type SpellKnown struct {
	// Spell identifier (for lookup in spell database)
	SpellID string `json:"spellID,omitempty" bson:"spellID,omitempty"`

	// Spell name (always present for display)
	Name string `json:"name" bson:"name"`

	// Spell level (0 = cantrip, 1-9 = spell levels)
	Level int `json:"level" bson:"level"`

	// School of magic
	School SpellSchool `json:"school,omitempty" bson:"school,omitempty"`

	// Quick reference data (optional, can be populated from spell database)
	QuickRef *SpellQuickRef `json:"quickRef,omitempty" bson:"quickRef,omitempty"`
}

// SpellQuickRef contains commonly-needed spell data for quick reference.
// Full spell details would come from a separate spell database.
type SpellQuickRef struct {
	CastingTime   string `json:"castingTime" bson:"castingTime"` // "1 action", "1 bonus action"
	Range         string `json:"range" bson:"range"`             // "60 feet", "Self", "Touch"
	Duration      string `json:"duration" bson:"duration"`       // "1 minute", "Instantaneous"
	Concentration bool   `json:"concentration" bson:"concentration"`
	Ritual        bool   `json:"ritual,omitempty" bson:"ritual,omitempty"`
	Components    string `json:"components" bson:"components"` // "V, S, M (a bit of fur)"
}

// SpellSchool represents the eight schools of magic in D&D.
type SpellSchool string

const (
	SchoolAbjuration    SpellSchool = "abjuration"
	SchoolConjuration   SpellSchool = "conjuration"
	SchoolDivination    SpellSchool = "divination"
	SchoolEnchantment   SpellSchool = "enchantment"
	SchoolEvocation     SpellSchool = "evocation"
	SchoolIllusion      SpellSchool = "illusion"
	SchoolNecromancy    SpellSchool = "necromancy"
	SchoolTransmutation SpellSchool = "transmutation"
)

// Spell represents a complete spell definition (for spell database).
// This would be stored separately from creatures, referenced by SpellID.
type Spell struct {
	ID     string      `json:"id" bson:"_id"`
	Name   Name        `json:"name" bson:"name"` // rus/eng names
	Level  int         `json:"level" bson:"level"`
	School SpellSchool `json:"school" bson:"school"`

	// Casting
	CastingTime CastingTime     `json:"castingTime" bson:"castingTime"`
	Range       SpellRange      `json:"range" bson:"range"`
	Components  SpellComponents `json:"components" bson:"components"`
	Duration    SpellDuration   `json:"duration" bson:"duration"`

	// Effects
	Description  string `json:"description" bson:"description"`
	HigherLevels string `json:"higherLevels,omitempty" bson:"higherLevels,omitempty"`

	// Automation data
	Effects []SpellEffect `json:"effects,omitempty" bson:"effects,omitempty"`

	// Metadata
	Classes []string `json:"classes,omitempty" bson:"classes,omitempty"` // wizard, cleric, etc.
	Source  Source   `json:"source,omitempty" bson:"source,omitempty"`
}

// CastingTime describes how long it takes to cast a spell.
type CastingTime struct {
	Type     CastingTimeType `json:"type" bson:"type"`
	Amount   int             `json:"amount,omitempty" bson:"amount,omitempty"`     // for minutes/hours
	Reaction string          `json:"reaction,omitempty" bson:"reaction,omitempty"` // trigger for reactions
}

// CastingTimeType categorizes casting times.
type CastingTimeType string

const (
	CastAction      CastingTimeType = "action"
	CastBonusAction CastingTimeType = "bonus_action"
	CastReaction    CastingTimeType = "reaction"
	CastMinutes     CastingTimeType = "minutes"
	CastHours       CastingTimeType = "hours"
)

// SpellRange describes the range of a spell.
type SpellRange struct {
	Type     SpellRangeType `json:"type" bson:"type"`
	Distance int            `json:"distance,omitempty" bson:"distance,omitempty"` // in feet
}

// SpellRangeType categorizes spell ranges.
type SpellRangeType string

const (
	RangeSelf      SpellRangeType = "self"
	RangeTouch     SpellRangeType = "touch"
	RangeDistance  SpellRangeType = "distance"
	RangeSight     SpellRangeType = "sight"
	RangeUnlimited SpellRangeType = "unlimited"
)

// SpellComponents describes the components required to cast a spell.
type SpellComponents struct {
	Verbal           bool   `json:"verbal" bson:"verbal"`
	Somatic          bool   `json:"somatic" bson:"somatic"`
	Material         bool   `json:"material" bson:"material"`
	Materials        string `json:"materials,omitempty" bson:"materials,omitempty"`       // description
	MaterialCost     int    `json:"materialCost,omitempty" bson:"materialCost,omitempty"` // gp, if consumed
	MaterialConsumed bool   `json:"materialConsumed,omitempty" bson:"materialConsumed,omitempty"`
}

// SpellDuration describes how long a spell lasts.
type SpellDuration struct {
	Type          SpellDurationType `json:"type" bson:"type"`
	Amount        int               `json:"amount,omitempty" bson:"amount,omitempty"`
	Unit          string            `json:"unit,omitempty" bson:"unit,omitempty"` // "rounds", "minutes", "hours"
	Concentration bool              `json:"concentration" bson:"concentration"`
	UpTo          bool              `json:"upTo,omitempty" bson:"upTo,omitempty"` // "up to 1 minute"
}

// SpellDurationType categorizes spell durations.
type SpellDurationType string

const (
	DurationInstantaneous  SpellDurationType = "instantaneous"
	DurationTimed          SpellDurationType = "timed"
	DurationUntilDispelled SpellDurationType = "until_dispelled"
	DurationSpecial        SpellDurationType = "special"
)

// SpellEffect describes a mechanical effect of a spell for automation.
type SpellEffect struct {
	// What triggers this effect
	Trigger SpellEffectTrigger `json:"trigger" bson:"trigger"`

	// Targeting
	Target SpellTarget   `json:"target" bson:"target"`
	Area   *AreaOfEffect `json:"area,omitempty" bson:"area,omitempty"`

	// Effect types (one or more)
	Damage      *SpellDamage     `json:"damage,omitempty" bson:"damage,omitempty"`
	Healing     *HealingEffect   `json:"healing,omitempty" bson:"healing,omitempty"`
	Condition   *ConditionEffect `json:"condition,omitempty" bson:"condition,omitempty"`
	SavingThrow *SavingThrowData `json:"savingThrow,omitempty" bson:"savingThrow,omitempty"`
	StatMod     *ModifierEffect  `json:"statMod,omitempty" bson:"statMod,omitempty"`
	Summon      *SummonEffect    `json:"summon,omitempty" bson:"summon,omitempty"`
}

// SpellEffectTrigger defines when a spell effect activates.
type SpellEffectTrigger string

const (
	TriggerOnCast       SpellEffectTrigger = "on_cast"
	TriggerOnHit        SpellEffectTrigger = "on_hit"
	TriggerOnFailedSave SpellEffectTrigger = "on_failed_save"
	TriggerStartOfTurn  SpellEffectTrigger = "start_of_turn"
	TriggerEndOfTurn    SpellEffectTrigger = "end_of_turn"
	TriggerOnEnter      SpellEffectTrigger = "on_enter" // entering spell area
	TriggerOnExit       SpellEffectTrigger = "on_exit"
)

// SpellTarget defines what a spell can target.
type SpellTarget struct {
	Type   SpellTargetType `json:"type" bson:"type"`
	Count  int             `json:"count,omitempty" bson:"count,omitempty"`   // number of targets
	Filter string          `json:"filter,omitempty" bson:"filter,omitempty"` // "creature", "humanoid", "ally"
}

// SpellTargetType categorizes spell targets.
type SpellTargetType string

const (
	TargetSelf      SpellTargetType = "self"
	TargetCreature  SpellTargetType = "creature"
	TargetCreatures SpellTargetType = "creatures"
	TargetPoint     SpellTargetType = "point"
	TargetObject    SpellTargetType = "object"
	TargetArea      SpellTargetType = "area"
	TargetWilling   SpellTargetType = "willing"
)

// SpellDamage describes damage dealt by a spell.
type SpellDamage struct {
	Base          DamageRoll  `json:"base" bson:"base"`
	PerLevel      *DamageRoll `json:"perLevel,omitempty" bson:"perLevel,omitempty"`           // additional per spell level
	CantripsScale bool        `json:"cantripsScale,omitempty" bson:"cantripsScale,omitempty"` // scales with caster level
}

// SummonEffect describes a summoning spell.
type SummonEffect struct {
	CreatureType string `json:"creatureType" bson:"creatureType"`       // "beast", "elemental", etc.
	CRMax        string `json:"crMax,omitempty" bson:"crMax,omitempty"` // max CR of summoned creature
	Count        string `json:"count" bson:"count"`                     // "1", "1d4+1", etc.
	Duration     string `json:"duration" bson:"duration"`
}
