package utils

import (
	"context"
	"net/http"
)

const (
	keyMethod  = "method"
	keyURL     = "path"
	keySession = "session"
)

func SaveRequestData(ctx context.Context, r *http.Request) context.Context {
	var sessionID string

	session, _ := r.Cookie("session_id")
	if session != nil {
		sessionID = session.Value
	}

	return context.WithValue(
		context.WithValue(
			context.WithValue(ctx, keyMethod, r.Method),
			keyURL, r.URL.String(),
		),
		keySession, sessionID,
	)
}

func GetMethod(ctx context.Context) string {
	if v, ok := ctx.Value(keyMethod).(string); ok {
		return v
	}
	return ""
}

func GetURL(ctx context.Context) string {
	if v, ok := ctx.Value(keyURL).(string); ok {
		return v
	}
	return ""
}

func GetSession(ctx context.Context) string {
	if v, ok := ctx.Value(keySession).(string); ok {
		return v
	}
	return ""
}
