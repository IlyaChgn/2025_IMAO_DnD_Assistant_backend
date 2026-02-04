package repository

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

const maxPlayersNum = 4

type tableManager struct {
	sessions       map[string]*session
	mu             sync.RWMutex
	metrics        metrics.WSMetrics
	sessionMetrics metrics.WSSessionMetrics
}

func NewTableManager(metrics metrics.WSMetrics, sessionMetrics metrics.WSSessionMetrics) tableinterfaces.TableManager {
	return &tableManager{
		sessions:       make(map[string]*session),
		metrics:        metrics,
		sessionMetrics: sessionMetrics,
	}
}

func (tm *tableManager) CreateSession(ctx context.Context, admin *models.User, encounter *models.Encounter,
	sessionID string, callback func(sessionID string)) {
	l := logger.FromContext(ctx)
	newSession := &session{
		encounterID:     encounter.UUID,
		encounterName:   encounter.Name,
		encounterData:   encounter.Data,
		adminID:         admin.ID,
		adminName:       admin.DisplayName,
		participants:    make(map[int]*participant),
		broadcast:       make(chan []byte),
		refreshCallback: callback,
		start:           time.Now(),
		metrics:         tm.sessionMetrics,
	}

	tm.mu.Lock()
	tm.sessions[sessionID] = newSession
	tm.metrics.IncSessions()
	tm.mu.Unlock()

	l.RepoInfo("created WS session", map[string]any{"admin_id": admin.ID, "session_id": sessionID})

	go newSession.run(ctx)
}

func (tm *tableManager) RemoveSession(ctx context.Context, sessionID string) {
	l := logger.FromContext(ctx)

	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		l.RepoWarn(apperrors.TableNotFoundErr, map[string]any{"session_id": sessionID})
		return
	}

	for key := range activeSession.participants {
		activeSession.RemoveParticipant(ctx, key)
		l.RepoInfo("user successfully removed", map[string]any{"session_id": sessionID, "participant_id": key})
	}

	tm.mu.Lock()
	tm.metrics.IncreaseDuration(time.Since(activeSession.start))
	delete(tm.sessions, sessionID)
	tm.mu.Unlock()

	l.RepoInfo("session removed", map[string]any{"session_id": sessionID})
}

func (tm *tableManager) GetTableData(ctx context.Context, sessionID string) (*models.TableData, error) {
	l := logger.FromContext(ctx)

	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		l.RepoWarn(apperrors.TableNotFoundErr, map[string]any{"session_id": sessionID})
		return nil, apperrors.TableNotFoundErr
	}

	return activeSession.GetTableData(), nil
}

func (tm *tableManager) GetEncounterData(ctx context.Context, sessionID string) ([]byte, error) {
	l := logger.FromContext(ctx)

	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		l.RepoWarn(apperrors.TableNotFoundErr, map[string]any{"session_id": sessionID})
		return nil, apperrors.TableNotFoundErr
	}

	return activeSession.GetEncounterData(), nil
}

func (tm *tableManager) AddNewConnection(ctx context.Context, user *models.User, sessionID string,
	conn *websocket.Conn) {
	l := logger.FromContext(ctx)

	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.metrics.IncConns()
	tm.mu.RUnlock()

	if !ok {
		l.RepoWarn(apperrors.TableNotFoundErr, map[string]any{"session_id": sessionID})
		responses.SendWSErrResponse(conn, responses.WSStatusBadRequest, responses.ErrInvalidWSSessionID)
		conn.Close()

		return
	}

	var err error

	if activeSession.CheckUser(user.ID) {
		l.RepoWarn(apperrors.UserAlreadyExistsErr, map[string]any{"session_id": sessionID, "user_id": user.ID})
		responses.SendWSErrResponse(conn, responses.WSStatusBadRequest, responses.ErrUserAlreadyExistsWS)
		conn.Close()

		return
	}

	if user.ID == activeSession.GetAdminID() {
		activeSession.AddAdmin(ctx, user.ID, user.DisplayName, conn)
	} else {
		err = activeSession.AddParticipant(ctx, user.ID, user.DisplayName, conn)
	}

	if err != nil {
		l.RepoWarn(err, map[string]any{"session_id": sessionID, "user_id": user.ID})
		responses.SendWSErrResponse(conn, responses.WSStatusBadRequest, responses.ErrMaxPlayersWS)
		conn.Close()

		return
	}

	activeSession.refreshCallback(sessionID)
	activeSession.WriteFirstMsg(ctx, user.ID)

	l.RepoInfo("new connection added", map[string]any{"session_id": sessionID, "user_id": user.ID})

	go func() {
		defer func() {
			if activeSession.CheckUser(user.ID) {
				l.RepoInfo("user successfully removed",
					map[string]any{"session_id": sessionID, "user_id": user.ID})
				activeSession.RemoveParticipant(ctx, user.ID)
			}
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				l.RepoError(err, map[string]any{"session_id": sessionID, "user_id": user.ID, "ws_msg": string(msg)})
				break
			}

			activeSession.refreshCallback(sessionID)
			activeSession.broadcast <- msg
			activeSession.metrics.IncReceivedMsgs()
		}
	}()
}

func (tm *tableManager) HasActiveUsers(ctx context.Context, sessionID string) bool {
	l := logger.FromContext(ctx)

	tm.mu.RLock()
	activeSession, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		l.RepoWarn(apperrors.TableNotFoundErr, map[string]any{"session_id": sessionID})
		return false
	}

	return activeSession.hasActiveUsers()
}
