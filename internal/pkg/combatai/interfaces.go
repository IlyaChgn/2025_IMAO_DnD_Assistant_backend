package combatai

// CombatAI takes a combat state snapshot and returns a decision for one NPC.
// Pure function — no side effects, no DB access.
// Suitable for extraction into a gRPC service: input and output are fully serializable.
type CombatAI interface {
	DecideTurn(input *TurnInput) (*TurnDecision, error)
}
