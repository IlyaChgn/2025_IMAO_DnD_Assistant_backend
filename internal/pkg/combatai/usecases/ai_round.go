package usecases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	actionsuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions/usecases"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

// ExecuteAIRound executes all NPC turns in initiative order for the current round.
func (uc *combatAIUsecases) ExecuteAIRound(
	ctx context.Context,
	encounterID string,
	userID int,
) (*combatai.AIRoundResult, error) {
	l := logger.FromContext(ctx)

	// 1. Load encounter + permission check.
	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, err
	}

	if !uc.encounterRepo.CheckPermission(ctx, encounterID, userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"encounterID": encounterID})
		return nil, apperrors.PermissionDeniedError
	}

	// 2. Parse encounter data and build turn order.
	ed, err := actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, fmt.Errorf("parse encounter data: %w", err)
	}

	npcOrder := initTurnOrder(ed.Participants)

	result := &combatai.AIRoundResult{
		Round: ed.CurrentRound(),
		Turns: make([]combatai.AIRoundTurn, 0, len(npcOrder)),
	}

	// 3. Execute each NPC turn.
	for _, npcID := range npcOrder {
		turn, ended, combatResult, err := uc.executeOneNPCTurn(ctx, encounterID, npcID, userID)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcID})
			// Non-fatal: skip this NPC and continue with the rest.
			// Don't expose internal error details in the API response.
			result.Turns = append(result.Turns, combatai.AIRoundTurn{
				NpcID:      npcID,
				Skipped:    true,
				SkipReason: "Internal error",
			})
			continue
		}

		result.Turns = append(result.Turns, *turn)

		// Broadcast per-NPC result via WS so frontend can animate each turn.
		uc.broadcastAITurnResult(ctx, encounterID, turn)

		// Check combat end.
		if ended {
			result.CombatEnded = true
			result.CombatResult = combatResult
			break
		}
	}

	// 4. Final broadcast of encounter state.
	uc.broadcastEncounter(ctx, encounterID, userID)

	return result, nil
}

// executeOneNPCTurn handles a single NPC's turn within ai-round.
// Returns the turn result and whether combat has ended.
func (uc *combatAIUsecases) executeOneNPCTurn(
	ctx context.Context,
	encounterID string,
	npcInstanceID string,
	userID int,
) (*combatai.AIRoundTurn, bool, string, error) {
	l := logger.FromContext(ctx)

	// Reload encounter (previous NPC's actions may have changed state).
	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		return nil, false, "", fmt.Errorf("reload encounter: %w", err)
	}

	ed, err := actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		return nil, false, "", fmt.Errorf("parse encounter data: %w", err)
	}

	// Find NPC in fresh data.
	npc, _, err := ed.FindParticipantByInstanceID(npcInstanceID)
	if err != nil {
		return nil, false, "", fmt.Errorf("find NPC %s: %w", npcInstanceID, err)
	}

	// Skip dead NPCs.
	if combatai.GetCurrentHP(npc) <= 0 {
		return &combatai.AIRoundTurn{
			NpcID:      npcInstanceID,
			NpcName:    npc.DisplayName,
			Skipped:    true,
			SkipReason: "Dead (HP <= 0)",
		}, false, "", nil
	}

	// Skip incapacitated NPCs.
	if combatai.IsIncapacitated(npc) {
		return &combatai.AIRoundTurn{
			NpcID:      npcInstanceID,
			NpcName:    npc.DisplayName,
			Skipped:    true,
			SkipReason: "Incapacitated",
		}, false, "", nil
	}

	// Load creature template.
	creature, err := uc.bestiaryRepo.GetCreatureByID(ctx, npc.CreatureID)
	if err != nil {
		return nil, false, "", fmt.Errorf("load creature %s: %w", npc.CreatureID, err)
	}
	if creature == nil {
		return nil, false, "", fmt.Errorf("creature not found: %s", npc.CreatureID)
	}

	// Process start of turn (recharge, reaction reset, legendary restore, conditions).
	if processStartOfTurn(npc, creature) {
		data, marshalErr := ed.Marshal()
		if marshalErr != nil {
			return nil, false, "", fmt.Errorf("marshal after processStartOfTurn: %w", marshalErr)
		}
		if updateErr := uc.encounterRepo.UpdateEncounter(ctx, data, encounterID); updateErr != nil {
			return nil, false, "", fmt.Errorf("persist processStartOfTurn: %w", updateErr)
		}
	}

	// Build TurnInput and run AI decision.
	input, err := uc.buildTurnInput(ctx, ed, npc, creature, userID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcInstanceID})
		return nil, false, "", fmt.Errorf("build turn input: %w", err)
	}

	decision, err := uc.ai.DecideTurn(input)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcInstanceID})
		return nil, false, "", fmt.Errorf("AI decide turn: %w", err)
	}

	// Handle nil decision (dead — shouldn't happen since we checked HP above).
	if decision == nil {
		return &combatai.AIRoundTurn{
			NpcID:   npcInstanceID,
			NpcName: npcActorName(npc, creature),
			Decision: &combatai.TurnDecision{Reasoning: "NPC is dead"},
			Skipped: true, SkipReason: "Dead",
		}, false, "", nil
	}

	// Execute the decision through the action pipeline.
	actionResults, err := uc.executeTurn(ctx, encounterID, decision, npc, ed, creature, userID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcInstanceID})
		return nil, false, "", fmt.Errorf("execute turn: %w", err)
	}

	turn := &combatai.AIRoundTurn{
		NpcID:         npcInstanceID,
		NpcName:       npcActorName(npc, creature),
		Decision:      decision,
		ActionResults: actionResults,
	}

	// Check combat end after this NPC's turn.
	encounter, err = uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		// Non-fatal for combat end check.
		return turn, false, "", nil
	}
	ed, err = actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		return turn, false, "", nil
	}
	ended, combatResult := checkCombatEnd(ed.Participants, ed.CurrentRound())

	return turn, ended, combatResult, nil
}

// broadcastAITurnResult sends a per-NPC turn result via WebSocket.
func (uc *combatAIUsecases) broadcastAITurnResult(ctx context.Context, encounterID string, turn *combatai.AIRoundTurn) {
	l := logger.FromContext(ctx)

	msg, err := json.Marshal(models.WSResponse{
		Type: models.AITurnResultMsg,
		Data: turn,
	})
	if err != nil {
		l.UsecasesWarn(err, 0, map[string]any{"encounterID": encounterID, "action": "broadcast_ai_turn"})
		return
	}

	// senderUserID = 0 ensures all connected clients receive the message.
	uc.tableManager.BroadcastToEncounter(ctx, encounterID, 0, msg)
}
