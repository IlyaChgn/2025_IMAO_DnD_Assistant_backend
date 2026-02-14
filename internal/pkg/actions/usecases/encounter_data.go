package usecases

import (
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
