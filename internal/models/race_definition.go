package models

// MechanicEffect represents an automatable game mechanic (discriminated union via Type field).
type MechanicEffect struct {
	Type         string `json:"type" bson:"type"` // "cantrip", "spell", "resistance", "advantage", "custom"
	SpellEngName string `json:"spellEngName,omitempty" bson:"spellEngName,omitempty"`
	Ability      string `json:"ability,omitempty" bson:"ability,omitempty"`
	MinLevel     int    `json:"minLevel,omitempty" bson:"minLevel,omitempty"`
	UsesPerDay   int    `json:"usesPerDay,omitempty" bson:"usesPerDay,omitempty"`
	DamageType   string `json:"damageType,omitempty" bson:"damageType,omitempty"`
	On           string `json:"on,omitempty" bson:"on,omitempty"`
	Effect       string `json:"effect,omitempty" bson:"effect,omitempty"`
}

// TraitDefinition represents a racial trait, subrace trait, or feat benefit feature.
type TraitDefinition struct {
	EngName     string          `json:"engName" bson:"engName"`
	Name        Name            `json:"name" bson:"name"`
	Description Name            `json:"description" bson:"description"`
	Mechanics   *MechanicEffect `json:"mechanics,omitempty" bson:"mechanics,omitempty"`
}

// RaceProficiencies holds optional proficiencies granted by a race or subrace.
type RaceProficiencies struct {
	Weapons []string     `json:"weapons,omitempty" bson:"weapons,omitempty"`
	Armor   []string     `json:"armor,omitempty" bson:"armor,omitempty"`
	Tools   []string     `json:"tools,omitempty" bson:"tools,omitempty"`
	Skills  *SkillChoice `json:"skills,omitempty" bson:"skills,omitempty"`
}

// AbilityBonusChoice represents a player's choice of ability score bonuses (e.g., Half-Elf).
type AbilityBonusChoice struct {
	Pick    int      `json:"pick" bson:"pick"`
	Amount  int      `json:"amount" bson:"amount"`
	Exclude []string `json:"exclude,omitempty" bson:"exclude,omitempty"`
}

// SubraceDefinition represents a subrace (e.g., High Elf, Hill Dwarf).
type SubraceDefinition struct {
	EngName        string            `json:"engName" bson:"engName"`
	Name           Name              `json:"name" bson:"name"`
	Description    Name              `json:"description" bson:"description"`
	AbilityBonuses map[string]int    `json:"abilityBonuses" bson:"abilityBonuses"`
	Features       []TraitDefinition `json:"features" bson:"features"`
	Proficiencies  *RaceProficiencies `json:"proficiencies,omitempty" bson:"proficiencies,omitempty"`
	Source         string            `json:"source" bson:"source"`
}

// RaceDefinition represents a D&D race (e.g., Elf, Human).
type RaceDefinition struct {
	EngName            string              `json:"engName" bson:"engName"`
	Name               Name                `json:"name" bson:"name"`
	Description        Name                `json:"description" bson:"description"`
	AbilityBonuses     map[string]int      `json:"abilityBonuses" bson:"abilityBonuses"`
	AbilityBonusChoice *AbilityBonusChoice `json:"abilityBonusChoice,omitempty" bson:"abilityBonusChoice,omitempty"`
	Size               string              `json:"size" bson:"size"`
	Speed              int                 `json:"speed" bson:"speed"`
	Darkvision         *int                `json:"darkvision,omitempty" bson:"darkvision,omitempty"`
	Languages          []string            `json:"languages" bson:"languages"`
	Proficiencies      *RaceProficiencies  `json:"proficiencies,omitempty" bson:"proficiencies,omitempty"`
	Features           []TraitDefinition   `json:"features" bson:"features"`
	Subraces           []SubraceDefinition `json:"subraces" bson:"subraces"`
	Source             string              `json:"source" bson:"source"`
	Tags               []string            `json:"tags" bson:"tags"`
	SchemaVersion      int                 `json:"schemaVersion" bson:"schemaVersion"`
}

// RaceFilterParams holds query parameters for listing race definitions.
type RaceFilterParams struct {
	Search string
	Page   int
	Size   int
}

// RaceListResponse is the paginated response for race definitions.
type RaceListResponse struct {
	Races []*RaceDefinition `json:"races"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}
