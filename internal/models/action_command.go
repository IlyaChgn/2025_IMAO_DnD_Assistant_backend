package models

// ActionType identifies which kind of action a player is executing.
type ActionType string

const (
	ActionCustomRoll   ActionType = "custom_roll"
	ActionAbilityCheck ActionType = "ability_check"
	ActionSavingThrow  ActionType = "saving_throw"
	ActionWeaponAttack ActionType = "weapon_attack"
	ActionSpellCast    ActionType = "spell_cast"
	ActionUseFeature   ActionType = "use_feature"
)

// ActionCommand is a flat discriminated union: the Type field determines which
// other fields are relevant.
type ActionCommand struct {
	Type         ActionType `json:"type"`
	Advantage    bool       `json:"advantage,omitempty"`
	Disadvantage bool       `json:"disadvantage,omitempty"`

	// custom_roll
	Dice     string `json:"dice,omitempty"`
	Modifier int    `json:"modifier,omitempty"`
	Label    string `json:"label,omitempty"`

	// ability_check / saving_throw
	Ability string `json:"ability,omitempty"`
	Skill   string `json:"skill,omitempty"` // ability_check only
	DC      int    `json:"dc,omitempty"`    // saving_throw only

	// weapon_attack
	WeaponID string `json:"weaponId,omitempty"`
	TargetID string `json:"targetId,omitempty"`

	// spell_cast
	SpellID   string `json:"spellId,omitempty"`
	SlotLevel int    `json:"slotLevel,omitempty"`

	// use_feature
	FeatureID string `json:"featureId,omitempty"`
}

// ActionRequest is the top-level request body for POST /api/encounter/{id}/actions.
type ActionRequest struct {
	CharacterID string        `json:"characterId"`
	Action      ActionCommand `json:"action"`
}

// ActionRollResult describes a single dice roll outcome.
type ActionRollResult struct {
	Expression      string `json:"expression"`
	Rolls           []int  `json:"rolls"`
	Modifier        int    `json:"modifier"`
	Total           int    `json:"total"`
	Natural         int    `json:"natural,omitempty"`
	DamageType      string `json:"damageType,omitempty"`
	AppliedModifier string `json:"appliedModifier,omitempty"` // "normal"|"resistance"|"vulnerability"|"immunity"
	FinalDamage     *int   `json:"finalDamage,omitempty"`
}

// StateChange describes a mutation applied to a participant's runtime state.
type StateChange struct {
	TargetID    string `json:"targetId,omitempty"`
	HPDelta     int    `json:"hpDelta,omitempty"`
	SlotSpent   int    `json:"slotSpent,omitempty"`
	FeatureUsed string `json:"featureUsed,omitempty"`
	Description string `json:"description"`
}

// ActionResponse is returned by the action execution endpoint.
type ActionResponse struct {
	RollResult   *ActionRollResult  `json:"rollResult,omitempty"`
	DamageRolls  []ActionRollResult `json:"damageRolls,omitempty"`
	StateChanges []StateChange      `json:"stateChanges,omitempty"`
	Summary      string             `json:"summary"`
	Hit          *bool              `json:"hit,omitempty"`
}
