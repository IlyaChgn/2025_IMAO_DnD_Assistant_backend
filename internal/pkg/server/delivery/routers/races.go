package router

import (
	racesdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races/delivery"
	"github.com/gorilla/mux"
)

func ServeRacesRouter(router *mux.Router, handler *racesdel.RacesHandler) {
	sub := router.PathPrefix("/reference/races").Subrouter()

	sub.HandleFunc("/{engName}", handler.GetRaceByEngName).Methods("GET")
	sub.HandleFunc("", handler.GetRaces).Methods("GET")
}
