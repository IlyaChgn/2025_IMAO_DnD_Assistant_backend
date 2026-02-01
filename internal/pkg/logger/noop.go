package logger

import "context"

// noop is a package-level no-op logger returned by FromContext when no logger
// is present in the context. It silently discards all log calls.
var noop Logger = &noopLogger{}

type noopLogger struct{}

func (n *noopLogger) WithContext(ctx context.Context) context.Context { return ctx }

func (n *noopLogger) ServerInfo(host, port string, isProduction bool)         {}
func (n *noopLogger) DBInfo(host, port, db string, dbName any, isSecure bool) {}
func (n *noopLogger) DBFatal(host, port, db string, dbName any, isSecure bool, msg string, err error) {
}

func (n *noopLogger) DeliveryInfo(ctx context.Context, msg string, fields any) {}
func (n *noopLogger) DeliveryError(ctx context.Context, code int, status string, err error, fields any) {
}

func (n *noopLogger) UsecasesInfo(msg string, userID int)             {}
func (n *noopLogger) UsecasesWarn(err error, userID int, fields any)  {}
func (n *noopLogger) UsecasesError(err error, userID int, fields any) {}

func (n *noopLogger) RepoInfo(msg string, params map[string]any) {}
func (n *noopLogger) RepoWarn(err error, params map[string]any)  {}
func (n *noopLogger) RepoError(err error, params map[string]any) {}

func (n *noopLogger) ExternalInfo(ctx context.Context, msg string, params map[string]any) {}
func (n *noopLogger) ExternalWarn(ctx context.Context, err error, params map[string]any)  {}
func (n *noopLogger) ExternalError(ctx context.Context, err error, params map[string]any) {}

func (n *noopLogger) Sync() {}
