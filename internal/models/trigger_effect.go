package models

// TriggerEvent represents the game event that activates an item trigger.
type TriggerEvent string

const (
	ItemTriggerOnEquip      TriggerEvent = "on_equip"
	ItemTriggerOnUnequip    TriggerEvent = "on_unequip"
	ItemTriggerOnHit        TriggerEvent = "on_hit"
	ItemTriggerOnCritical   TriggerEvent = "on_critical"
	ItemTriggerOnTakeDamage TriggerEvent = "on_take_damage"
	ItemTriggerOnTurnStart  TriggerEvent = "on_turn_start"
	ItemTriggerOnTurnEnd    TriggerEvent = "on_turn_end"
	ItemTriggerOnUse        TriggerEvent = "on_use"
	ItemTriggerPassive      TriggerEvent = "passive"
)

// EffectType represents the kind of effect produced by a trigger.
type EffectType string

const (
	EffectDealDamage      EffectType = "deal_damage"
	EffectHeal            EffectType = "heal"
	EffectApplyCondition  EffectType = "apply_condition"
	EffectRemoveCondition EffectType = "remove_condition"
	EffectGrantTempHP     EffectType = "grant_temp_hp"
)

// Effect describes what happens when a trigger fires.
type Effect struct {
	Type   EffectType             `json:"type" bson:"type"`
	Params map[string]interface{} `json:"params" bson:"params"`
}

// TriggerEffect pairs a game event with an effect and optional gating.
type TriggerEffect struct {
	Trigger   TriggerEvent `json:"trigger" bson:"trigger"`
	Chance    float32      `json:"chance" bson:"chance"`                         // 0.0-1.0; 0 treated as 1.0 (always fires)
	Cooldown  string       `json:"cooldown,omitempty" bson:"cooldown,omitempty"` // e.g. "1/turn", "1/short_rest"
	Effect    Effect       `json:"effect" bson:"effect"`
	Condition string       `json:"condition,omitempty" bson:"condition,omitempty"` // future: pre-condition expression
}

// CooldownState tracks which cooldown keys have been used.
// Checked by engine, managed/persisted by T42.
type CooldownState map[string]bool

// TriggerResult is returned from the engine for each evaluated trigger.
type TriggerResult struct {
	Event       TriggerEvent `json:"triggerEvent"`
	EffectType  EffectType   `json:"effectType"`
	Description string       `json:"description"`
	Skipped     bool         `json:"skipped,omitempty"`
	SkipReason  string       `json:"skipReason,omitempty"` // "chance" or "cooldown"

	// Exactly one of these is set (based on EffectType):
	DamageResult    *TriggerDamageResult    `json:"damageResult,omitempty"`
	HealResult      *TriggerHealResult      `json:"healResult,omitempty"`
	ConditionResult *TriggerConditionResult `json:"conditionResult,omitempty"`
	TempHPResult    *TriggerTempHPResult    `json:"tempHPResult,omitempty"`
}

// TriggerDamageResult holds dice info for deal_damage effects.
type TriggerDamageResult struct {
	Dice       string `json:"dice"`
	DamageType string `json:"damageType"`
	Rolls      []int  `json:"rolls"`
	Total      int    `json:"total"`
}

// TriggerHealResult holds dice info for heal effects.
type TriggerHealResult struct {
	Dice  string `json:"dice,omitempty"`
	Rolls []int  `json:"rolls,omitempty"`
	Total int    `json:"total"`
}

// TriggerConditionResult holds condition application/removal info.
type TriggerConditionResult struct {
	Action     string   `json:"action"`               // "apply" or "remove"
	Condition  string   `json:"condition,omitempty"`  // for apply_condition
	Conditions []string `json:"conditions,omitempty"` // for remove_condition
	Duration   string   `json:"duration,omitempty"`   // for apply_condition
}

// TriggerTempHPResult holds temp HP grant info.
type TriggerTempHPResult struct {
	Amount int    `json:"amount"`
	Dice   string `json:"dice,omitempty"`
	Rolls  []int  `json:"rolls,omitempty"`
}
