package logger

import (
	"context"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (l *logger) WithContext(ctx context.Context) context.Context {
	l.zap.With(zap.String("request-uuid", uuid.NewString()))

	return context.WithValue(ctx, ctxKey, l)
}

func FromContext(ctx context.Context) Logger {
	return ctx.Value(ctxKey).(Logger)
}
