package combatai

import "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"

// GetCurrentHP returns the current HP for a participant, abstracting the
// NPC (RuntimeState) vs PC (CharacterRuntime) storage difference.
func GetCurrentHP(p *models.ParticipantFull) int {
	if p.CharacterRuntime != nil {
		return p.CharacterRuntime.CurrentHP
	}
	return p.RuntimeState.CurrentHP
}

// IsAlive returns true if the participant has positive HP.
func IsAlive(p *models.ParticipantFull) bool {
	return GetCurrentHP(p) > 0
}
