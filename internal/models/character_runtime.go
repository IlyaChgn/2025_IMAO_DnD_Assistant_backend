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

	CurrentHP   int               `json:"currentHp" bson:"currentHp"`
	TemporaryHP int               `json:"temporaryHp,omitempty" bson:"temporaryHp,omitempty"`
	Initiative  int               `json:"initiative" bson:"initiative"`
	Position    *CellsCoordinates `json:"position,omitempty" bson:"position,omitempty"`

	SpentSpellSlots map[int]int    `json:"spentSpellSlots,omitempty" bson:"spentSpellSlots,omitempty"`
	SpentPactSlots  int            `json:"spentPactSlots,omitempty" bson:"spentPactSlots,omitempty"`
	UsedFeatures    map[string]int `json:"usedFeatures,omitempty" bson:"usedFeatures,omitempty"`

	Conditions    []ActiveCondition       `json:"conditions,omitempty" bson:"conditions,omitempty"` // persistent only
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
