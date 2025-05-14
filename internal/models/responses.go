package models

type ErrResponse struct {
	Status string `json:"status"`
}

type WSErrResponse struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

type WSResponse struct {
	Type WSMsgType `json:"type"`
	Data any       `json:"data"`
}

type WSMsgType string

const (
	BattleInfo       WSMsgType = "battleInfo"
	ParticipantsInfo WSMsgType = "participantsInfo"
)
