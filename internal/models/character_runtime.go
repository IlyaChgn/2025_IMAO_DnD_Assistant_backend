package models

// CharacterRuntime — ephemeral per-encounter state for a player character.
// Embedded in ParticipantFull when IsPlayerCharacter is true.
//
// Authority split (§ 4.7):
//
//	CharacterRuntime owns: HP, spell slots, feature uses, concentration,
//	                       persistent conditions, death saves, inspiration
//	ParticipantFull owns:  initiative order, grid position, combat conditions
type CharacterRuntime struct {
	CharacterID string `json:"characterId" bson:"characterId"` // FK → CharacterBase._id
	EncounterID string `json:"encounterId" bson:"encounterId"` // FK → Encounter.uuid

	CurrentHP   int `json:"currentHp" bson:"currentHp"`
	TemporaryHP int `json:"temporaryHp,omitempty" bson:"temporaryHp,omitempty"`

	SpentSpellSlots map[int]int    `json:"spentSpellSlots,omitempty" bson:"spentSpellSlots,omitempty"`
	SpentPactSlots  int            `json:"spentPactSlots,omitempty" bson:"spentPactSlots,omitempty"`
	UsedFeatures    map[string]int `json:"usedFeatures,omitempty" bson:"usedFeatures,omitempty"`

	// Persistent conditions only (e.g. exhaustion between sessions).
	// Combat conditions (paralyzed, stunned, etc.) live on Creature.activeConditions
	// via creature.slice — see condition-tracker-system-plan § 5.1.
	Conditions    []ConditionInstance     `json:"conditions,omitempty" bson:"conditions,omitempty"`
	Concentration *CharacterConcentration `json:"concentration,omitempty" bson:"concentration,omitempty"`
	DeathSaves    *DeathSaveState         `json:"deathSaves,omitempty" bson:"deathSaves,omitempty"`
	Inspiration   bool                    `json:"inspiration,omitempty" bson:"inspiration,omitempty"`
}

// CharacterConcentration — simpler than creature ConcentrationState.
// Matches frontend CharacterRuntimeSpellcasting.concentration shape.
type CharacterConcentration struct {
	SpellID      string `json:"spellId" bson:"spellId"`
	SpellName    string `json:"spellName" bson:"spellName"`
	StartedRound int    `json:"startedRound" bson:"startedRound"`
}

// ConditionInstance represents a runtime condition instance on a character or creature.
// Canonical type per condition-tracker-system-plan § 3.2.
type ConditionInstance struct {
	ID               string            `json:"id" bson:"id"`
	Type             ConditionType     `json:"type" bson:"type"`
	Level            int               `json:"level,omitempty" bson:"level,omitempty"`                       // exhaustion 1-6
	SourceCreatureID string            `json:"sourceCreatureId,omitempty" bson:"sourceCreatureId,omitempty"` // who applied it
	SourceName       string            `json:"sourceName,omitempty" bson:"sourceName,omitempty"`             // spell/ability name
	AppliedOnRound   int               `json:"appliedOnRound" bson:"appliedOnRound"`
	Duration         ConditionDuration `json:"duration" bson:"duration"`
	SaveRetry        *SaveRetry        `json:"saveRetry,omitempty" bson:"saveRetry,omitempty"`
	ExpiresOnRound   int               `json:"expiresOnRound,omitempty" bson:"expiresOnRound,omitempty"`
	Notes            string            `json:"notes,omitempty" bson:"notes,omitempty"`
}

// ConditionDuration describes how long a condition lasts.
// Frontend discriminated union modeled as flat struct with Type discriminator.
//
// Type values: "permanent", "rounds", "end_of_next_turn", "start_of_next_turn",
// "concentration", "until_removed", "until_long_rest", "until_short_rest".
type ConditionDuration struct {
	Type      string `json:"type" bson:"type"`
	Remaining int    `json:"remaining,omitempty" bson:"remaining,omitempty"` // for "rounds"
	CasterID  string `json:"casterId,omitempty" bson:"casterId,omitempty"`   // for "concentration"
}

// SaveRetry describes a repeating save to end a condition.
// See condition-tracker-system-plan § 3.2.
type SaveRetry struct {
	Timing           string `json:"timing" bson:"timing"` // "end_of_turn", "start_of_turn", "when_damaged"
	DC               int    `json:"dc" bson:"dc"`
	Ability          string `json:"ability" bson:"ability"`                                     // "str", "dex", "con", "int", "wis", "cha"
	SuccessesNeeded  int    `json:"successesNeeded,omitempty" bson:"successesNeeded,omitempty"` // usually 1; diseases may require 3
	CurrentSuccesses int    `json:"currentSuccesses,omitempty" bson:"currentSuccesses,omitempty"`
}
