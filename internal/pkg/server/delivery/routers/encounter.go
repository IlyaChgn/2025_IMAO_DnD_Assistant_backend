package router

import (
	encounterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/delivery"
	"github.com/gorilla/mux"
)

func ServeEncounteRouter(router *mux.Router, encounterHandler *encounterdel.EncounterHandler) {
	subrouter := router.PathPrefix("/encounter").Subrouter()

	subrouter.HandleFunc("/list", encounterHandler.GetEncountersList).Methods("POST")
	subrouter.HandleFunc("/add_encounter", encounterHandler.AddEncounter).Methods("POST")
	subrouter.HandleFunc("/{id}", encounterHandler.GetEncounterByMongoId).Methods("GET")

}
