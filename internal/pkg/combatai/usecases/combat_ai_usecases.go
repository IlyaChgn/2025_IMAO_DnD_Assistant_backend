package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
	actionsuc "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions/usecases"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/compute"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
)

type combatAIUsecases struct {
	ai            combatai.CombatAI
	encounterRepo encounterinterfaces.EncounterRepository
	bestiaryRepo  bestiaryinterfaces.BestiaryRepository
	characterRepo characterinterfaces.CharacterBaseRepository
	actionsUC     actionsinterfaces.ActionsUsecases
	auditLogRepo  actionsinterfaces.AuditLogRepository
	tableManager  tableinterfaces.TableManager
}

// NewCombatAIUsecases creates a CombatAIUsecases with all required dependencies.
func NewCombatAIUsecases(
	ai combatai.CombatAI,
	encounterRepo encounterinterfaces.EncounterRepository,
	bestiaryRepo bestiaryinterfaces.BestiaryRepository,
	characterRepo characterinterfaces.CharacterBaseRepository,
	actionsUC actionsinterfaces.ActionsUsecases,
	auditLogRepo actionsinterfaces.AuditLogRepository,
	tableManager tableinterfaces.TableManager,
) combatai.CombatAIUsecases {
	return &combatAIUsecases{
		ai:            ai,
		encounterRepo: encounterRepo,
		bestiaryRepo:  bestiaryRepo,
		characterRepo: characterRepo,
		actionsUC:     actionsUC,
		auditLogRepo:  auditLogRepo,
		tableManager:  tableManager,
	}
}

// ExecuteAITurn loads encounter data, runs the AI engine, and executes the
// resulting action through the action pipeline (or handles Dodge directly).
func (uc *combatAIUsecases) ExecuteAITurn(
	ctx context.Context,
	encounterID string,
	npcInstanceID string,
	userID int,
) (*combatai.AITurnResult, error) {
	l := logger.FromContext(ctx)

	// 1. Load encounter.
	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, err
	}

	// 2. Permission check — only encounter owner (DM) can trigger AI turns.
	if !uc.encounterRepo.CheckPermission(ctx, encounterID, userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"encounterID": encounterID})
		return nil, apperrors.PermissionDeniedError
	}

	// 3. Parse encounter data.
	ed, err := actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, fmt.Errorf("parse encounter data: %w", err)
	}

	// 4. Find active NPC.
	npc, _, err := ed.FindParticipantByInstanceID(npcInstanceID)
	if err != nil {
		return nil, apperrors.ParticipantNotFoundErr
	}

	// 5. Validate: must be NPC, must be alive.
	if npc.IsPlayerCharacter {
		return nil, apperrors.NPCIsPlayerCharacterErr
	}
	if combatai.GetCurrentHP(npc) <= 0 {
		return nil, apperrors.NPCIsDeadErr
	}

	// 6. Load creature template.
	creature, err := uc.bestiaryRepo.GetCreatureByID(ctx, npc.CreatureID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"creatureID": npc.CreatureID})
		return nil, fmt.Errorf("load creature: %w", err)
	}
	if creature == nil {
		return nil, fmt.Errorf("creature not found: %s", npc.CreatureID)
	}

	// 7. Build TurnInput.
	input, err := uc.buildTurnInput(ctx, ed, npc, creature, userID, nil)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcInstanceID})
		return nil, fmt.Errorf("build turn input: %w", err)
	}

	// 8. Call AI engine.
	decision, err := uc.ai.DecideTurn(input)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcInstanceID})
		return nil, fmt.Errorf("AI decide turn: %w", err)
	}

	// Dead NPC — should not happen since we checked HP above, but handle gracefully.
	if decision == nil {
		return &combatai.AITurnResult{
			NpcInstanceID: npcInstanceID,
			Decision:      &combatai.TurnDecision{Reasoning: "NPC is dead"},
		}, nil
	}

	// 9. Execute decision.
	actionResults, err := uc.executeTurn(ctx, encounterID, decision, npc, ed, creature, userID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcInstanceID})
		return nil, fmt.Errorf("execute turn: %w", err)
	}

	// 10. Broadcast updated state (best-effort).
	uc.broadcastEncounter(ctx, encounterID, userID)

	// 11. Return result.
	return &combatai.AITurnResult{
		NpcInstanceID: npcInstanceID,
		Decision:      decision,
		ActionResults: actionResults,
	}, nil
}

