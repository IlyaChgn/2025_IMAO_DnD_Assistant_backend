package combatai

// dodgeDecision returns an ActionDecision for the Dodge action.
// Dodge doesn't go through the action pipeline — it's handled
// directly by the usecases layer (PR-6). ActionType is left empty
// because Dodge doesn't map to any existing models.ActionType.
func dodgeDecision() *ActionDecision {
	return &ActionDecision{
		ActionID:   "dodge",
		ActionName: "Dodge",
		Reasoning:  "Low HP, no good attacks — Dodge for survival",
	}
}
