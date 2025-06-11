package dbinit

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

const (
	defaultMaxConns          = int32(90)
	defaultMinConns          = int32(0)
	defaultMaxConnLifetime   = time.Hour
	defaultMaxConnIdleTime   = time.Minute * 30
	defaultHealthCheckPeriod = time.Minute
	defaultConnectTimeout    = time.Second * 5
)

func NewConnectionString(user, password, host, port, dbname string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)
}

func postgresPoolConfig(dbURL string) *pgxpool.Config {
	dbConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("Failed to create postgres config ", err)
	}

	dbConfig.MaxConns = defaultMaxConns
	dbConfig.MinConns = defaultMinConns
	dbConfig.MaxConnLifetime = defaultMaxConnLifetime
	dbConfig.MaxConnIdleTime = defaultMaxConnIdleTime
	dbConfig.HealthCheckPeriod = defaultHealthCheckPeriod
	dbConfig.ConnConfig.ConnectTimeout = defaultConnectTimeout

	dbConfig.BeforeAcquire = func(_ context.Context, _ *pgx.Conn) bool {
		return true
	}

	dbConfig.AfterRelease = func(_ *pgx.Conn) bool {
		return true
	}

	return dbConfig
}

func NewPostgresPool(dbURL string) (*pgxpool.Pool, error) {
	postgresCfg := postgresPoolConfig(dbURL)

	pool, err := pgxpool.NewWithConfig(context.Background(), postgresCfg)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