// buildTurnInput assembles a TurnInput from encounter data and DB lookups.
// recentTargets is nil for single-turn calls; non-nil during ExecuteAIRound for focus-fire.
func (uc *combatAIUsecases) buildTurnInput(
	ctx context.Context,
	ed *actionsuc.EncounterData,
	npc *models.ParticipantFull,
	creature *models.Creature,
	userID int,
	recentTargets map[string]string,
) (*combatai.TurnInput, error) {
	l := logger.FromContext(ctx)

	stats := make(map[string]combatai.CombatantStats, len(ed.Participants))

	for i := range ed.Participants {
		p := &ed.Participants[i]

		if combatai.GetCurrentHP(p) <= 0 {
			continue // skip dead participants
		}

		if p.IsPlayerCharacter {
			cs, err := uc.buildPCStats(ctx, p, userID)
			if err != nil {
				l.UsecasesWarn(err, userID, map[string]any{"participantID": p.InstanceID})
				continue // skip PC if we can't load stats
			}
			stats[p.InstanceID] = cs
		} else {
			cs, err := uc.buildNPCStats(ctx, p, creature, userID)
			if err != nil {
				l.UsecasesWarn(err, userID, map[string]any{"participantID": p.InstanceID})
				continue
			}
			stats[p.InstanceID] = cs
		}
	}

	intelligence := combatai.ComputeIntelligence(creature.Ability.Int, 0.0)

	input := &combatai.TurnInput{
		ActiveNPC:        *npc,
		CreatureTemplate: *creature,
		Participants:     ed.Participants,
		CurrentRound:     ed.CurrentRound(),
		CombatantStats:   stats,
		Intelligence:     intelligence,
		PreviousTargetID: "", // Phase 1: no sticky targeting persistence
		RecentNPCTargets: recentTargets,
	}

	// Parse walkability grid from encounter data.
	if walkRaw := ed.RawField("walkability"); walkRaw != nil {
		var walkGrid [][]int
		if err := json.Unmarshal(walkRaw, &walkGrid); err == nil && len(walkGrid) > 0 {
			input.WalkabilityGrid = convertWalkGrid(walkGrid)
			input.MapHeight = len(walkGrid)
			if len(walkGrid[0]) > 0 {
				input.MapWidth = len(walkGrid[0])
			}
		}
	}

	// Parse blocked edges from encounter data.
	if edgesRaw := ed.RawField("edges"); edgesRaw != nil {
		var edges []models.SerializedEdge
		if err := json.Unmarshal(edgesRaw, &edges); err == nil {
			input.BlockedEdges = parseBlockedEdges(edges)
		}
	}

	return input, nil
}

// buildPCStats loads a PC's CharacterBase and computes CombatantStats.
func (uc *combatAIUsecases) buildPCStats(
	ctx context.Context,
	p *models.ParticipantFull,
	userID int,
) (combatai.CombatantStats, error) {
	if p.CharacterRuntime == nil {
		return combatai.CombatantStats{IsPC: true}, nil
	}

	charBase, err := uc.characterRepo.GetByID(ctx, p.CharacterRuntime.CharacterID)
	if err != nil {
		return combatai.CombatantStats{}, fmt.Errorf("load character %s: %w", p.CharacterRuntime.CharacterID, err)
	}

	derived := compute.ComputeDerived(charBase)

	saveBonuses := make(map[string]int, len(derived.SaveBonuses))
	for ability, bonus := range derived.SaveBonuses {
		saveBonuses[strings.ToUpper(ability)] = bonus.Total
	}

	return combatai.CombatantStats{
		MaxHP:           derived.MaxHp,
		AC:              derived.ArmorClass,
		SaveBonuses:     saveBonuses,
		Resistances:     derived.Resistances,
		Immunities:      derived.Immunities,
		Vulnerabilities: derived.Vulnerabilities,
		IsPC:            true,
	}, nil
}

