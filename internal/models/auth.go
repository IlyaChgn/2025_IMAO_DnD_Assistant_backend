package models

type LoginRequest struct {
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
	VKID   string `json:"vkID"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Status string `json:"status,omitempty"`
}

type FullSessionData struct {
	Tokens TokensData `json:"tokens"`
	User   User       `json:"user"`
}

type AuthResponse struct {
	IsAuth bool `json:"isAuth"`
	User   User `json:"user"`
}
