package repository

import (
	"context"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/websocket"
)

func (s *session) AddParticipant(ctx context.Context, userID int, name string, conn *websocket.Conn) error {
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

	s.sendParticipantsInfo(ctx, userID, models.Connected)

	return nil
}

func (s *session) AddAdmin(ctx context.Context, userID int, name string, conn *websocket.Conn) {
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

	s.sendParticipantsInfo(ctx, userID, models.Connected)
}

func (s *session) RemoveParticipant(ctx context.Context, userID int) {
	l := logger.FromContext(ctx)

	defer func() {
		if err := recover(); err != nil {
			l.RepoError(fmt.Errorf("recovered from panic after deleting participant, %v", err),
				map[string]any{"user_id": userID})
		}
	}()

	s.mu.Lock()

	if s.participants[userID].Participant.Role != models.Admin {
		s.playersNum--
	}

	s.participants[userID].Conn.Close()
	delete(s.participants, userID)

	s.mu.Unlock()

	s.sendParticipantsInfo(ctx, userID, models.Disconnected)
}

func (s *session) sendParticipantsInfo(ctx context.Context, userID int, status models.ParticipantStatus) {
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
