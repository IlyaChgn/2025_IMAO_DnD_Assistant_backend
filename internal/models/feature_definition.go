package models

// FeatureDefinition is a reference-data entry for a class feature, racial trait, feat, etc.
// Characters link to these via FeatureInstance.FeatureDefID.
type FeatureDefinition struct {
	ID               string              `json:"id" bson:"_id,omitempty"`
	EngName          string              `json:"engName" bson:"engName"`
	Name             Name                `json:"name" bson:"name"`
	Description      Name                `json:"description" bson:"description"`
	Source           string              `json:"source" bson:"source"`                                   // "class", "race", "feat", "background"
	SourceDetail     string              `json:"sourceDetail,omitempty" bson:"sourceDetail,omitempty"`   // "fighter", "paladin:oath_of_devotion"
	Level            int                 `json:"level,omitempty" bson:"level,omitempty"`                 // class level requirement (0 = always available)
	Resource         *FeatureResource    `json:"resource,omitempty" bson:"resource,omitempty"`
	PassiveModifiers []ModifierEffect    `json:"passiveModifiers,omitempty" bson:"passiveModifiers,omitempty"`
	ActiveAction     *CharacterActionDef `json:"activeAction,omitempty" bson:"activeAction,omitempty"`
	Prerequisites    []string            `json:"prerequisites,omitempty" bson:"prerequisites,omitempty"`
	Tags             []string            `json:"tags,omitempty" bson:"tags,omitempty"`
	SchemaVersion    int                 `json:"schemaVersion" bson:"schemaVersion"`
}

// FeatureFilterParams holds query parameters for listing feature definitions.
type FeatureFilterParams struct {
	Source string // "class", "race", "feat"
	Class  string // "fighter", "paladin"
	Level  *int
	Search string
	Page   int
	Size   int
}

// FeatureListResponse is the paginated response for feature definitions.
type FeatureListResponse struct {
	Features []*FeatureDefinition `json:"features"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	Size     int                  `json:"size"`
}
