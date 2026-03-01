package repository

import (
	"context"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils/merger"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

// incomingMessage is used to parse the type field from incoming WebSocket messages
type incomingMessage struct {
	Type models.WSMsgType `json:"type"`
}

// broadcastMessage carries a WebSocket message together with the sender's user ID.
type broadcastMessage struct {
	senderID int
	data     []byte
}

type participant struct {
	models.Participant
	Conn *websocket.Conn
}

type session struct {
	encounterID   string
	encounterName string
	encounterData []byte

	adminID         int
	adminName       string
	participants    map[int]*participant // Ключ - UserID
	playersNum      int
	broadcast       chan broadcastMessage
	refreshCallback func(sessionID string) // Вызов обновления таймера

	// AI auto-play settings (Phase 1: stored for future use).
	aiAutoPlay      bool
	aiDifficultyMod float64

	mu sync.RWMutex

	start   time.Time
	metrics metrics.WSSessionMetrics
}

func (s *session) run(ctx context.Context) {
	l := logger.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.broadcast:
			s.mu.Lock()

			// Check if this is a patch message that should be relayed directly
			var incoming incomingMessage
			if err := json.Unmarshal(msg.data, &incoming); err == nil && models.IsPatchMessage(incoming.Type) {
				// Relay patch message directly to all participants except the sender
				s.relayPatchMessage(l, msg.senderID, msg.data)
				s.mu.Unlock()
				continue
			}

			// Full state message - merge and broadcast
			var err error

			s.encounterData, err = merger.Merge(s.encounterData, msg.data)
			if err != nil {
				l.RepoError(err, nil)
				s.mu.Unlock()
				continue
			}

			s.broadcastState(l)

			s.mu.Unlock()
		}
	}
}

// relayPatchMessage broadcasts a patch message directly to all participants except the sender.
// Must be called with s.mu held.
func (s *session) relayPatchMessage(l logger.Logger, senderID int, msg []byte) {
	for id, p := range s.participants {
		if id == senderID {
			continue
		}

		err := p.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			l.RepoError(err, nil)
			p.Conn.Close()

			delete(s.participants, id)
			continue
		}

		s.metrics.IncSentMsgs()
	}
}

// broadcastState sends the current encounter state to all participants.
// Must be called with s.mu held.
func (s *session) broadcastState(l logger.Logger) {
	for id, p := range s.participants {
		err := responses.SendWSOkResponse(p.Conn, models.BattleInfo,
			&models.EncounterData{EncounterData: s.encounterData})
		if err != nil {
			l.RepoError(err, nil)
			p.Conn.Close()

			delete(s.participants, id)
			continue
		}

		s.metrics.IncSentMsgs()
	}
}

func (s *session) WriteFirstMsg(ctx context.Context, userID int) {
	l := logger.FromContext(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.IncSentMsgs()

	conn := s.participants[userID].Conn

	err := responses.SendWSOkResponse(conn, models.BattleInfo, &models.EncounterData{EncounterData: s.encounterData})
	if err != nil {
		l.RepoError(err, nil)
		conn.Close()

		delete(s.participants, userID)
	}
}

func (s *session) GetTableData() *models.TableData {
	data := &models.TableData{}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data.EncounterName = s.encounterName
	data.AdminName = s.adminName
	data.EncounterData = s.encounterData

	return data
}

func (s *session) GetEncounterData() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.encounterData
}

func (s *session) GetAdminID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.adminID
}

func (s *session) CheckUser(userID int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.participants[userID]

	return ok
}
