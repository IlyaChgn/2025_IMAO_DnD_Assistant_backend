package logger

import "context"

type Logger interface {
	WithContext(ctx context.Context) context.Context

	ServerInfo(host, port string, isProduction bool)
	DBInfo(host, port, db string, dbName any, isSecure bool)
	DBFatal(host, port, db string, dbName any, isSecure bool, msg string, err error)

	Sync()
}
