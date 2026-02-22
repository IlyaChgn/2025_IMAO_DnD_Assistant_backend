package router

import (
	combataideliv "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/combatai/delivery"
	"github.com/gorilla/mux"
)

// ServeCombatAIRouter registers the combat AI endpoint.
func ServeCombatAIRouter(router *mux.Router, handler *combataideliv.CombatAIHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/encounter").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	subrouter.HandleFunc("/{id}/ai-turn", handler.AITurn).Methods("POST")
	subrouter.HandleFunc("/{id}/ai-round", handler.AIRound).Methods("POST")
}
