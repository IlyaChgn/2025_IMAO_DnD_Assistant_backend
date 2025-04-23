package router

import (
	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	"github.com/gorilla/mux"
)

func ServeBestiaryRouter(router *mux.Router, bestiaryHandler *bestiarydel.BestiaryHandler) {
	subrouter := router.PathPrefix("/bestiary").Subrouter()

	subrouter.HandleFunc("/list", bestiaryHandler.GetCreaturesList).Methods("POST")
	subrouter.HandleFunc("/{name}", bestiaryHandler.GetCreatureByName).Methods("GET")
	subrouter.HandleFunc("/generated_creature", bestiaryHandler.AddGeneratedCreature).Methods("POST")
}
