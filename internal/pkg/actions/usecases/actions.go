package usecases

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/compute"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	spellsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells"
)

type actionsUsecases struct {
	encounterRepo encounterinterfaces.EncounterRepository
	characterRepo characterinterfaces.CharacterBaseRepository
	spellsRepo    spellsinterfaces.SpellsRepository
	bestiaryRepo  bestiaryinterfaces.BestiaryRepository
	auditLogRepo  actionsinterfaces.AuditLogRepository
}

func NewActionsUsecases(
	encounterRepo encounterinterfaces.EncounterRepository,
	characterRepo characterinterfaces.CharacterBaseRepository,
	spellsRepo spellsinterfaces.SpellsRepository,
	bestiaryRepo bestiaryinterfaces.BestiaryRepository,
	auditLogRepo actionsinterfaces.AuditLogRepository,
) actionsinterfaces.ActionsUsecases {
	return &actionsUsecases{
		encounterRepo: encounterRepo,
		characterRepo: characterRepo,
		spellsRepo:    spellsRepo,
		bestiaryRepo:  bestiaryRepo,
		auditLogRepo:  auditLogRepo,
	}
}

// ExecuteAction loads the encounter, validates permissions, computes derived
// stats, and dispatches to the appropriate resolver.
//
// NOTE: the read-modify-write cycle on encounter data is not protected by
// optimistic locking. Concurrent actions on the same encounter may cause lost
// updates. This will be addressed when WS-based sync is added (T33).
func (uc *actionsUsecases) ExecuteAction(
	ctx context.Context,
	encounterID string,
	req *models.ActionRequest,
	userID int,
) (*models.ActionResponse, error) {
	l := logger.FromContext(ctx)

	// 1. Validate required fields
	if req.CharacterID == "" {
		return nil, apperrors.MissingCharacterIDErr
	}

	// 2. Load encounter
	encounter, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, err
	}

	// 3. Permission check: DM (encounter owner) OR participant owner
	isDM := uc.encounterRepo.CheckPermission(ctx, encounterID, userID)

	// 4. Parse encounter data
	ed, err := ParseEncounterData(encounter.Data)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, fmt.Errorf("parse encounter data: %w", err)
	}

	// 5. Find participant
	participant, _, err := ed.FindParticipant(req.CharacterID)
	if err != nil {
		return nil, err
	}

	// If not DM, verify the caller owns this participant
	if !isDM {
		if participant.OwnerID != strconv.Itoa(userID) {
			return nil, apperrors.PermissionDeniedError
		}
	}

	// 6. Load character base
	charBase, err := uc.characterRepo.GetByID(ctx, req.CharacterID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"characterID": req.CharacterID})
		return nil, fmt.Errorf("load character: %w", err)
	}

	// 7. Compute derived stats
	derived := compute.ComputeDerived(charBase)

	// 8. Dispatch to resolver based on action type
	cmd := &req.Action
	var resp *models.ActionResponse

	switch cmd.Type {
	case models.ActionCustomRoll:
		resp, err = resolveCustomRoll(cmd)

	case models.ActionAbilityCheck:
		resp, err = resolveAbilityCheck(cmd, charBase.Name, derived)

	case models.ActionSavingThrow:
		resp, err = resolveSavingThrow(cmd, charBase.Name, derived)

	case models.ActionWeaponAttack:
		resp, err = resolveWeaponAttack(ctx, uc, cmd, encounterID, charBase, derived, participant, ed, userID)

	case models.ActionSpellCast:
		resp, err = resolveSpellCast(ctx, uc, cmd, encounterID, charBase, derived, participant, ed, userID)

	case models.ActionUseFeature:
		resp, err = resolveUseFeature(ctx, uc, cmd, encounterID, charBase, derived, participant, ed, userID)

	default:
		return nil, apperrors.InvalidActionTypeErr
	}

	if err != nil {
		return nil, err
	}

	// 9. Fire-and-forget audit log entry
	entry := &models.AuditLogEntry{
		EncounterID:      encounterID,
		Round:            ed.CurrentRound(),
		Turn:             ed.CurrentTurnIndex(),
		ActorID:          req.CharacterID,
		ActorName:        charBase.Name,
		ActionType:       cmd.Type,
		Summary:          resp.Summary,
		RollResult:       resp.RollResult,
		DamageRolls:      resp.DamageRolls,
		StateChanges:     resp.StateChanges,
		ConditionApplied: resp.ConditionApplied,
		Hit:              resp.Hit,
	}

	if insertErr := uc.auditLogRepo.Insert(ctx, entry); insertErr != nil {
		l.UsecasesWarn(insertErr, userID, map[string]any{
			"encounterID": encounterID,
			"action":      "audit_log_insert",
		})
	}

	return resp, nil
}

// GetActionLog retrieves the action log for an encounter.
func (uc *actionsUsecases) GetActionLog(
	ctx context.Context,
	encounterID string,
	userID int,
	limit int,
	before time.Time,
) ([]*models.AuditLogEntry, error) {
	l := logger.FromContext(ctx)

	if !uc.encounterRepo.CheckPermission(ctx, encounterID, userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"encounterID": encounterID})
		return nil, apperrors.PermissionDeniedError
	}

	entries, err := uc.auditLogRepo.GetByEncounterID(ctx, encounterID, limit, before)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
		return nil, err
	}

	return entries, nil
}
