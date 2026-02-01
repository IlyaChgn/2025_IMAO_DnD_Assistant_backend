package table

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_table.go -package=mocks

import (
	"context"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/gorilla/websocket"
)

type TableManager interface {
	CreateSession(ctx context.Context, admin *models.User, encounter *models.Encounter, sessionID string,
		callback func(sessionID string))
	RemoveSession(ctx context.Context, sessionID string)
	GetTableData(ctx context.Context, sessionID string) (*models.TableData, error)
	GetEncounterData(ctx context.Context, sessionID string) ([]byte, error)
	AddNewConnection(ctx context.Context, user *models.User, sessionID string, conn *websocket.Conn)
	HasActiveUsers(ctx context.Context, sessionID string) bool
}

type TableUsecases interface {
	CreateSession(ctx context.Context, admin *models.User, encounterID string) (string, error)
	GetTableData(ctx context.Context, sessionID string) (*models.TableData, error)
	AddNewConnection(ctx context.Context, user *models.User, sessionID string, conn *websocket.Conn)
}

type SessionIDGenerator interface {
	NewSessionID() string
}

type SessionTimer interface {
	Stop() bool
	Reset(d time.Duration) bool
}

type TimerFactory interface {
	AfterFunc(d time.Duration, f func()) SessionTimer
}
