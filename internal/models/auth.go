package models

type CodeExchangeRequest struct {
	Code         string `json:"code"`
	State        string `json:"state"`
	CodeVerifier string `json:"codeVerifier"`
	DeviceID     string `json:"deviceID"`
}

type TokensData struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	IDToken      string `json:"idToken"`
}

type User struct {
	ID     int    `json:"id"`
	VKID   int    `json:"vkID"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type FullSessionData struct {
	Tokens TokensData `json:"tokens"`
	User   User       `json:"user"`
}
