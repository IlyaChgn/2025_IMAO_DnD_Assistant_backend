package repository

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

const maxPlayersNum = 4

type tableManager struct {
	sessions map[string]*session
	mu       sync.RWMutex
}

func NewTableManager() tableinterfaces.TableManager {
	return &tableManager{
		sessions: make(map[string]*session),
	}
}

func (tm *tableManager) CreateSession(admin *models.User, encounter *models.Encounter, sessionID string,
	callback func(sessionID string)) {
	newSession := &session{
		encounterID:     encounter.UUID,
		encounterName:   encounter.Name,
		encounterData:   encounter.Data,
		adminID:         admin.ID,
		adminName:       admin.Name,
		participants:    make(map[int]*participant),
		broadcast:       make(chan []byte),
		refreshCallback: callback,
	}

	tm.mu.Lock()
	tm.sessions[sessionID] = newSession
	tm.mu.Unlock()

	go newSession.run()
}

func (tm *tableManager) RemoveSession(sessionID string) {
	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		log.Println(apperrors.TableNotFoundErr)

		return
	}

	for key := range activeSession.participants {
		activeSession.RemoveParticipant(key)
	}

	tm.mu.Lock()
	delete(tm.sessions, sessionID)
	tm.mu.Unlock()
}

func (tm *tableManager) GetTableData(sessionID string) (*models.TableData, error) {
	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		return nil, apperrors.TableNotFoundErr
	}

	return activeSession.GetTableData(), nil
}

func (tm *tableManager) GetEncounterData(sessionID string) ([]byte, error) {
	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		return nil, apperrors.TableNotFoundErr
	}

	return activeSession.GetEncounterData(), nil
}

func (tm *tableManager) AddNewConnection(user *models.User, sessionID string, conn *websocket.Conn) {
	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		responses.SendWSErrResponse(conn, responses.WSStatusBadRequest, responses.ErrInvalidWSSessionID)
		conn.Close()

		return
	}

	var err error

	if activeSession.CheckUser(user.ID) {
		responses.SendWSErrResponse(conn, responses.WSStatusBadRequest, responses.ErrUserAlreadyExistsWS)
		conn.Close()

		return
	}

	if user.ID == activeSession.GetAdminID() {
		activeSession.AddAdmin(user.ID, user.Name, conn)
	} else {
		err = activeSession.AddParticipant(user.ID, user.Name, conn)
	}

	if err != nil {
		responses.SendWSErrResponse(conn, responses.WSStatusBadRequest, responses.ErrMaxPlayersWS)
		conn.Close()

		return
	}

	activeSession.refreshCallback(sessionID)
	activeSession.WriteFirstMsg(user.ID)

	go func() {
		defer func() {
			if activeSession.CheckUser(user.ID) {
				activeSession.RemoveParticipant(user.ID)
			}
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			activeSession.refreshCallback(sessionID)
			activeSession.broadcast <- msg
		}
	}()
}

func (tm *tableManager) HasActiveUsers(sessionID string) bool {
	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		log.Println(apperrors.TableNotFoundErr)

		return false
	}

	return activeSession.hasActiveUsers()
}
