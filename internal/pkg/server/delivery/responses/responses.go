package responses

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

const (
	StatusOk = 200

	StatusBadRequest   = 400
	StatusUnauthorized = 401
	StatusForbidden    = 403

	StatusInternalServerError = 500
)

const (
	ErrInternalServer = "Server error"
	ErrVKServer       = "VK server error"
	ErrOAuthProvider  = "OAuth provider error"

	ErrBadJSON               = "Wrong JSON format"
	ErrBadRequest            = "Bad request"
	ErrIdentityAlreadyLinked = "Identity already linked to another user"
	ErrLastIdentity          = "Cannot unlink last identity"
	ErrNotAuthorized         = "User not authorized"
	ErrForbidden             = "User have no access to this content"
	ErrUserInactive          = "USER_INACTIVE"

	ErrCreatureNotFound  = "Creature with same URL not found"
	ErrCharacterNotFound = "Character with same URL not found"
	ErrSizeOrPosition    = "Size and position cannot be less than zero"
	ErrWrongDirection    = "Wrong direction type in order"

	ErrEmptyCharacterData = "Empty character data"

	ErrWrongFileSize = "File is too large"
	ErrWrongFileType = "Invalid file type. Only JSON files are allowed"

	ErrWrongEncounterName = "Encounter name must not be empty and more than 60 characters"
	ErrInvalidID          = "Invalid ID"

	ErrWrongTableID = "Wrong table ID"
	ErrWSUpgrade    = "Websocket upgrade error"

	ErrWrongImage  = "Bad image"
	ErrEmptyImage  = "Image not provided"
	ErrWrongJobID  = "Wrong job ID"
	ErrWrongBase64 = "Invalid base64 format"
)

func newErrResponse(status string) *models.ErrResponse {
	return &models.ErrResponse{
		Status: status,
	}
}

func sendResponse(writer http.ResponseWriter, response any) {
	serverResponse, err := json.Marshal(response)
	if err != nil {
		log.Println("Something went wrong while marshalling JSON", err)
		http.Error(writer, ErrInternalServer, StatusInternalServerError)

		return
	}

	_, err = writer.Write(serverResponse)
	if err != nil {
		log.Println("Something went wrong while senddng response", err)
		http.Error(writer, ErrInternalServer, StatusInternalServerError)

		return
	}
}

func SendOkResponse(writer http.ResponseWriter, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(StatusOk)

	sendResponse(writer, body)
}

func SendErrResponse(writer http.ResponseWriter, code int, status string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)

	response := newErrResponse(status)

	sendResponse(writer, response)
}
