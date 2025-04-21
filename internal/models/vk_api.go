package models

type VKTokensData struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	UserID       int    `json:"user_id"`
	State        string `json:"state"`
	Scope        string `json:"scope"`
}

type UserPublicInfo struct {
	UserID    string `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Avatar    string `json:"avatar"`
	Email     string `json:"email"`
}

type PublicInfo struct {
	User UserPublicInfo `json:"user"`
}

type VKExchangeError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type VKInfoError struct {
	VKExchangeError
	State string `json:"state"`
}
