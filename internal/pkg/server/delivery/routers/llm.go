package router

import (
	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	"github.com/gorilla/mux"
)

func ServeLLMRouter(router *mux.Router, llmHandler *bestiarydel.LLMHandler) {
	subrouter := router.PathPrefix("/llm").Subrouter()

	subrouter.HandleFunc("/text", llmHandler.SubmitGenerationPrompt).Methods("POST")
	subrouter.HandleFunc("/image", llmHandler.SubmitGenerationImage).Methods("POST")
	subrouter.HandleFunc("/{id}", llmHandler.GetGenerationStatus).Methods("GET")

}
