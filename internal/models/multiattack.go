package models

// MultiattackGroup represents a set of attacks a creature makes as a single action.
// For example, "The dragon makes three attacks: one with its bite and two with its claws."
type MultiattackGroup struct {
	ID      string             `json:"id" bson:"id"`
	Name    string             `json:"name" bson:"name"`
	Actions []MultiattackEntry `json:"actions" bson:"actions"`
}

// MultiattackEntry references a StructuredAction by ID and how many times it is used.
type MultiattackEntry struct {
	ActionID string `json:"actionId" bson:"actionId"` // → StructuredAction.ID
	Count    int    `json:"count" bson:"count"`
}