// buildNPCStats extracts CombatantStats from a creature template.
// If the participant is the active NPC, uses the already-loaded creature;
// otherwise loads from the bestiary.
func (uc *combatAIUsecases) buildNPCStats(
	ctx context.Context,
	p *models.ParticipantFull,
	activeCreature *models.Creature,
	userID int,
) (combatai.CombatantStats, error) {
	var creature *models.Creature

	if p.CreatureID == activeCreature.ID.Hex() {
		creature = activeCreature
	} else {
		var err error
		creature, err = uc.bestiaryRepo.GetCreatureByID(ctx, p.CreatureID)
		if err != nil {
			return combatai.CombatantStats{}, fmt.Errorf("load creature %s: %w", p.CreatureID, err)
		}
		if creature == nil {
			return combatai.CombatantStats{}, fmt.Errorf("creature not found: %s", p.CreatureID)
		}
	}

	saveBonuses := parseSaveBonuses(creature)

	return combatai.CombatantStats{
		MaxHP:           p.RuntimeState.MaxHP,
		AC:              creature.ArmorClass,
		SaveBonuses:     saveBonuses,
		Resistances:     creature.DamageResistances,
		Immunities:      creature.DamageImmunities,
		Vulnerabilities: creature.DamageVulnerabilities,
		IsPC:            false,
	}, nil
}

// executeTurn dispatches the AI decision to the action pipeline.
func (uc *combatAIUsecases) executeTurn(
	ctx context.Context,
	encounterID string,
	decision *combatai.TurnDecision,
	npc *models.ParticipantFull,
	ed *actionsuc.EncounterData,
	creature *models.Creature,
	userID int,
) ([]*models.ActionResponse, error) {
	l := logger.FromContext(ctx)
	var actionResults []*models.ActionResponse

	// Movement (before action — D&D 5e turn order).
	if decision.Movement != nil {
		if err := uc.executeMovement(ctx, encounterID, decision, npc, ed, creature, userID); err != nil {
			return nil, err
		}
		// Reload fresh data after position update.
		encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
		if err != nil {
			return nil, fmt.Errorf("reload encounter after movement: %w", err)
		}
		ed, err = actionsuc.ParseEncounterData(encounter.Data)
		if err != nil {
			return nil, fmt.Errorf("parse encounter data after movement: %w", err)
		}
		npc, _, err = ed.FindParticipantByInstanceID(npc.InstanceID)
		if err != nil {
			return nil, fmt.Errorf("find NPC after movement: %w", err)
		}
	}

	// Main action.
	if decision.Action != nil {
		switch decision.Action.ActionID {
		case "dodge":
			if err := uc.executeDodge(ctx, encounterID, npc, ed, creature, decision, userID); err != nil {
				return nil, err
			}
		case "dash":
			if err := uc.executeDash(ctx, encounterID, npc, ed, creature, decision, userID); err != nil {
				return nil, err
			}
		case "disengage":
			if err := uc.executeDisengage(ctx, encounterID, npc, ed, creature, decision, userID); err != nil {
				return nil, err
			}
		default:
			results, err := uc.executeSingleOrMultiattack(ctx, encounterID, npc.InstanceID, decision.Action, userID)
			if err != nil {
				return nil, err
			}
			actionResults = append(actionResults, results...)
		}
	}

	// Bonus action (independent of main action).
	if decision.BonusAction != nil {
		bonusResults, err := uc.executeSingleOrMultiattack(ctx, encounterID, npc.InstanceID, decision.BonusAction, userID)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{
				"encounterID": encounterID,
				"npcID":       npc.InstanceID,
				"action":      "bonus_action",
			})
		} else {
			actionResults = append(actionResults, bonusResults...)
		}

		// Mark bonus action as used (persist via encounter update).
		uc.markBonusActionUsed(ctx, encounterID, npc.InstanceID, userID)
	}

	return actionResults, nil
}

// executeSingleOrMultiattack dispatches a single ActionDecision through the action pipeline.
func (uc *combatAIUsecases) executeSingleOrMultiattack(
	ctx context.Context,
	encounterID string,
	npcInstanceID string,
	action *combatai.ActionDecision,
	userID int,
) ([]*models.ActionResponse, error) {
	if action.MultiattackSteps != nil {
		return uc.executeMultiattack(ctx, encounterID, npcInstanceID, action, userID)
	}

	req := toActionRequest(npcInstanceID, action, nil)
	resp, err := uc.actionsUC.ExecuteAction(ctx, encounterID, req, userID)
	if err != nil {
		return nil, fmt.Errorf("execute action %s: %w", action.ActionID, err)
	}

	return []*models.ActionResponse{resp}, nil
}

