package router

import (
	backgroundsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/backgrounds/delivery"
	"github.com/gorilla/mux"
)

func ServeBackgroundsRouter(router *mux.Router, handler *backgroundsdel.BackgroundsHandler) {
	sub := router.PathPrefix("/reference/backgrounds").Subrouter()

	sub.HandleFunc("/{engName}", handler.GetBackgroundByEngName).Methods("GET")
	sub.HandleFunc("", handler.GetBackgrounds).Methods("GET")
}
