package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// CharacterBase is the domain-shaped persistent model for a player character.
// Designed for automation: DC calculation, spell validation, attack resolution.
// Replaces the LSS form-shaped Character model for new characters.
type CharacterBase struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID  string             `json:"userId" bson:"userId"`
	Version int                `json:"version" bson:"version"` // optimistic locking

	// Identity
	Name       string       `json:"name" bson:"name"`
	Race       string       `json:"race" bson:"race"`
	Classes    []ClassEntry `json:"classes" bson:"classes"`
	Background string       `json:"background,omitempty" bson:"background,omitempty"`
	Alignment  string       `json:"alignment,omitempty" bson:"alignment,omitempty"`
	Experience int          `json:"experience,omitempty" bson:"experience,omitempty"`
	Edition    string       `json:"edition,omitempty" bson:"edition,omitempty"` // "2014" | "2024"

	// Appearance
	Appearance Appearance       `json:"appearance,omitempty" bson:"appearance,omitempty"`
	Avatar     *CharacterAvatar `json:"avatar,omitempty" bson:"avatar,omitempty"`

	// Core stats (raw scores, modifiers are derived)
	AbilityScores AbilityScores `json:"abilityScores" bson:"abilityScores"`

	// Proficiencies
	Proficiencies Proficiencies `json:"proficiencies" bson:"proficiencies"`

	// Expertise (skills with double proficiency bonus)
	Expertise []string `json:"expertise,omitempty" bson:"expertise,omitempty"`

	// Hit points
	HitPoints HitPointsBase `json:"hitPoints" bson:"hitPoints"`

	// Speed (base, before modifiers)
	BaseSpeed int `json:"baseSpeed" bson:"baseSpeed"`

	// Armor Class override (until equipment system is ready)
	ArmorClassOverride *int `json:"armorClassOverride,omitempty" bson:"armorClassOverride,omitempty"`

	// Weapons
	Weapons []WeaponDef `json:"weapons,omitempty" bson:"weapons,omitempty"`

	// Spellcasting (optional — non-casters omit)
	Spellcasting *CharacterSpellcasting `json:"spellcasting,omitempty" bson:"spellcasting,omitempty"`

	// Currency
	Coins Coins `json:"coins" bson:"coins"`

	// Narrative text (plain strings, converted from Tiptap on import)
	PersonalityTraits string `json:"personalityTraits,omitempty" bson:"personalityTraits,omitempty"`
	Ideals            string `json:"ideals,omitempty" bson:"ideals,omitempty"`
	Bonds             string `json:"bonds,omitempty" bson:"bonds,omitempty"`
	Flaws             string `json:"flaws,omitempty" bson:"flaws,omitempty"`
	Backstory         string `json:"backstory,omitempty" bson:"backstory,omitempty"`
	Notes             string `json:"notes,omitempty" bson:"notes,omitempty"` // catch-all for unconverted text

	// Features (structured, with optional resource tracking)
	Features []FeatureInstance `json:"features,omitempty" bson:"features,omitempty"`

	// Equipment (placeholder until inventory system)
	EquippedItems *EquippedSlots `json:"equippedItems,omitempty" bson:"equippedItems,omitempty"`

	// LSS import metadata (preserved for re-import / debugging)
	ImportSource *ImportSource `json:"importSource,omitempty" bson:"importSource,omitempty"`

	// Timestamps
	CreatedAt string `json:"createdAt" bson:"createdAt"`
	UpdatedAt string `json:"updatedAt" bson:"updatedAt"`
}

// ClassEntry represents a single class in a character's class list (supports multiclass).
type ClassEntry struct {
	ClassName string `json:"className" bson:"className"`                    // "wizard", "paladin", or custom
	Subclass  string `json:"subclass,omitempty" bson:"subclass,omitempty"` // "evocation", "oath_of_devotion"
	Level     int    `json:"level" bson:"level"`
}

// AbilityScores stores the six raw ability scores. Modifiers are derived: floor((score-10)/2).
type AbilityScores struct {
	Str int `json:"str" bson:"str"`
	Dex int `json:"dex" bson:"dex"`
	Con int `json:"con" bson:"con"`
	Int int `json:"int" bson:"int"`
	Wis int `json:"wis" bson:"wis"`
	Cha int `json:"cha" bson:"cha"`
}