// markBonusActionUsed persists BonusActionUsed=true for the NPC.
func (uc *combatAIUsecases) markBonusActionUsed(
	ctx context.Context,
	encounterID string,
	npcInstanceID string,
	userID int,
) {
	l := logger.FromContext(ctx)

	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "mark_bonus_used"})
		return
	}

	ed, err := actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "mark_bonus_used"})
		return
	}

	npc, _, err := ed.FindParticipantByInstanceID(npcInstanceID)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcInstanceID})
		return
	}

	npc.RuntimeState.Resources.BonusActionUsed = true

	data, err := ed.Marshal()
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "mark_bonus_used"})
		return
	}

	if err := uc.encounterRepo.UpdateEncounter(ctx, data, encounterID); err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "mark_bonus_used"})
	}
}

// executeDodge applies the Dodge universal action directly (no action pipeline).
func (uc *combatAIUsecases) executeDodge(
	ctx context.Context,
	encounterID string,
	npc *models.ParticipantFull,
	ed *actionsuc.EncounterData,
	creature *models.Creature,
	decision *combatai.TurnDecision,
	userID int,
) error {
	l := logger.FromContext(ctx)

	// Add Dodge StatModifier to NPC's RuntimeState.
	dodgeMod := models.StatModifier{
		ID:       fmt.Sprintf("dodge-%s", npc.InstanceID),
		Name:     "Dodge",
		SourceID: npc.InstanceID,
		Modifiers: []models.ModifierEffect{
			{
				Target:    models.ModTargetAttackRolls,
				Operation: models.ModOpDisadvantage,
			},
		},
		Duration: models.DurationUntilTurn,
	}
	npc.RuntimeState.StatModifiers = append(npc.RuntimeState.StatModifiers, dodgeMod)

	// Persist updated encounter data.
	data, err := ed.Marshal()
	if err != nil {
		return fmt.Errorf("marshal encounter data: %w", err)
	}
	if err := uc.encounterRepo.UpdateEncounter(ctx, data, encounterID); err != nil {
		return fmt.Errorf("update encounter: %w", err)
	}

	// Audit log for Dodge.
	actorName := npcActorName(npc, creature)
	entry := &models.AuditLogEntry{
		EncounterID: encounterID,
		Round:       ed.CurrentRound(),
		Turn:        ed.CurrentTurnIndex(),
		ActorID:     npc.InstanceID,
		ActorName:   actorName,
		ActionType:  "dodge",
		Summary:     fmt.Sprintf("%s takes the Dodge action", actorName),
	}
	if insertErr := uc.auditLogRepo.Insert(ctx, entry); insertErr != nil {
		l.UsecasesWarn(insertErr, userID, map[string]any{
			"encounterID": encounterID,
			"action":      "audit_log_insert",
		})
	}

	return nil
}

// executeMultiattack executes each step of a multiattack sequence.
func (uc *combatAIUsecases) executeMultiattack(
	ctx context.Context,
	encounterID string,
	npcInstanceID string,
	action *combatai.ActionDecision,
	userID int,
) ([]*models.ActionResponse, error) {
	results := make([]*models.ActionResponse, 0, len(action.MultiattackSteps))

	for i := range action.MultiattackSteps {
		step := &action.MultiattackSteps[i]
		req := toActionRequest(npcInstanceID, action, step)

		resp, err := uc.actionsUC.ExecuteAction(ctx, encounterID, req, userID)
		if err != nil {
			return results, fmt.Errorf("multiattack step %d (%s): %w", i, step.ActionID, err)
		}

		results = append(results, resp)
	}

	return results, nil
}

// toActionRequest converts an AI ActionDecision (and optional multiattack step)
// into an ActionRequest for the action pipeline.
func toActionRequest(npcInstanceID string, action *combatai.ActionDecision, step *combatai.MultiattackStep) *models.ActionRequest {
	req := &models.ActionRequest{
		CharacterID: npcInstanceID,
	}

	if step != nil {
		req.Action = buildActionCommand(step.ActionType, step.ActionID, step.TargetIDs, 0)
		return req
	}

	req.Action = buildActionCommand(action.ActionType, action.ActionID, action.TargetIDs, action.SlotLevel)
	return req
}

