package models

type CodeExchangeRequest struct {
	Code         string `json:"code"`
	State        string `json:"state"`
	CodeVerifier string `json:"codeVerifier"`
	DeviceID     string `json:"deviceID"`
}
