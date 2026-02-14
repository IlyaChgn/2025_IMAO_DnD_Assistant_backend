package router

import (
	conditionsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/conditions/delivery"
	"github.com/gorilla/mux"
)

func ServeConditionsRouter(router *mux.Router, handler *conditionsdel.ConditionsHandler) {
	sub := router.PathPrefix("/reference/conditions").Subrouter()

	sub.HandleFunc("/{type}", handler.GetConditionByType).Methods("GET")
	sub.HandleFunc("", handler.GetConditions).Methods("GET")
}
