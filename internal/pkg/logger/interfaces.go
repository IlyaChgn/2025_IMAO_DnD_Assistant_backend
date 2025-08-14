package logger

import "context"

type Logger interface {
	WithContext(ctx context.Context) context.Context

	Info(msg string)
}
