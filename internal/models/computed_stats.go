package models

// ACBreakdownEntry describes one contributor to AC calculation.
type ACBreakdownEntry struct {
	Source    string  `json:"source"`
	Operation string  `json:"operation"`
	Value     float64 `json:"value"`
}

// ComputedCharacterStats holds the computed stats derived from equipped items.
type ComputedCharacterStats struct {
	AC                int                `json:"ac"`
	ACBreakdown       []ACBreakdownEntry `json:"acBreakdown"`
	TotalWeight       float64            `json:"totalWeight"`
	CarryingCapacity  float64            `json:"carryingCapacity"`
	Encumbered        bool               `json:"encumbered"`
	HeavilyEncumbered bool               `json:"heavilyEncumbered"`
	Resistances       []string           `json:"resistances"`
	Immunities        []string           `json:"immunities"`
	Vulnerabilities   []string           `json:"vulnerabilities"`
}
