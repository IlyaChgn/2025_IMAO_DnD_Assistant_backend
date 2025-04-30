package models

import "encoding/json"

type Encounter struct {
	UserID int             `json:"userID"`
	Name   string          `json:"name"`
	Data   json.RawMessage `json:"data"`
	UUID   string          `json:"id"`
}

type EncounterInList struct {
	UserID int    `json:"userID"`
	Name   string `json:"name"`
	UUID   string `json:"id"`
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
