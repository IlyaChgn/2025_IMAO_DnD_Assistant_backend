package models

// SkillChoice represents a pick-N from a list of skills.
type SkillChoice struct {
	Pick int      `json:"pick" bson:"pick"`
	From []string `json:"from" bson:"from"`
}

// ToolChoice represents either a fixed set of tools or a pick-N choice.
type ToolChoice struct {
	Type  string   `json:"type" bson:"type"` // "fixed" or "pick"
	Tools []string `json:"tools,omitempty" bson:"tools,omitempty"`
	Pick  int      `json:"pick,omitempty" bson:"pick,omitempty"`
	From  []string `json:"from,omitempty" bson:"from,omitempty"`
}

// EquipmentItem represents a single item with quantity.
type EquipmentItem struct {
	EngName  string `json:"engName" bson:"engName"`
	Quantity int    `json:"quantity" bson:"quantity"`
}

// EquipmentChoice represents either fixed equipment or a choice between option sets.
type EquipmentChoice struct {
	Type    string            `json:"type" bson:"type"` // "fixed" or "choice"
	Items   []EquipmentItem   `json:"items,omitempty" bson:"items,omitempty"`
	Options [][]EquipmentItem `json:"options,omitempty" bson:"options,omitempty"`
}

// ClassProficiencies holds proficiencies granted at L1.
type ClassProficiencies struct {
	Armor        []string     `json:"armor" bson:"armor"`
	Weapons      []string     `json:"weapons" bson:"weapons"`
	Tools        []ToolChoice `json:"tools" bson:"tools"`
	SavingThrows []string     `json:"savingThrows" bson:"savingThrows"`
	Skills       SkillChoice  `json:"skills" bson:"skills"`
}

// RefSpellcastingConfig holds spellcasting data for a class or subclass.
type RefSpellcastingConfig struct {
	Ability              string  `json:"ability" bson:"ability"`
	CasterType           string  `json:"casterType" bson:"casterType"`                                       // "full", "half", "third", "pact"
	PreparationType      string  `json:"preparationType" bson:"preparationType"`                              // "known", "prepared", "spellbook"
	RitualCasting        string  `json:"ritualCasting" bson:"ritualCasting"`                                  // "from_prepared", "from_spellbook", "none"
	PreparedCountFormula *string `json:"preparedCountFormula,omitempty" bson:"preparedCountFormula,omitempty"` // "ability_mod_plus_level", etc.
	CastingStartLevel    int     `json:"castingStartLevel" bson:"castingStartLevel"`
	UsesFocus            bool    `json:"usesFocus" bson:"usesFocus"`
	CantripsKnownTable   []int   `json:"cantripsKnownTable" bson:"cantripsKnownTable"`
	SpellsKnownTable     []int   `json:"spellsKnownTable,omitempty" bson:"spellsKnownTable,omitempty"`
}

// ClassFeatureProgression lists feature engNames gained at a specific level.
type ClassFeatureProgression struct {
	Level    int      `json:"level" bson:"level"`
	Features []string `json:"features" bson:"features"`
}

// SubclassSpellEntry holds domain/oath/patron spells for a given class level.
type SubclassSpellEntry struct {
	ClassLevel int      `json:"classLevel" bson:"classLevel"`
	Spells     []string `json:"spells" bson:"spells"`
}

// SubclassDefinition represents a class subclass (e.g., Evocation, Berserker).
type SubclassDefinition struct {
	EngName      string                  `json:"engName" bson:"engName"`
	Name         Name                    `json:"name" bson:"name"`
	Description  Name                    `json:"description" bson:"description"`
	Features     []ClassFeatureProgression `json:"features" bson:"features"`
	Spellcasting *RefSpellcastingConfig  `json:"spellcasting,omitempty" bson:"spellcasting,omitempty"`
	SpellList    []SubclassSpellEntry    `json:"spellList,omitempty" bson:"spellList,omitempty"`
	Source       string                  `json:"source" bson:"source"`
}

// MulticlassRequirements holds the ability requirements and proficiencies gained.
type MulticlassRequirements struct {
	Abilities          map[string]int `json:"abilities" bson:"abilities"`
	ProficienciesGained []string      `json:"proficienciesGained" bson:"proficienciesGained"`
}

// ClassDefinition represents a D&D class (e.g., Wizard, Fighter).
type ClassDefinition struct {
	EngName                string                  `json:"engName" bson:"engName"`
	Name                   Name                    `json:"name" bson:"name"`
	Description            Name                    `json:"description" bson:"description"`
	HitDie                 string                  `json:"hitDie" bson:"hitDie"`
	Proficiencies          ClassProficiencies      `json:"proficiencies" bson:"proficiencies"`
	StartingEquipment      []EquipmentChoice       `json:"startingEquipment" bson:"startingEquipment"`
	Spellcasting           *RefSpellcastingConfig  `json:"spellcasting,omitempty" bson:"spellcasting,omitempty"`
	SubclassLevel          int                     `json:"subclassLevel" bson:"subclassLevel"`
	SubclassName           Name                    `json:"subclassName" bson:"subclassName"`
	Subclasses             []SubclassDefinition    `json:"subclasses" bson:"subclasses"`
	Features               []ClassFeatureProgression `json:"features" bson:"features"`
	MulticlassRequirements *MulticlassRequirements `json:"multiclassRequirements,omitempty" bson:"multiclassRequirements,omitempty"`
	Source                 string                  `json:"source" bson:"source"`
	Tags                   []string                `json:"tags" bson:"tags"`
	SchemaVersion          int                     `json:"schemaVersion" bson:"schemaVersion"`
}

// ClassFilterParams holds query parameters for listing class definitions.
type ClassFilterParams struct {
	Search string
	Page   int
	Size   int
}

// ClassListResponse is the paginated response for class definitions.
type ClassListResponse struct {
	Classes []*ClassDefinition `json:"classes"`
	Total   int64              `json:"total"`
	Page    int                `json:"page"`
	Size    int                `json:"size"`
}
