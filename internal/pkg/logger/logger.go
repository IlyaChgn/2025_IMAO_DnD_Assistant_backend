package logger

import "os"

func (l *logger) DeliveryInfo(msg string) {
	ln := l.with("ex", []int{1, 2, 3, 5})
	ln.zap.Info(msg)
}

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