// buildActionCommand creates an ActionCommand for the given action type.
func buildActionCommand(actionType models.ActionType, actionID string, targetIDs []string, slotLevel int) models.ActionCommand {
	cmd := models.ActionCommand{Type: actionType}

	switch actionType {
	case models.ActionWeaponAttack:
		cmd.WeaponID = actionID
		if len(targetIDs) > 0 {
			cmd.TargetID = targetIDs[0]
		}
	case models.ActionSpellCast:
		cmd.SpellID = actionID
		cmd.SlotLevel = slotLevel
		cmd.TargetIDs = targetIDs
	case models.ActionUseFeature:
		cmd.FeatureID = actionID
		if len(targetIDs) > 1 {
			cmd.TargetIDs = targetIDs // AoE: pass all targets
		} else if len(targetIDs) > 0 {
			cmd.TargetID = targetIDs[0]
		}
	}

	return cmd
}

// broadcastEncounter sends the updated encounter state to connected WS clients.
func (uc *combatAIUsecases) broadcastEncounter(ctx context.Context, encounterID string, userID int) {
	l := logger.FromContext(ctx)

	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "broadcast_load"})
		return
	}

	msg, err := json.Marshal(map[string]any{
		"type": "encounter_update",
		"data": json.RawMessage(encounter.Data),
	})
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "broadcast_marshal"})
		return
	}

	uc.tableManager.BroadcastToEncounter(ctx, encounterID, userID, msg)
}

// parseSaveBonuses extracts save bonuses from a creature's SavingThrows slice.
func parseSaveBonuses(creature *models.Creature) map[string]int {
	if len(creature.SavingThrows) == 0 {
		return nil
	}

	bonuses := make(map[string]int, len(creature.SavingThrows))
	for _, st := range creature.SavingThrows {
		key := strings.ToUpper(st.ShortName)
		if key == "" && len(st.Name) >= 3 {
			key = strings.ToUpper(st.Name[:3])
		}
		if key == "" {
			continue
		}

		// Value is interface{} — handle float64 (JSON), int32/int64 (BSON), int, string.
		switch v := st.Value.(type) {
		case float64:
			bonuses[key] = int(v)
		case int32:
			bonuses[key] = int(v)
		case int64:
			bonuses[key] = int(v)
		case int:
			bonuses[key] = v
		case string:
			s := strings.TrimPrefix(v, "+")
			if n, err := strconv.Atoi(s); err == nil {
				bonuses[key] = n
			}
		}
	}

	return bonuses
}

// npcActorName resolves a display name for an NPC (replicates actions/usecases logic).
func npcActorName(participant *models.ParticipantFull, creature *models.Creature) string {
	if participant.DisplayName != "" {
		return participant.DisplayName
	}
	if creature.Name.Eng != "" {
		return creature.Name.Eng
	}
	return participant.InstanceID
}

