package router

import (
	authdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/delivery"
	"github.com/gorilla/mux"
)

func ServeAuthRouter(router *mux.Router, authHandler *authdel.AuthHandler) {
	subrouter := router.PathPrefix("/auth").Subrouter()

	subrouter.HandleFunc("/exchange", authHandler.Exchange).Methods("POST")
}
