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
	// Track NPC target choices for focus-fire coordination.
	recentTargets := make(map[string]string)

	for _, npcID := range npcOrder {
		turn, ended, combatResult, err := uc.executeOneNPCTurn(ctx, encounterID, npcID, userID, recentTargets)
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

		// Record this NPC's primary target for subsequent NPCs' focus fire.
		if target := extractPrimaryTarget(turn); target != "" {
			recentTargets[npcID] = target
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

		// Legendary action phase — after this NPC's turn, other NPCs
		// with remaining legendary actions get to use one (D&D 5e).
		legendaryResults := uc.executeLegendaryActions(ctx, encounterID, npcID, npcOrder, userID, recentTargets)
		if len(legendaryResults) > 0 {
			result.Turns[len(result.Turns)-1].LegendaryActionResults = legendaryResults
			uc.broadcastLegendaryResults(ctx, encounterID, legendaryResults)
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
	recentTargets map[string]string,
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
	input, err := uc.buildTurnInput(ctx, ed, npc, creature, userID, recentTargets)
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
			NpcID:    npcInstanceID,
			NpcName:  npcActorName(npc, creature),
			Decision: &combatai.TurnDecision{Reasoning: "NPC is dead"},
			Skipped:  true, SkipReason: "Dead",
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

// executeLegendaryActions lets NPCs (other than the one who just acted)
// spend a legendary action after each turn. Per D&D 5e: legendary actions
// can only be used at the end of another creature's turn.
func (uc *combatAIUsecases) executeLegendaryActions(
	ctx context.Context,
	encounterID string,
	justActedNpcID string,
	npcOrder []string,
	userID int,
	recentTargets map[string]string,
) []*combatai.LegendaryActionResult {
	l := logger.FromContext(ctx)
	var results []*combatai.LegendaryActionResult

	// Reload encounter with fresh state after the main turn.
	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "legendary_reload"})
		return nil
	}

	ed, err := actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "legendary_parse"})
		return nil
	}

	for _, npcID := range npcOrder {
		if npcID == justActedNpcID {
			continue // can't use legendary actions after own turn
		}

		npc, _, err := ed.FindParticipantByInstanceID(npcID)
		if err != nil || combatai.GetCurrentHP(npc) <= 0 {
			continue
		}
		if npc.RuntimeState.Resources.LegendaryActions <= 0 {
			continue
		}

		creature, err := uc.bestiaryRepo.GetCreatureByID(ctx, npc.CreatureID)
		if err != nil || creature == nil {
			continue
		}

		input, err := uc.buildTurnInput(ctx, ed, npc, creature, userID, recentTargets)
		if err != nil {
			continue
		}

		decision := combatai.SelectLegendaryAction(input, nil)
		if decision == nil {
			continue
		}

		// Execute the legendary action through the action pipeline.
		actionResults, err := uc.executeSingleOrMultiattack(ctx, encounterID, npc.InstanceID, decision, userID)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{
				"encounterID": encounterID,
				"npcID":       npcID,
				"action":      "legendary_execute",
			})
			continue
		}

		// Reload encounter data — the action pipeline persisted its own changes
		// (damage, conditions, etc.). We must not overwrite them with stale data.
		encounter, err = uc.encounterRepo.GetEncounterByID(ctx, encounterID)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "legendary_reload_post"})
			continue
		}
		ed, err = actionsuc.ParseEncounterData(encounter.Data)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "legendary_parse_post"})
			continue
		}

		// Re-find NPC in fresh data to deduct legendary cost.
		npc, _, err = ed.FindParticipantByInstanceID(npcID)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcID})
			continue
		}

		// Deduct legendary action cost and persist.
		npc.RuntimeState.Resources.LegendaryActions -= decision.LegendaryCost
		data, err := ed.Marshal()
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "legendary_marshal"})
			continue
		}
		if err := uc.encounterRepo.UpdateEncounter(ctx, data, encounterID); err != nil {
			l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "legendary_persist"})
			continue
		}

		results = append(results, &combatai.LegendaryActionResult{
			NpcID:         npcID,
			NpcName:       npcActorName(npc, creature),
			Decision:      decision,
			ActionResults: actionResults,
		})

		// Record legendary action target for focus-fire coordination.
		if target := extractActionTarget(decision); target != "" {
			recentTargets[npcID] = target
		}
	}

	return results
}

// broadcastLegendaryResults sends legendary action results via WebSocket.
func (uc *combatAIUsecases) broadcastLegendaryResults(ctx context.Context, encounterID string, results []*combatai.LegendaryActionResult) {
	l := logger.FromContext(ctx)

	msg, err := json.Marshal(models.WSResponse{
		Type: models.AILegendaryResultMsg,
		Data: results,
	})
	if err != nil {
		l.UsecasesWarn(err, 0, map[string]any{"encounterID": encounterID, "action": "broadcast_legendary"})
		return
	}

	uc.tableManager.BroadcastToEncounter(ctx, encounterID, 0, msg)
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

// extractPrimaryTarget returns the first target instanceID from a turn result.
// Used for focus-fire coordination: tracks what each NPC targeted this round.
// Falls back to bonus action target if the main action has no target (e.g. Dodge + bonus attack).
// Returns "" for skip/dodge/dash/disengage turns (no offensive target).
func extractPrimaryTarget(turn *combatai.AIRoundTurn) string {
	if turn == nil || turn.Decision == nil {
		return ""
	}
	if target := extractActionTarget(turn.Decision.Action); target != "" {
		return target
	}
	return extractActionTarget(turn.Decision.BonusAction)
}

// extractActionTarget returns the first target instanceID from an ActionDecision.
// Handles both single actions (TargetIDs) and multiattack steps.
func extractActionTarget(action *combatai.ActionDecision) string {
	if action == nil {
		return ""
	}
	if len(action.TargetIDs) > 0 {
		return action.TargetIDs[0]
	}
	for _, step := range action.MultiattackSteps {
		if len(step.TargetIDs) > 0 {
			return step.TargetIDs[0]
		}
	}
	return ""
}
