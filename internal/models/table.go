package models

type CreateTableRequest struct {
	EncounterID string `json:"encounterID"`
}

type CreateTableResponse struct {
	SessionID string `json:"sessionID"`
}

type TableData struct {
	AdminName     string `json:"adminName"`
	EncounterName string `json:"encounterName"`
}
