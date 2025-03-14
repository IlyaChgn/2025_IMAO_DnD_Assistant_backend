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

	StatusInternalServerError = 500
)

const (
	ErrInternalServer = "Server error"
	ErrBadJSON        = "Wrong JSON format"

	ErrCreatureNotFound = "Creature with same URL not found"
	ErrSizeOrPosition   = "Size and position cannot be less than zero"
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
