package models

type SearchParams struct {
	Value string `json:"value"`
	Exact bool   `json:"exact"`
}

type Order struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type FilterParams struct {
	Book                []string `json:"book"`
	Npc                 []string `json:"npc"`
	ChallengeRating     []string `json:"challengeRating"`
	Type                []string `json:"type"`
	Size                []string `json:"size"`
	Tag                 []string `json:"tag"`
	Moving              []string `json:"moving"`
	Senses              []string `json:"senses"`
	VulnerabilityDamage []string `json:"vulnerabilityDamage"`
	ResistanceDamage    []string `json:"resistanceDamage"`
	ImmunityDamage      []string `json:"immunityDamage"`
	ImmunityCondition   []string `json:"immunityCondition"`
	Features            []string `json:"features"`
	Environment         []string `json:"environment"`
}
