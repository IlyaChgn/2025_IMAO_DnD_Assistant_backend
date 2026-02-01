package logger

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// loggerCtxKey is an unexported struct type used as the context key for the logger.
// Using a struct type (not string) avoids collisions and is safe for parallel tests.
type loggerCtxKey struct{}

type logger struct {
	zap *zap.SugaredLogger
}

func New(outputPath, errPath string) (Logger, error) {
	config := zap.NewProductionConfig()
	config.DisableStacktrace = false
	config.OutputPaths = []string{outputPath}
	config.ErrorOutputPaths = []string{errPath}
	config.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "lvl",
		CallerKey:      "caller",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	l, err := config.Build()
	if err != nil {
		return nil, err
	}

	mylogger := &logger{
		zap: l.Sugar().WithOptions(zap.AddCallerSkip(1)),
	}

	return mylogger, nil
}

func (l *logger) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, l.with("request-uuid", uuid.NewString()))
}

// FromContext extracts the Logger from context. If no logger is present,
// it returns a no-op logger instead of panicking.
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerCtxKey{}).(Logger); ok {
		return l
	}
	return noop
}

func (l *logger) with(key string, val interface{}) *logger {
	var newLogger *zap.SugaredLogger

	switch v := val.(type) {
	case string:
		newLogger = l.zap.With(zap.String(key, v))
	case int:
		newLogger = l.zap.With(zap.Int(key, v))
	case int64:
		newLogger = l.zap.With(zap.Int64(key, v))
	case float64:
		newLogger = l.zap.With(zap.Float64(key, v))
	case bool:
		newLogger = l.zap.With(zap.Bool(key, v))
	default:
		newLogger = l.zap.With(zap.Any(key, v))
	}

	return &logger{zap: newLogger}
}

func (l *logger) Sync() {
	l.zap.Sync()
}
