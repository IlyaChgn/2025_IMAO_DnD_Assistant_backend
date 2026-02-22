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
	input, err := uc.buildTurnInput(ctx, ed, npc, creature, userID)
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
func (uc *combatAIUsecases) buildTurnInput(
	ctx context.Context,
	ed *actionsuc.EncounterData,
	npc *models.ParticipantFull,
	creature *models.Creature,
	userID int,
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

	return &combatai.TurnInput{
		ActiveNPC:        *npc,
		CreatureTemplate: *creature,
		Participants:     ed.Participants,
		CurrentRound:     ed.CurrentRound(),
		CombatantStats:   stats,
		Intelligence:     intelligence,
		PreviousTargetID: "", // Phase 1: no sticky targeting persistence
	}, nil
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
	// No action → skip turn.
	if decision.Action == nil {
		return nil, nil
	}

	// Dodge special path — no action pipeline.
	if decision.Action.ActionID == "dodge" {
		return nil, uc.executeDodge(ctx, encounterID, npc, ed, creature, decision, userID)
	}

	// Multiattack path.
	if decision.Action.MultiattackSteps != nil {
		return uc.executeMultiattack(ctx, encounterID, npc.InstanceID, decision.Action, userID)
	}

	// Single action path.
	req := toActionRequest(npc.InstanceID, decision.Action, nil)
	resp, err := uc.actionsUC.ExecuteAction(ctx, encounterID, req, userID)
	if err != nil {
		return nil, fmt.Errorf("execute action %s: %w", decision.Action.ActionID, err)
	}

	return []*models.ActionResponse{resp}, nil
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
		if len(targetIDs) > 0 {
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
