package combatai

import "context"

// CombatAI takes a combat state snapshot and returns a decision for one NPC.
// Pure function — no side effects, no DB access.
// Suitable for extraction into a gRPC service: input and output are fully serializable.
type CombatAI interface {
	DecideTurn(input *TurnInput) (*TurnDecision, error)
}

// CombatAIUsecases orchestrates AI turn execution: loads encounter data,
// calls CombatAI.DecideTurn, and executes the resulting action through the pipeline.
type CombatAIUsecases interface {
	ExecuteAITurn(ctx context.Context, encounterID string, npcInstanceID string, userID int) (*AITurnResult, error)
	ExecuteAIRound(ctx context.Context, encounterID string, userID int) (*AIRoundResult, error)
}
