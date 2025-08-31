package logger

import "context"

type Logger interface {
	WithContext(ctx context.Context) context.Context

	ServerInfo(host, port string, isProduction bool)
	DBInfo(host, port, db string, dbName any, isSecure bool)
	DBFatal(host, port, db string, dbName any, isSecure bool, msg string, err error)

	DeliveryInfo(ctx context.Context, msg string, fields any)
	DeliveryError(ctx context.Context, code int, status string, err error, fields any)

	Sync()
}
