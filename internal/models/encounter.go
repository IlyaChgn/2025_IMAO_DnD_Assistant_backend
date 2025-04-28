package models

import "encoding/json"

type Encounter struct {
	ID     int             `json:"id"`
	UserID int             `json:"userID"`
	Name   string          `json:"name"`
	Data   json.RawMessage `json:"data"`
	UUID   string          `json:"encounterUUID"`
}

type EncounterInList struct {
	ID     int    `json:"id"`
	UserID int    `json:"userID"`
	Name   string `json:"name"`
	UUID   string `json:"encounterUUID"`
}

type EncountersList []*EncounterInList

type SaveEncounterReq struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

type GetEncountersListReq struct {
	Start  int          `json:"start"`
	Size   int          `json:"size"`
	Search SearchParams `json:"search"`
}