// Proficiencies stores a character's proficiency lists.
type Proficiencies struct {
	Skills       []string      `json:"skills" bson:"skills"`                                   // ["athletics", "perception"]
	SavingThrows []AbilityType `json:"savingThrows" bson:"savingThrows"`                       // ["WIS", "CHA"] — uses AbilityType
	Tools        []string      `json:"tools,omitempty" bson:"tools,omitempty"`                 // ["thieves_tools"]
	Languages    []string      `json:"languages,omitempty" bson:"languages,omitempty"`         // ["common", "elvish"]
	Armor        []string      `json:"armor,omitempty" bson:"armor,omitempty"`                 // ["light", "medium", "shields"]
	Weapons      []string      `json:"weapons,omitempty" bson:"weapons,omitempty"`             // ["simple", "martial"]
}

// Appearance stores physical description fields.
type Appearance struct {
	Age    string `json:"age,omitempty" bson:"age,omitempty"`
	Height string `json:"height,omitempty" bson:"height,omitempty"`
	Weight string `json:"weight,omitempty" bson:"weight,omitempty"`
	Eyes   string `json:"eyes,omitempty" bson:"eyes,omitempty"`
	Skin   string `json:"skin,omitempty" bson:"skin,omitempty"`
	Hair   string `json:"hair,omitempty" bson:"hair,omitempty"`
}

// HitPointsBase stores base HP data. Current/temp HP are tracked at runtime.
type HitPointsBase struct {
	MaxOverride  *int   `json:"maxOverride,omitempty" bson:"maxOverride,omitempty"` // manual max HP
	HitDiceRolls []int  `json:"hitDiceRolls,omitempty" bson:"hitDiceRolls,omitempty"`
	HitDie       string `json:"hitDie,omitempty" bson:"hitDie,omitempty"` // "d10", "d8"
}

// WeaponDef describes a weapon in the character's inventory.
// Range reuses RangeData from action_structured.go (same package).
type WeaponDef struct {
	ID              string     `json:"id" bson:"id"`
	Name            string     `json:"name" bson:"name"`
	AttackType      string     `json:"attackType" bson:"attackType"`                                 // "melee", "ranged", "melee_or_ranged"
	AbilityOverride string     `json:"abilityOverride,omitempty" bson:"abilityOverride,omitempty"`   // "dex" for finesse
	DamageDice      string     `json:"damageDice" bson:"damageDice"`                                 // "1d8"
	DamageType      string     `json:"damageType" bson:"damageType"`                                 // "slashing"
	Properties      []string   `json:"properties,omitempty" bson:"properties,omitempty"`
	MagicBonus      int        `json:"magicBonus,omitempty" bson:"magicBonus,omitempty"`
	Range           *RangeData `json:"range,omitempty" bson:"range,omitempty"` // reuses existing RangeData
	Reach           int        `json:"reach,omitempty" bson:"reach,omitempty"`
}

// CharacterAvatar is defined in character.go (shared with legacy Character model).

// CharacterSpellcasting stores a player character's spellcasting data.
// Distinct from Spellcasting (creatures) — characters track spell lists, not just slots.
type CharacterSpellcasting struct {
	Ability        AbilityType `json:"ability" bson:"ability"` // "INT", "WIS", "CHA" — uses AbilityType constants
	CantripsKnown  []SpellRef  `json:"cantripsKnown,omitempty" bson:"cantripsKnown,omitempty"`
	SpellsKnown    []SpellRef  `json:"spellsKnown,omitempty" bson:"spellsKnown,omitempty"`
	Spellbook      []SpellRef  `json:"spellbook,omitempty" bson:"spellbook,omitempty"`
	PreparedSpells []SpellRef  `json:"preparedSpells,omitempty" bson:"preparedSpells,omitempty"`
	AlwaysPrepared []SpellRef  `json:"alwaysPrepared,omitempty" bson:"alwaysPrepared,omitempty"`
	// Imported spell text: raw text extracted from LSS Tiptap, stored temporarily.
	// Lifecycle: populated on import -> user links spells to SpellRefs -> spellTexts entries removed.
	SpellTexts map[int]string `json:"spellTexts,omitempty" bson:"spellTexts,omitempty"` // level -> plain text list
}

// SpellRef references a spell in a character's spell list.
// Name is denormalized (always present) so the UI can render without a SpellDefinition lookup.
// SpellID is optional: empty on LSS import, filled when user links to a SpellDefinition.
type SpellRef struct {
	SpellID  string `json:"spellId,omitempty" bson:"spellId,omitempty"` // ref to SpellDefinition.engName
	Name     string `json:"name" bson:"name"`                           // display name (always present)
	Prepared bool   `json:"prepared" bson:"prepared"`
	Source   string `json:"source,omitempty" bson:"source,omitempty"` // "class", "subclass", "race", "feat", "item"
}

