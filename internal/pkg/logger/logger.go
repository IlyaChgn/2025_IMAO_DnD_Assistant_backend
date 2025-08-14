package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ctxKey string

type logger struct {
	zap *zap.SugaredLogger
}

func New(key, outputPath, errPath string) (Logger, error) {
	config := zap.NewProductionConfig()
	config.DisableStacktrace = false
	config.OutputPaths = []string{outputPath}
	config.ErrorOutputPaths = []string{errPath}
	config.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "lvl",
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

	ctxKey = key
	mylogger := &logger{
		zap: l.Sugar(),
	}

	return mylogger, nil
}

func (l *logger) Info(msg string) {
	l.zap.Info(msg)
}
