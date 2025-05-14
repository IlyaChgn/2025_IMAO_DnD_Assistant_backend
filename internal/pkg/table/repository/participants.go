package repository

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/websocket"
)

func (s *session) AddParticipant(userID int, name string, conn *websocket.Conn) error {
	s.mu.Lock()

	if s.playersNum == maxPlayersNum {
		return apperrors.PlayersNumErr
	}
	s.playersNum++

	newParticipant := &participant{
		Participant: models.Participant{
			ID:   userID,
			Name: name,
			Role: models.Player,
		},
		Conn: conn,
	}
	s.participants[userID] = newParticipant

	s.mu.Unlock()

	s.sendParticipantsInfo(userID, models.Connected)

	return nil
}

func (s *session) AddAdmin(userID int, name string, conn *websocket.Conn) {
	s.mu.Lock()

	newParticipant := &participant{
		Participant: models.Participant{
			ID:   userID,
			Name: name,
			Role: models.Admin,
		},
		Conn: conn,
	}
	s.participants[userID] = newParticipant

	s.mu.Unlock()

	s.sendParticipantsInfo(userID, models.Connected)
}

func (s *session) RemoveParticipant(userID int) {
	s.mu.Lock()

	if s.participants[userID].Participant.Role != models.Admin {
		s.playersNum--
	}

	s.participants[userID].Conn.Close()
	delete(s.participants, userID)

	s.mu.Unlock()

	s.sendParticipantsInfo(userID, models.Disconnected)
}

func (s *session) sendParticipantsInfo(userID int, status models.ParticipantStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]models.Participant, 0, maxPlayersNum+1)
	for _, p := range s.participants {
		list = append(list, p.Participant)
	}

	connectedMsg := models.ParticipantsInfoMsg{
		Status:       status,
		ID:           userID,
		Participants: list,
	}

	for id, p := range s.participants {
		err := responses.SendWSOkResponse(p.Conn, models.ParticipantsInfo, connectedMsg)
		if err != nil {
			p.Conn.Close()

			delete(s.participants, id)
		}
	}
}

func (s *session) hasActiveUsers() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.participants) > 0
}
