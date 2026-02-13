package router

import (
	itemsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items/delivery"
	"github.com/gorilla/mux"
)

func ServeItemsRouter(router *mux.Router, handler *itemsdel.ItemsHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	sub := router.PathPrefix("/items/definitions").Subrouter()

	// Public routes
	sub.HandleFunc("/{engName}", handler.GetItemByEngName).Methods("GET")
	sub.HandleFunc("", handler.GetItems).Methods("GET")

	// Protected routes
	protected := sub.PathPrefix("").Subrouter()
	protected.Use(loginRequiredMiddleware)
	protected.HandleFunc("", handler.CreateItem).Methods("POST")
	protected.HandleFunc("/{id}", handler.UpdateItem).Methods("PUT")
	protected.HandleFunc("/{id}", handler.DeleteItem).Methods("DELETE")
}
