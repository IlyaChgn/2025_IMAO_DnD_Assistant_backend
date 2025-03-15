package router

import (
	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	creaturedel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature/delivery"
	"github.com/gorilla/mux"
)

func ServeBestiaryRouter(router *mux.Router, creatureHandler *creaturedel.CreatureHandler,
	bestiaryHandler *bestiarydel.BestiaryHandler) {
	subrouter := router.PathPrefix("/bestiary").Subrouter()

	subrouter.HandleFunc("/list", bestiaryHandler.GetCreaturesList).Methods("POST")
	subrouter.HandleFunc("/{name}", creatureHandler.GetCreatureByName).Methods("GET")
}
