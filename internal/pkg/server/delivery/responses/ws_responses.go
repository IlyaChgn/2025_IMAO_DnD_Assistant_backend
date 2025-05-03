package responses

import (
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/gorilla/websocket"
	"log"
)

const (
	WSStatusBadRequest  = 1008
	WSStatusInternalErr = 1011
)

const (
	ErrInvalidWSSessionID  = "Invalid session ID"
	ErrMaxPlayersWS        = "Max players number had reached"
	ErrInternalWS          = "Internal server error"
	ErrUserAlreadyExistsWS = "User with this name already exists in session"
)

func newWsErrResponse(err string) *models.WSErrResponse {
	return &models.WSErrResponse{
		Type:  "error",
		Error: err,
	}
}

func SendWSErrResponse(conn *websocket.Conn, code int, message string) {
	msg := newWsErrResponse(message)

	serverResponse, err := json.Marshal(msg)
	if err != nil {
		log.Println("Something went wrong while marshalling JSON", err)

		return
	}

	conn.WriteMessage(websocket.TextMessage, serverResponse)
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, message))
}
