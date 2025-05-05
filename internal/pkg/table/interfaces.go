package table

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/gorilla/websocket"
)

type TableManager interface {
	CreateSession(admin *models.User, encounter *models.Encounter, sessionID string, callback func(sessionID string))
	RemoveSession(sessionID string)
	GetTableData(sessionID string) (*models.TableData, error)
	GetEncounterData(sessionID string) ([]byte, error)
	AddNewConnection(user *models.User, sessionID string, conn *websocket.Conn)
	HasActiveUsers(sessionID string) bool
}

type TableUsecases interface {
	CreateSession(ctx context.Context, admin *models.User, encounterID string) (string, error)
	GetTableData(sessionID string) (*models.TableData, error)
	AddNewConnection(user *models.User, sessionID string, conn *websocket.Conn)
}