// ProcessMove handles server-validated movement and checks for NPC opportunity attacks.
// When a PC moves out of an NPC's melee reach, eligible NPCs may take opportunity attacks.
func (uc *combatAIUsecases) ProcessMove(
	ctx context.Context,
	encounterID string,
	req *combatai.MoveRequest,
	userID int,
) (*combatai.MoveResult, error) {
	l := logger.FromContext(ctx)

	// 1. Load encounter.
	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, err
	}

	// 2. Permission check.
	if !uc.encounterRepo.CheckPermission(ctx, encounterID, userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"encounterID": encounterID})
		return nil, apperrors.PermissionDeniedError
	}

	// 3. Parse encounter data.
	ed, err := actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, fmt.Errorf("parse encounter data: %w", err)
	}

	// 4. Find mover.
	mover, _, err := ed.FindParticipantByInstanceID(req.ParticipantID)
	if err != nil {
		return nil, apperrors.ParticipantNotFoundErr
	}

	// 5. Validate mover is alive.
	if combatai.GetCurrentHP(mover) <= 0 {
		return nil, apperrors.NPCIsDeadErr
	}

	oldPos := mover.CellsCoords

	result := &combatai.MoveResult{
		ParticipantID:   req.ParticipantID,
		OldPosition:     oldPos,
		NewPosition:     &req.NewPosition,
		MovementApplied: true,
	}

	// 6. Check for opportunity attacks (only when a PC moves — Phase 2 scope).
	if mover.IsPlayerCharacter && oldPos != nil {
		oaResults, moverDied := uc.executeOpportunityAttacks(
			ctx, encounterID, ed, mover, oldPos, &req.NewPosition, userID,
		)
		result.OpportunityAttacks = oaResults

		if moverDied {
			result.MovementApplied = false
			result.NewPosition = oldPos
			uc.broadcastMoveResult(ctx, encounterID, result)
			uc.broadcastEncounter(ctx, encounterID, userID)
			return result, nil
		}
	}

	// 7. Apply position update — reload fresh data after OA execution.
	encounter, err = uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID, "action": "reload_after_oa"})
		return nil, fmt.Errorf("reload encounter after OA: %w", err)
	}
	ed, err = actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		return nil, fmt.Errorf("parse encounter data after OA: %w", err)
	}
	mover, _, err = ed.FindParticipantByInstanceID(req.ParticipantID)
	if err != nil {
		return nil, fmt.Errorf("find mover after OA: %w", err)
	}

	mover.CellsCoords = &req.NewPosition

	data, err := ed.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal encounter data: %w", err)
	}
	if err := uc.encounterRepo.UpdateEncounter(ctx, data, encounterID); err != nil {
		return nil, fmt.Errorf("persist position update: %w", err)
	}

	// 8. Broadcast.
	uc.broadcastMoveResult(ctx, encounterID, result)
	uc.broadcastEncounter(ctx, encounterID, userID)

	return result, nil
}

// executeOpportunityAttacks finds eligible NPCs and executes their opportunity attacks.
// Returns the list of attack results and whether the mover died.
func (uc *combatAIUsecases) executeOpportunityAttacks(
	ctx context.Context,
	encounterID string,
	ed *actionsuc.EncounterData,
	mover *models.ParticipantFull,
	oldPos, newPos *models.CellsCoordinates,
	userID int,
) ([]combatai.OpportunityAttackResult, bool) {
	l := logger.FromContext(ctx)

	// Load creature templates for all NPCs.
	creatures := uc.loadNPCCreatures(ctx, ed.Participants, userID)

	// Find candidates (pure function).
	candidates := combatai.FindOpportunityAttackCandidates(
		ed.Participants, mover.InstanceID, oldPos, newPos, creatures,
	)

	if len(candidates) == 0 {
		return nil, false
	}

	var results []combatai.OpportunityAttackResult

	for _, cand := range candidates {
		// Intelligence gate.
		intelligence := combatai.ComputeIntelligence(cand.Creature.Ability.Int, 0.0)
		if !combatai.ShouldTakeOpportunityAttack(intelligence, nil) {
			results = append(results, combatai.OpportunityAttackResult{
				NpcID:      cand.NPC.InstanceID,
				NpcName:    npcActorName(cand.NPC, cand.Creature),
				Skipped:    true,
				SkipReason: fmt.Sprintf("Intelligence check failed (%.2f)", intelligence),
			})
			continue
		}

		// Build ActionDecision for the single melee attack.
		action := &combatai.ActionDecision{
			ActionType: models.ActionWeaponAttack,
			ActionID:   cand.Action.ID,
			ActionName: cand.Action.Name,
			TargetIDs:  []string{mover.InstanceID},
		}

		// Execute through existing action pipeline.
		actionResults, err := uc.executeSingleOrMultiattack(
			ctx, encounterID, cand.NPC.InstanceID, action, userID,
		)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{
				"encounterID": encounterID,
				"npcID":       cand.NPC.InstanceID,
				"action":      "opportunity_attack",
			})
			results = append(results, combatai.OpportunityAttackResult{
				NpcID:      cand.NPC.InstanceID,
				NpcName:    npcActorName(cand.NPC, cand.Creature),
				Skipped:    true,
				SkipReason: "Execution error",
			})
			continue
		}

		results = append(results, combatai.OpportunityAttackResult{
			NpcID:         cand.NPC.InstanceID,
			NpcName:       npcActorName(cand.NPC, cand.Creature),
			ActionID:      cand.Action.ID,
			ActionName:    cand.Action.Name,
			ActionResults: actionResults,
		})

		// Mark reaction used — reload fresh data since action pipeline persisted changes.
		uc.markReactionUsed(ctx, encounterID, cand.NPC.InstanceID, userID)

		// Check if mover died — reload fresh data.
		encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
		if err != nil {
			continue
		}
		freshEd, err := actionsuc.ParseEncounterData(encounter.Data)
		if err != nil {
			continue
		}
		freshMover, _, err := freshEd.FindParticipantByInstanceID(mover.InstanceID)
		if err != nil {
			continue
		}
		if combatai.GetCurrentHP(freshMover) <= 0 {
			return results, true
		}
	}

	return results, false
}

