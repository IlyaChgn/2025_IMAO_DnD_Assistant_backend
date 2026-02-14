package usecases

import (
	"context"
	"fmt"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	actionsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/compute"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type actionsUsecases struct {
	encounterRepo encounterinterfaces.EncounterRepository
	characterRepo characterinterfaces.CharacterBaseRepository
}

func NewActionsUsecases(
	encounterRepo encounterinterfaces.EncounterRepository,
	characterRepo characterinterfaces.CharacterBaseRepository,
) actionsinterfaces.ActionsUsecases {
	return &actionsUsecases{
		encounterRepo: encounterRepo,
		characterRepo: characterRepo,
	}
}

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
		return nil, fmt.Errorf("%w: %s", apperrors.EncounterNotFoundErr, err.Error())
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
	switch cmd.Type {
	case models.ActionCustomRoll:
		return resolveCustomRoll(cmd)

	case models.ActionAbilityCheck:
		return resolveAbilityCheck(cmd, charBase.Name, derived)

	case models.ActionSavingThrow:
		return resolveSavingThrow(cmd, charBase.Name, derived)

	case models.ActionWeaponAttack:
		return resolveWeaponAttack(ctx, uc, cmd, encounterID, charBase, derived, participant, ed, userID)

	case models.ActionSpellCast:
		return resolveSpellCast(ctx, uc, cmd, encounterID, charBase, derived, participant, ed, userID)

	case models.ActionUseFeature:
		return resolveUseFeature(ctx, uc, cmd, encounterID, charBase, derived, participant, ed, userID)

	default:
		return nil, apperrors.InvalidActionTypeErr
	}
}
