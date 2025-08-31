package logger

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"os"
)

func (l *logger) ServerInfo(host, port string, isProduction bool) {
	mode := "development"
	if isProduction {
		mode = "production"
	}

	ln := l.with("layer", "server").
		with("host", host).
		with("port", port).
		with("mode", mode).
		with("msg", "server started successfully")
	ln.zap.Info()
}

func (l *logger) logDB(host, port, db string, dbName any, isSecure bool, msg string) *logger {
	connType := "insecure"
	if isSecure {
		connType = "secure"
	}

	return l.with("layer", "server").
		with("host", host).
		with("port", port).
		with("db", db).
		with("db_name", dbName).
		with("conn_type", connType).
		with("msg", msg)
}

func (l *logger) DBInfo(host, port, db string, dbName any, isSecure bool) {
	ln := l.logDB(host, port, db, dbName, isSecure, "db connection opened successfully")
	ln.zap.Info()
}

func (l *logger) DBFatal(host, port, db string, dbName any, isSecure bool, msg string, err error) {
	ln := l.logDB(host, port, db, dbName, isSecure, msg).with("err", err.Error())
	ln.zap.Error()

	os.Exit(1)
}

func (l *logger) logDeliveryRequest(ctx context.Context) *logger {
	return l.with("layer", "delivery").
		with("path", utils.GetURL(ctx)).
		with("method", utils.GetMethod(ctx)).
		with("session_id", utils.GetSession(ctx))
}

func (l *logger) logDeliveryResponse(code int, status string) *logger {
	return l.with("code", code).
		with("status", status)
}

func (l *logger) DeliveryInfo(ctx context.Context, msg string, fields any) {
	newLogger := l
	newLogger = newLogger.with("fields", fields)

	newLogger.logDeliveryRequest(ctx).
		with("msg", msg).
		zap.Info()
}

func (l *logger) DeliveryError(ctx context.Context, code int, status string, err error, fields any) {
	newLogger := l

	newLogger = newLogger.logDeliveryRequest(ctx).logDeliveryResponse(code, status)

	if code == 500 {
		newLogger.with("err", err.Error()).
			with("fields", fields).
			zap.Error()
		return
	}
	newLogger.zap.Warn()
}
