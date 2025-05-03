package repository

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/gorilla/websocket"
	"github.com/qjebbs/go-jsons"
	"sync"
)

type Role string

const (
	Admin  Role = "admin"
	Player Role = "player"
)

type participant struct {
	Name string
	Role Role
	Conn *websocket.Conn
}

type session struct {
	encounterID   string
	encounterName string
	encounterData []byte

	adminID   int
	adminName string

	participants map[int]*participant // Ключ - UserID
	playersNum   int
	mu           sync.RWMutex

	broadcast chan []byte

	refreshCallback func(sessionID string) // Вызов обновления таймера
}

func (s *session) run() {
	for {
		select {
		case msg := <-s.broadcast:
			s.mu.Lock()

			s.encounterData, _ = jsons.Merge(s.encounterData, msg)

			for id, p := range s.participants {
				err := p.Conn.WriteMessage(websocket.TextMessage, s.encounterData)
				if err != nil {
					p.Conn.Close()

					delete(s.participants, id)
				}
			}

			s.mu.Unlock()
		}
	}
}

func (s *session) AddParticipant(userID int, name string, conn *websocket.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.playersNum == maxPlayersNum {
		return apperrors.PlayersNumErr
	}
	s.playersNum++

	newParticipant := &participant{
		Name: name,
		Role: Player,
		Conn: conn,
	}

	s.participants[userID] = newParticipant

	return nil
}

func (s *session) AddAdmin(userID int, name string, conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newParticipant := &participant{
		Name: name,
		Role: Admin,
		Conn: conn,
	}

	s.participants[userID] = newParticipant
}

func (s *session) RemoveParticipant(userID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.participants[userID].Conn.Close()

	delete(s.participants, userID)
}

func (s *session) GetTableData() *models.TableData {
	data := &models.TableData{}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data.EncounterName = s.encounterName
	data.AdminName = s.adminName

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
