package router

import (
	actionsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/actions/delivery"
	"github.com/gorilla/mux"
)

func ServeActionsRouter(router *mux.Router, actionsHandler *actionsdel.ActionsHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/encounter").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	subrouter.HandleFunc("/{id}/actions", actionsHandler.ExecuteAction).Methods("POST")
}
