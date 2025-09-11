package logger

import "context"

type Logger interface {
	WithContext(ctx context.Context) context.Context

	ServerInfo(host, port string, isProduction bool)
	DBInfo(host, port, db string, dbName any, isSecure bool)
	DBFatal(host, port, db string, dbName any, isSecure bool, msg string, err error)

	DeliveryInfo(ctx context.Context, msg string, fields any)
	DeliveryError(ctx context.Context, code int, status string, err error, fields any)

	UsecasesInfo(msg string, userID int)
	UsecasesWarn(err error, userID int, fields any)
	UsecasesError(err error, userID int, fields any)

	RepoInfo(msg string, params map[string]any)
	RepoWarn(err error, params map[string]any)
	RepoError(err error, params map[string]any)

	ExternalInfo(ctx context.Context, msg string, params map[string]any)
	ExternalWarn(ctx context.Context, err error, params map[string]any)
	ExternalError(ctx context.Context, err error, params map[string]any)

	Sync()
}
