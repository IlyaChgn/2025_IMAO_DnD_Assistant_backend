package models

// LanguageChoice represents either fixed languages or a pick-N choice.
type LanguageChoice struct {
	Type      string   `json:"type" bson:"type"` // "fixed" or "pick"
	Languages []string `json:"languages,omitempty" bson:"languages,omitempty"`
	Pick      int      `json:"pick,omitempty" bson:"pick,omitempty"`
}

// BackgroundFeature represents the unique feature granted by a background.
type BackgroundFeature struct {
	Name        Name `json:"name" bson:"name"`
	Description Name `json:"description" bson:"description"`
}

// SuggestedCharacteristics holds personality traits, ideals, bonds, and flaws.
type SuggestedCharacteristics struct {
	PersonalityTraits []Name `json:"personalityTraits" bson:"personalityTraits"`
	Ideals            []Name `json:"ideals" bson:"ideals"`
	Bonds             []Name `json:"bonds" bson:"bonds"`
	Flaws             []Name `json:"flaws" bson:"flaws"`
}

// BackgroundDefinition represents a D&D background (e.g., Acolyte, Soldier).
type BackgroundDefinition struct {
	EngName                  string                    `json:"engName" bson:"engName"`
	Name                     Name                      `json:"name" bson:"name"`
	Description              Name                      `json:"description" bson:"description"`
	SkillProficiencies       []string                  `json:"skillProficiencies" bson:"skillProficiencies"`
	ToolProficiencies        []string                  `json:"toolProficiencies" bson:"toolProficiencies"`
	Languages                LanguageChoice            `json:"languages" bson:"languages"`
	Equipment                []EquipmentItem           `json:"equipment" bson:"equipment"`
	Feature                  BackgroundFeature         `json:"feature" bson:"feature"`
	SuggestedCharacteristics *SuggestedCharacteristics `json:"suggestedCharacteristics,omitempty" bson:"suggestedCharacteristics,omitempty"`
	Source                   string                    `json:"source" bson:"source"`
	Tags                     []string                  `json:"tags" bson:"tags"`
	SchemaVersion            int                       `json:"schemaVersion" bson:"schemaVersion"`
}

// BackgroundFilterParams holds query parameters for listing background definitions.
type BackgroundFilterParams struct {
	Search string
	Page   int
	Size   int
}

// BackgroundListResponse is the paginated response for background definitions.
type BackgroundListResponse struct {
	Backgrounds []*BackgroundDefinition `json:"backgrounds"`
	Total       int64                   `json:"total"`
	Page        int                     `json:"page"`
	Size        int                     `json:"size"`
}