// loadNPCCreatures loads creature templates for all NPC participants.
// Caches by CreatureID to avoid duplicate loads.
func (uc *combatAIUsecases) loadNPCCreatures(
	ctx context.Context,
	participants []models.ParticipantFull,
	userID int,
) map[string]*models.Creature {
	l := logger.FromContext(ctx)
	creatures := make(map[string]*models.Creature)
	byCreatureID := make(map[string]*models.Creature)

	for i := range participants {
		p := &participants[i]
		if p.IsPlayerCharacter {
			continue
		}

		if c, ok := byCreatureID[p.CreatureID]; ok {
			creatures[p.InstanceID] = c
			continue
		}

		creature, err := uc.bestiaryRepo.GetCreatureByID(ctx, p.CreatureID)
		if err != nil || creature == nil {
			l.UsecasesWarn(err, userID, map[string]any{"creatureID": p.CreatureID})
			continue
		}

		byCreatureID[p.CreatureID] = creature
		creatures[p.InstanceID] = creature
	}

	return creatures
}

// markReactionUsed persists ReactionUsed=true for the NPC.
// Follows the same reload-mutate-persist pattern as markBonusActionUsed.
func (uc *combatAIUsecases) markReactionUsed(
	ctx context.Context,
	encounterID string,
	npcInstanceID string,
	userID int,
) {
	l := logger.FromContext(ctx)

	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "mark_reaction_used"})
		return
	}

	ed, err := actionsuc.ParseEncounterData(encounter.Data)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "mark_reaction_used"})
		return
	}

	npc, _, err := ed.FindParticipantByInstanceID(npcInstanceID)
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "npcID": npcInstanceID})
		return
	}

	npc.RuntimeState.Resources.ReactionUsed = true

	data, err := ed.Marshal()
	if err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "mark_reaction_used"})
		return
	}

	if err := uc.encounterRepo.UpdateEncounter(ctx, data, encounterID); err != nil {
		l.UsecasesWarn(err, userID, map[string]any{"encounterID": encounterID, "action": "mark_reaction_used"})
	}
}

// broadcastMoveResult sends the move result via WebSocket.
func (uc *combatAIUsecases) broadcastMoveResult(ctx context.Context, encounterID string, result *combatai.MoveResult) {
	l := logger.FromContext(ctx)

	msg, err := json.Marshal(models.WSResponse{
		Type: models.MoveResultMsg,
		Data: result,
	})
	if err != nil {
		l.UsecasesWarn(err, 0, map[string]any{"encounterID": encounterID, "action": "broadcast_move"})
		return
	}

	uc.tableManager.BroadcastToEncounter(ctx, encounterID, 0, msg)
}

// executeMovement updates the NPC's grid position and persists.
func (uc *combatAIUsecases) executeMovement(
	ctx context.Context,
	encounterID string,
	decision *combatai.TurnDecision,
	npc *models.ParticipantFull,
	ed *actionsuc.EncounterData,
	creature *models.Creature,
	userID int,
) error {
	l := logger.FromContext(ctx)

	npc.CellsCoords = &models.CellsCoordinates{
		CellsX: decision.Movement.TargetX,
		CellsY: decision.Movement.TargetY,
	}

	data, err := ed.Marshal()
	if err != nil {
		return fmt.Errorf("marshal encounter data: %w", err)
	}
	if err := uc.encounterRepo.UpdateEncounter(ctx, data, encounterID); err != nil {
		return fmt.Errorf("update encounter: %w", err)
	}

	// Audit log for movement.
	actorName := npcActorName(npc, creature)
	entry := &models.AuditLogEntry{
		EncounterID: encounterID,
		Round:       ed.CurrentRound(),
		Turn:        ed.CurrentTurnIndex(),
		ActorID:     npc.InstanceID,
		ActorName:   actorName,
		ActionType:  "movement",
		Summary:     fmt.Sprintf("%s moves to (%d,%d)", actorName, decision.Movement.TargetX, decision.Movement.TargetY),
	}
	if insertErr := uc.auditLogRepo.Insert(ctx, entry); insertErr != nil {
		l.UsecasesWarn(insertErr, userID, map[string]any{
			"encounterID": encounterID,
			"action":      "audit_log_insert",
		})
	}

	return nil
}

