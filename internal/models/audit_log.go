package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuditLogEntry represents a single action recorded in the encounter combat log.
// Append-only: entries are never updated or deleted (TTL handles cleanup).
type AuditLogEntry struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EncounterID string             `json:"encounterId" bson:"encounterId"`
	Round       int                `json:"round" bson:"round"`
	Turn        int                `json:"turn" bson:"turn"`
	ActorID     string             `json:"actorId" bson:"actorId"`
	ActorName   string             `json:"actorName" bson:"actorName"`
	ActionType  ActionType         `json:"actionType" bson:"actionType"`
	Summary     string             `json:"summary" bson:"summary"`

	RollResult       *ActionRollResult  `json:"rollResult,omitempty" bson:"rollResult,omitempty"`
	DamageRolls      []ActionRollResult `json:"damageRolls,omitempty" bson:"damageRolls,omitempty"`
	StateChanges     []StateChange      `json:"stateChanges,omitempty" bson:"stateChanges,omitempty"`
	ConditionApplied []ConditionApplied `json:"conditionApplied,omitempty" bson:"conditionApplied,omitempty"`
	Hit              *bool              `json:"hit,omitempty" bson:"hit,omitempty"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}
