package usecases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	"github.com/gorilla/websocket"
)

const (
	sessionDuration = 15 * time.Minute
)

type tableUsecases struct {
	tableManager  tableinterfaces.TableManager
	encounterRepo encounterinterfaces.EncounterRepository

	idGen        tableinterfaces.SessionIDGenerator
	timerFactory tableinterfaces.TimerFactory

	sessionWatcher map[string]tableinterfaces.SessionTimer
	mu             sync.RWMutex
}

func NewTableUsecases(encounterRepo encounterinterfaces.EncounterRepository,
	manager tableinterfaces.TableManager,
	idGen tableinterfaces.SessionIDGenerator,
	timerFactory tableinterfaces.TimerFactory) tableinterfaces.TableUsecases {
	return &tableUsecases{
		tableManager:   manager,
		encounterRepo:  encounterRepo,
		idGen:          idGen,
		timerFactory:   timerFactory,
		sessionWatcher: make(map[string]tableinterfaces.SessionTimer),
	}
}

func (uc *tableUsecases) CreateSession(ctx context.Context, admin *models.User, encounterID string) (string, error) {
	l := logger.FromContext(ctx)

	encounterData, err := uc.encounterRepo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		l.UsecasesError(err, admin.ID, map[string]any{"id": encounterID})
		return "", err
	}

	if encounterData.UserID != admin.ID {
		l.UsecasesWarn(apperrors.PermissionDeniedError, admin.ID, map[string]any{"id": encounterID})
		return "", apperrors.PermissionDeniedError
	}

	sessionID := uc.idGen.NewSessionID()

	uc.tableManager.CreateSession(ctx, admin, encounterData, sessionID, uc.refreshSession)

	uc.mu.Lock()

	uc.sessionWatcher[sessionID] = uc.timerFactory.AfterFunc(sessionDuration, func() {
		uc.stopTimer(ctx, sessionID, encounterID)
		l.UsecasesInfo(fmt.Sprintf("session timer stopped, sessionID: %s", sessionID), admin.ID)
	})

	uc.mu.Unlock()

	return sessionID, nil
}

func (uc *tableUsecases) GetTableData(ctx context.Context, sessionID string) (*models.TableData, error) {
	return uc.tableManager.GetTableData(ctx, sessionID)
}

func (uc *tableUsecases) AddNewConnection(ctx context.Context, user *models.User, sessionID string,
	conn *websocket.Conn) {
	uc.tableManager.AddNewConnection(ctx, user, sessionID, conn)
}

func (uc *tableUsecases) refreshSession(sessionID string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.sessionWatcher[sessionID].Stop()
	uc.sessionWatcher[sessionID].Reset(sessionDuration)
}

func (uc *tableUsecases) stopTimer(ctx context.Context, sessionID, encounterID string) {
	data, _ := uc.tableManager.GetEncounterData(ctx, sessionID)

	uc.mu.Lock()
	uc.sessionWatcher[sessionID].Stop()
	delete(uc.sessionWatcher, sessionID)
	uc.mu.Unlock()

	uc.encounterRepo.UpdateEncounter(context.Background(), data, encounterID)
	uc.tableManager.RemoveSession(ctx, sessionID)
}
