package delivery

import (
	"net/http"
	"time"
)

func (h *AuthHandler) createSession(sessionID string) *http.Cookie {
	return &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(h.sessionDuration),
		HttpOnly: true,
	}
}
