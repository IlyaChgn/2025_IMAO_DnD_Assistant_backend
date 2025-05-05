package router

import (
	tabledel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table/delivery"
	"github.com/gorilla/mux"
)

func ServeTableRouter(router *mux.Router, tableHandler *tabledel.TableHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/table").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	subrouter.HandleFunc("/session", tableHandler.CreateSession).Methods("POST")
	subrouter.HandleFunc("/session/{id}", tableHandler.GetTableData).Methods("GET")
	subrouter.HandleFunc("/session/{id}/connect", tableHandler.ServeWS).Methods("GET")
}
