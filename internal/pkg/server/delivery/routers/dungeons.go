package router

import (
	dungeongendel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dungeongen/delivery"
	"github.com/gorilla/mux"
)

func ServeDungeonsRouter(
	router *mux.Router,
	dungeonGenHandler *dungeongendel.DungeonGenHandler,
	loginRequiredMiddleware mux.MiddlewareFunc,
) {
	subrouter := router.PathPrefix("/dungeons").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	subrouter.HandleFunc("/generate", dungeonGenHandler.GenerateDungeon).Methods("POST")
}
