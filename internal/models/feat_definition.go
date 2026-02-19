package models

// FeatPrerequisites holds the requirements for taking a feat.
type FeatPrerequisites struct {
	Abilities    map[string]int `json:"abilities,omitempty" bson:"abilities,omitempty"`
	Proficiencies []string      `json:"proficiencies,omitempty" bson:"proficiencies,omitempty"`
	Spellcasting *bool          `json:"spellcasting,omitempty" bson:"spellcasting,omitempty"`
	Race         []string       `json:"race,omitempty" bson:"race,omitempty"`
}

// AbilityChoice represents a pick-N ability score increase choice.
type AbilityChoice struct {
	Pick   int      `json:"pick" bson:"pick"`
	From   []string `json:"from" bson:"from"`
	Amount int      `json:"amount" bson:"amount"`
}

// FeatBenefits holds the benefits granted by a feat.
type FeatBenefits struct {
	AbilityIncrease map[string]int    `json:"abilityIncrease,omitempty" bson:"abilityIncrease,omitempty"`
	AbilityChoice   *AbilityChoice    `json:"abilityChoice,omitempty" bson:"abilityChoice,omitempty"`
	Proficiencies   []string          `json:"proficiencies,omitempty" bson:"proficiencies,omitempty"`
	Features        []TraitDefinition `json:"features" bson:"features"`
}

// FeatDefinition represents a D&D feat (e.g., Alert, Great Weapon Master).
type FeatDefinition struct {
	EngName       string             `json:"engName" bson:"engName"`
	Name          Name               `json:"name" bson:"name"`
	Description   Name               `json:"description" bson:"description"`
	Prerequisites *FeatPrerequisites `json:"prerequisites,omitempty" bson:"prerequisites,omitempty"`
	Benefits      FeatBenefits       `json:"benefits" bson:"benefits"`
	Source        string             `json:"source" bson:"source"`
	Tags          []string           `json:"tags" bson:"tags"`
	SchemaVersion int                `json:"schemaVersion" bson:"schemaVersion"`
}

// FeatFilterParams holds query parameters for listing feat definitions.
type FeatFilterParams struct {
	Search string
	Page   int
	Size   int
}

// FeatListResponse is the paginated response for feat definitions.
type FeatListResponse struct {
	Feats []*FeatDefinition `json:"feats"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}
