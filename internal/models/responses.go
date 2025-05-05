package models

type ErrResponse struct {
	Status string `json:"status"`
}

type WSErrResponse struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}
