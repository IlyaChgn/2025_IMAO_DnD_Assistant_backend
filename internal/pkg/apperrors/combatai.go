package apperrors

import "errors"

var (
	NPCIsPlayerCharacterErr = errors.New("participant is a player character, not an NPC")
	NPCIsDeadErr            = errors.New("NPC is dead")
)
