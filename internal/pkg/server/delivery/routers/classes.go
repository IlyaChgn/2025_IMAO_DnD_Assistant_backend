package router

import (
	classesdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/classes/delivery"
	"github.com/gorilla/mux"
)

func ServeClassesRouter(router *mux.Router, handler *classesdel.ClassesHandler) {
	sub := router.PathPrefix("/reference/classes").Subrouter()

	sub.HandleFunc("/{engName}", handler.GetClassByEngName).Methods("GET")
	sub.HandleFunc("", handler.GetClasses).Methods("GET")
}
