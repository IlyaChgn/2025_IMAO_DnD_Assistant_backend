package models

import "encoding/json"

type CreateTableRequest struct {
	EncounterID string `json:"encounterID"`
}

type CreateTableResponse struct {
	SessionID string `json:"sessionID"`
}

type TableData struct {
	AdminName     string          `json:"adminName"`
	EncounterName string          `json:"encounterName"`
	EncounterData json.RawMessage `json:"encounterData"`
}

type Role string

const (
	Admin  Role = "admin"
	Player Role = "player"
)

type Participant struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role Role   `json:"role"`
}

type ParticipantStatus string

const (
	Connected    ParticipantStatus = "connected"
	Disconnected ParticipantStatus = "disconnected"
)

type ParticipantsInfoMsg struct {
	Status ParticipantStatus `json:"status"`
	ID     int               `json:"id"`

	Participants []Participant `json:"participants"`
}

type EncounterData struct {
	EncounterData json.RawMessage `json:"encounterData"`
}
