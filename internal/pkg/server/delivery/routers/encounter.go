package router

import (
	encounterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/delivery"
	"github.com/gorilla/mux"
)

func ServeEncounteRouter(router *mux.Router, encounterHandler *encounterdel.EncounterHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/encounter").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	subrouter.HandleFunc("", encounterHandler.SaveEncounter).Methods("POST")
	subrouter.HandleFunc("/list", encounterHandler.GetEncountersList).Methods("POST")
	subrouter.HandleFunc("/{id}", encounterHandler.GetEncounterByID).Methods("GET")
	subrouter.HandleFunc("/{id}", encounterHandler.UpdateEncounter).Methods("POST")
	subrouter.HandleFunc("/{id}", encounterHandler.RemoveEncounter).Methods("DELETE")
}
