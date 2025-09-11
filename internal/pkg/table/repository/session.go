package repository

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils/merger"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

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
	broadcast       chan []byte
	refreshCallback func(sessionID string) // Вызов обновления таймера

	mu sync.RWMutex

	start   time.Time
	metrics metrics.WSSessionMetrics
}

func (s *session) run(ctx context.Context) {
	l := logger.FromContext(ctx)

	for {
		select {
		case msg := <-s.broadcast:
			s.mu.Lock()

			var err error

			s.encounterData, err = merger.Merge(s.encounterData, msg)
			if err != nil {
				l.RepoError(err, nil)
				return
			}

			for id, p := range s.participants {
				err := responses.SendWSOkResponse(p.Conn, models.BattleInfo,
					&models.EncounterData{EncounterData: s.encounterData})
				if err != nil {
					l.RepoError(err, nil)
					p.Conn.Close()

					delete(s.participants, id)
				}

				s.metrics.IncSentMsgs()
			}

			s.mu.Unlock()
		}
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
