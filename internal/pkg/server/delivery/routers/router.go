package router

import (
	myrecovery "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/middleware/recover"
	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	router := mux.NewRouter()

	router.Use(myrecovery.RecoveryMiddleware)

	// apiRouter := router.PathPrefix("/api").Subrouter()

	return router
}
