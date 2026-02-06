package router

import (
	maptilesdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles/delivery"
	"github.com/gorilla/mux"
)

func ServeMapTilesRouter(
	router *mux.Router,
	mapTilesHandler *maptilesdel.MapTilesHandler,
	loginRequiredMiddleware mux.MiddlewareFunc,
) {
	subrouter := router.PathPrefix("/map-tiles").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	// Соответствует фронту: baseUrl '/api/map-tiles' + '/categories' (GET)
	subrouter.HandleFunc("/categories", mapTilesHandler.GetCategories).Methods("GET")

	// Walkability endpoints
	subrouter.HandleFunc("/walkability/{tileId}", mapTilesHandler.GetWalkabilityByTileID).Methods("GET")
	subrouter.HandleFunc("/walkability/{tileId}", mapTilesHandler.UpsertWalkability).Methods("PUT")
	subrouter.HandleFunc("/walkability", mapTilesHandler.GetWalkabilityBySetID).Methods("GET")
}
