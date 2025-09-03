package usecases

import (
	"context"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

const (
	sessionStringLen = 32
	sessionDuration  = 15 * time.Minute
)

type tableUsecases struct {
	tableManager  tableinterfaces.TableManager
	encounterRepo encounterinterfaces.EncounterRepository

	sessionWatcher map[string]*time.Timer
	mu             sync.RWMutex
}

func NewTableUsecases(encounterRepo encounterinterfaces.EncounterRepository,
	manager tableinterfaces.TableManager) tableinterfaces.TableUsecases {
	return &tableUsecases{
		tableManager:   manager,
		encounterRepo:  encounterRepo,
		sessionWatcher: make(map[string]*time.Timer),
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

	sessionID := utils.RandString(sessionStringLen)

	uc.tableManager.CreateSession(admin, encounterData, sessionID, uc.refreshSession)

	uc.mu.Lock()

	uc.sessionWatcher[sessionID] = time.AfterFunc(sessionDuration, func() {
		uc.stopTimer(sessionID, encounterID)
		l.UsecasesInfo(fmt.Sprintf("session timer stopped, sessionID: %s", sessionID), admin.ID)
	})

	uc.mu.Unlock()

	return sessionID, nil
}

func (uc *tableUsecases) GetTableData(sessionID string) (*models.TableData, error) {
	return uc.tableManager.GetTableData(sessionID)
}

func (uc *tableUsecases) AddNewConnection(user *models.User, sessionID string, conn *websocket.Conn) {
	uc.tableManager.AddNewConnection(user, sessionID, conn)
}

func (uc *tableUsecases) refreshSession(sessionID string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	uc.sessionWatcher[sessionID].Stop()
	uc.sessionWatcher[sessionID].Reset(sessionDuration)
}

func (uc *tableUsecases) stopTimer(sessionID, encounterID string) {
	data, _ := uc.tableManager.GetEncounterData(sessionID)

	uc.mu.Lock()
	uc.sessionWatcher[sessionID].Stop()
	delete(uc.sessionWatcher, sessionID)
	uc.mu.Unlock()

	uc.encounterRepo.UpdateEncounter(context.Background(), data, encounterID)
	uc.tableManager.RemoveSession(sessionID)
}
