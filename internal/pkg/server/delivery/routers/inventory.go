package router

import (
	itemsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items/delivery"
	"github.com/gorilla/mux"
)

func ServeInventoryRouter(router *mux.Router, handler *itemsdel.InventoryHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	sub := router.PathPrefix("/inventory").Subrouter()
	sub.Use(loginRequiredMiddleware)

	sub.HandleFunc("/containers", handler.GetContainers).Methods("GET")
	sub.HandleFunc("/containers/{id}", handler.GetContainer).Methods("GET")
	sub.HandleFunc("/containers", handler.CreateContainer).Methods("POST")
	sub.HandleFunc("/containers/{id}", handler.DeleteContainer).Methods("DELETE")
	sub.HandleFunc("/commands", handler.ExecuteCommand).Methods("POST")
	sub.HandleFunc("/generate-loot", handler.GenerateLoot).Methods("POST")
}