// FeatureInstance represents a character feature (class feature, racial trait, feat, etc.).
type FeatureInstance struct {
	ID           string `json:"id" bson:"id"`
	FeatureDefID string `json:"featureDefId,omitempty" bson:"featureDefId,omitempty"` // ref to future FeatureDefinition
	Name         string `json:"name" bson:"name"`
	Description  string `json:"description,omitempty" bson:"description,omitempty"`
	Source       string `json:"source" bson:"source"`                                 // "class", "race", "feat", "background", "custom"
	SourceDetail string `json:"sourceDetail,omitempty" bson:"sourceDetail,omitempty"` // e.g. "wizard:evocation"
	Resource     *FeatureResource `json:"resource,omitempty" bson:"resource,omitempty"`
	// Automation fields (empty on LSS import, filled when user configures or FeatureDef is linked).
	// PassiveModifiers reuses ModifierEffect from creature_runtime.go (same package).
	PassiveModifiers []ModifierEffect    `json:"passiveModifiers,omitempty" bson:"passiveModifiers,omitempty"`
	ActiveAction     *CharacterActionDef `json:"activeAction,omitempty" bson:"activeAction,omitempty"`
}

// FeatureResource tracks limited-use feature resources (e.g. 3x/long rest).
type FeatureResource struct {
	MaxUses int    `json:"maxUses" bson:"maxUses"`
	ResetOn string `json:"resetOn" bson:"resetOn"` // "short_rest", "long_rest", "dawn", "never"
}

// CharacterActionDef describes an active action granted by a character feature.
// Named CharacterActionDef to avoid collision with existing StructuredAction (creature-focused).
type CharacterActionDef struct {
	Category    ActionCategory   `json:"category" bson:"category"` // "action", "bonus_action", "reaction"
	Attack      *AttackRollData  `json:"attack,omitempty" bson:"attack,omitempty"`
	SavingThrow *SavingThrowData `json:"savingThrow,omitempty" bson:"savingThrow,omitempty"`
	Healing     *HealingEffect   `json:"healing,omitempty" bson:"healing,omitempty"`
	Uses        *UsesData        `json:"uses,omitempty" bson:"uses,omitempty"`
	Effects     []ActionEffect   `json:"effects,omitempty" bson:"effects,omitempty"`
}

// EquippedSlots tracks which items are equipped (placeholder until inventory system).
type EquippedSlots struct {
	Head      string `json:"head,omitempty" bson:"head,omitempty"`
	Neck      string `json:"neck,omitempty" bson:"neck,omitempty"`
	Shoulders string `json:"shoulders,omitempty" bson:"shoulders,omitempty"`
	Chest     string `json:"chest,omitempty" bson:"chest,omitempty"`
	Hands     string `json:"hands,omitempty" bson:"hands,omitempty"`
	Waist     string `json:"waist,omitempty" bson:"waist,omitempty"`
	Legs      string `json:"legs,omitempty" bson:"legs,omitempty"`
	Feet      string `json:"feet,omitempty" bson:"feet,omitempty"`
	Ring1     string `json:"ring1,omitempty" bson:"ring1,omitempty"`
	Ring2     string `json:"ring2,omitempty" bson:"ring2,omitempty"`
	MainHand  string `json:"mainHand,omitempty" bson:"mainHand,omitempty"`
	OffHand   string `json:"offHand,omitempty" bson:"offHand,omitempty"`
}

// Coins stores currency in copper/silver/electrum/gold/platinum.
type Coins struct {
	Cp int `json:"cp" bson:"cp"`
	Sp int `json:"sp" bson:"sp"`
	Ep int `json:"ep" bson:"ep"`
	Gp int `json:"gp" bson:"gp"`
	Pp int `json:"pp" bson:"pp"`
}

// ImportSource records where a character was imported from.
type ImportSource struct {
	Format     string   `json:"format" bson:"format"`         // "lss_v2"
	ImportedAt string   `json:"importedAt" bson:"importedAt"`
	Warnings   []string `json:"warnings,omitempty" bson:"warnings,omitempty"`
}

// ConversionReport summarizes the results of an LSS -> CharacterBase conversion.
type ConversionReport struct {
	CharacterName string              `json:"characterName"`
	Success       bool                `json:"success"`
	FieldsCopied  int                 `json:"fieldsCopied"`  // high-confidence direct copies
	FieldsParsed  int                 `json:"fieldsParsed"`  // medium-confidence parsed fields
	FieldsSkipped int                 `json:"fieldsSkipped"` // could not convert
	Warnings      []ConversionWarning `json:"warnings"`
}

// ConversionWarning describes a single issue encountered during conversion.
type ConversionWarning struct {
	Field   string `json:"field"`   // "info.charClass"
	Message string `json:"message"` // "Unknown class 'Литера', stored as custom"
	Level   string `json:"level"`   // "info", "warning", "error"
}
