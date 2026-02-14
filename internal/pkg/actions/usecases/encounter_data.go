package usecases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
)

// EncounterData wraps encounter blob parsing, giving access to participants
// while preserving all other fields (map, fog, walkability, etc.) as raw bytes.
type EncounterData struct {
	raw          map[string]json.RawMessage
	Participants []models.ParticipantFull
}

// ParseEncounterData unmarshals an encounter data blob. Only the "participants"
// key is fully parsed; all other keys are kept as raw JSON.
func ParseEncounterData(data json.RawMessage) (*EncounterData, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal encounter data: %w", err)
	}

	ed := &EncounterData{raw: raw}

	participantsRaw, ok := raw["participants"]
	if !ok {
		return ed, nil
	}

	if err := json.Unmarshal(participantsRaw, &ed.Participants); err != nil {
		return nil, fmt.Errorf("unmarshal participants: %w", err)
	}

	return ed, nil
}

// FindParticipant searches for a participant whose CharacterRuntime.CharacterID
// matches the given characterID. Returns the participant and its index, or an
// error if not found.
func (ed *EncounterData) FindParticipant(characterID string) (*models.ParticipantFull, int, error) {
	for i := range ed.Participants {
		p := &ed.Participants[i]
		if p.CharacterRuntime != nil && p.CharacterRuntime.CharacterID == characterID {
			return p, i, nil
		}
	}

	return nil, -1, apperrors.ParticipantNotFoundErr
}

// Marshal serializes the encounter data back to JSON, preserving all non-participant
// fields and using the (possibly mutated) participants slice.
func (ed *EncounterData) Marshal() (json.RawMessage, error) {
	participantsBytes, err := json.Marshal(ed.Participants)
	if err != nil {
		return nil, fmt.Errorf("marshal participants: %w", err)
	}

	ed.raw["participants"] = participantsBytes

	result, err := json.Marshal(ed.raw)
	if err != nil {
		return nil, fmt.Errorf("marshal encounter data: %w", err)
	}

	return result, nil
}

// FindParticipantByInstanceID searches by the InstanceID (encounter slot ID),
// used when targeting a specific creature/character in the encounter.
func (ed *EncounterData) FindParticipantByInstanceID(instanceID string) (*models.ParticipantFull, int, error) {
	for i := range ed.Participants {
		p := &ed.Participants[i]
		if p.InstanceID == instanceID {
			return p, i, nil
		}
	}

	return nil, -1, apperrors.ParticipantNotFoundErr
}

// persistEncounterData marshals and saves the mutated encounter data.
func persistEncounterData(ctx context.Context, uc *actionsUsecases, ed *EncounterData, encounterID string) error {
	data, err := ed.Marshal()
	if err != nil {
		return err
	}

	return uc.encounterRepo.UpdateEncounter(ctx, data, encounterID)
}

// applyDamageToTarget subtracts damage from a participant's HP (temp HP first).
func applyDamageToTarget(target *models.ParticipantFull, damage int) {
	if target.CharacterRuntime != nil {
		remaining := damage
		if target.CharacterRuntime.TemporaryHP > 0 {
			if target.CharacterRuntime.TemporaryHP >= remaining {
				target.CharacterRuntime.TemporaryHP -= remaining
				return
			}
			remaining -= target.CharacterRuntime.TemporaryHP
			target.CharacterRuntime.TemporaryHP = 0
		}
		target.CharacterRuntime.CurrentHP -= remaining
		if target.CharacterRuntime.CurrentHP < 0 {
			target.CharacterRuntime.CurrentHP = 0
		}

		return
	}

	// Creature runtime state path
	remaining := damage
	if target.RuntimeState.TempHP > 0 {
		if target.RuntimeState.TempHP >= remaining {
			target.RuntimeState.TempHP -= remaining
			return
		}
		remaining -= target.RuntimeState.TempHP
		target.RuntimeState.TempHP = 0
	}
	target.RuntimeState.CurrentHP -= remaining
	if target.RuntimeState.CurrentHP < 0 {
		target.RuntimeState.CurrentHP = 0
	}
}
