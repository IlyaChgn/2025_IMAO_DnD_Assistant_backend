package models

// ConditionDefinition is a static reference entry for a D&D 5e condition.
// Served from an in-memory slice (no MongoDB).
type ConditionDefinition struct {
	Type        ConditionType    `json:"type"`
	Name        Name             `json:"name"`
	Description Name             `json:"description"`
	Effects     ConditionEffects `json:"effects"`
	HasLevels   bool             `json:"hasLevels,omitempty"`
	MaxLevel    int              `json:"maxLevel,omitempty"`
	Implies     []ConditionType  `json:"implies,omitempty"`
}

// ConditionEffects describes the mechanical effects of a condition (SRD 5.1).
type ConditionEffects struct {
	Speed         string            `json:"speed,omitempty"`
	CanAct        *bool             `json:"canAct,omitempty"`
	CanReact      *bool             `json:"canReact,omitempty"`
	CanMove       *bool             `json:"canMove,omitempty"`
	AttackRolls   string            `json:"attackRolls,omitempty"`
	BeingAttacked string            `json:"beingAttacked,omitempty"`
	AbilityChecks string            `json:"abilityChecks,omitempty"`
	SavingThrows  map[string]string `json:"savingThrows,omitempty"`
	MeleeCrits    bool              `json:"meleeCrits,omitempty"`
	DropsItems    bool              `json:"dropsItems,omitempty"`
	FallsProne    bool              `json:"fallsProne,omitempty"`
}
