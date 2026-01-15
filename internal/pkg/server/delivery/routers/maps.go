package router

import (
	mapsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps/delivery"
	"github.com/gorilla/mux"
)

func ServeMapsRouter(router *mux.Router, mapsHandler *mapsdel.MapsHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/maps").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	subrouter.HandleFunc("", mapsHandler.ListMaps).Methods("GET")
	subrouter.HandleFunc("", mapsHandler.CreateMap).Methods("POST")
	subrouter.HandleFunc("/{id}", mapsHandler.GetMapByID).Methods("GET")
	subrouter.HandleFunc("/{id}", mapsHandler.UpdateMap).Methods("PUT")
	subrouter.HandleFunc("/{id}", mapsHandler.DeleteMap).Methods("DELETE")
}
