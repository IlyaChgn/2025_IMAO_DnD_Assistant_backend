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
	ID          int    `json:"id"`
	VKID        string `json:"vkID"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Status      string `json:"status,omitempty"`
}

type UserIdentity struct {
	ID             int    `json:"id"`
	UserID         int    `json:"userId"`
	Provider       string `json:"provider"`
	ProviderUserID string `json:"providerUserId"`
	Email          string `json:"email,omitempty"`
	CreatedAt      string `json:"createdAt,omitempty"`
	LastUsedAt     string `json:"lastUsedAt,omitempty"`
}

type FullSessionData struct {
	Tokens TokensData `json:"tokens"`
	User   User       `json:"user"`
}

type AuthResponse struct {
	IsAuth bool `json:"isAuth"`
	User   User `json:"user"`
}
