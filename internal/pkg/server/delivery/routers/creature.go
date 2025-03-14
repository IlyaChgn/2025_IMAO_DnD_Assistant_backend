package router

import (
	creaturedel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature/delivery"
	"github.com/gorilla/mux"
)

func ServeCreatureRouter(router *mux.Router, creatureHandler *creaturedel.CreatureHandler) {
	subrouter := router.PathPrefix("/bestiary").Subrouter()

	subrouter.HandleFunc("/{name}", creatureHandler.GetCreatureByName).Methods("GET")

}
