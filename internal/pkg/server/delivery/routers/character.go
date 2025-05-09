package router

import (
	characterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/delivery"
	"github.com/gorilla/mux"
)

func ServeCharacterRouter(router *mux.Router, characterHandler *characterdel.CharacterHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/character").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	subrouter.HandleFunc("/list", characterHandler.GetCharactersList).Methods("POST")
	subrouter.HandleFunc("/add_character", characterHandler.AddCharacter).Methods("POST")
	subrouter.HandleFunc("/{id}", characterHandler.GetCharacterByMongoId).Methods("GET")
}