// executeDash handles the Dash action — audit log only (movement already applied).
func (uc *combatAIUsecases) executeDash(
	ctx context.Context,
	encounterID string,
	npc *models.ParticipantFull,
	ed *actionsuc.EncounterData,
	creature *models.Creature,
	decision *combatai.TurnDecision,
	userID int,
) error {
	l := logger.FromContext(ctx)

	actorName := npcActorName(npc, creature)
	entry := &models.AuditLogEntry{
		EncounterID: encounterID,
		Round:       ed.CurrentRound(),
		Turn:        ed.CurrentTurnIndex(),
		ActorID:     npc.InstanceID,
		ActorName:   actorName,
		ActionType:  "dash",
		Summary:     fmt.Sprintf("%s takes the Dash action", actorName),
	}
	if insertErr := uc.auditLogRepo.Insert(ctx, entry); insertErr != nil {
		l.UsecasesWarn(insertErr, userID, map[string]any{
			"encounterID": encounterID,
			"action":      "audit_log_insert",
		})
	}

	return nil
}

// executeDisengage handles the Disengage action — audit log only (movement already applied).
func (uc *combatAIUsecases) executeDisengage(
	ctx context.Context,
	encounterID string,
	npc *models.ParticipantFull,
	ed *actionsuc.EncounterData,
	creature *models.Creature,
	decision *combatai.TurnDecision,
	userID int,
) error {
	l := logger.FromContext(ctx)

	actorName := npcActorName(npc, creature)
	entry := &models.AuditLogEntry{
		EncounterID: encounterID,
		Round:       ed.CurrentRound(),
		Turn:        ed.CurrentTurnIndex(),
		ActorID:     npc.InstanceID,
		ActorName:   actorName,
		ActionType:  "disengage",
		Summary:     fmt.Sprintf("%s takes the Disengage action", actorName),
	}
	if insertErr := uc.auditLogRepo.Insert(ctx, entry); insertErr != nil {
		l.UsecasesWarn(insertErr, userID, map[string]any{
			"encounterID": encounterID,
			"action":      "audit_log_insert",
		})
	}

	return nil
}

// convertWalkGrid converts [][]int (1=passable, 0=blocked) to [][]bool.
func convertWalkGrid(grid [][]int) [][]bool {
	result := make([][]bool, len(grid))
	for i, row := range grid {
		result[i] = make([]bool, len(row))
		for j, val := range row {
			result[i][j] = val == 1
		}
	}
	return result
}

// parseBlockedEdges converts SerializedEdges with MoveBlock=true into
// the bidirectional map[[4]int]bool used by PathfindingParams.
// Edge key format: "x1,y1-x2,y2" (both directions set).
func parseBlockedEdges(edges []models.SerializedEdge) map[[4]int]bool {
	result := make(map[[4]int]bool)
	for _, e := range edges {
		if !e.MoveBlock {
			continue
		}
		parts := strings.SplitN(e.Key, "-", 2)
		if len(parts) != 2 {
			continue
		}
		from := strings.Split(parts[0], ",")
		to := strings.Split(parts[1], ",")
		if len(from) != 2 || len(to) != 2 {
			continue
		}
		x1, err1 := strconv.Atoi(strings.TrimSpace(from[0]))
		y1, err2 := strconv.Atoi(strings.TrimSpace(from[1]))
		x2, err3 := strconv.Atoi(strings.TrimSpace(to[0]))
		y2, err4 := strconv.Atoi(strings.TrimSpace(to[1]))
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			continue
		}
		// Bidirectional.
		result[[4]int{x1, y1, x2, y2}] = true
		result[[4]int{x2, y2, x1, y1}] = true
	}
	return result
}
