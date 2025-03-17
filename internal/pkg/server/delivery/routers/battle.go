package router

import (
	descriptiondel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery"
	"github.com/gorilla/mux"
)

func ServeBattleRouter(router *mux.Router, descriptionHandler *descriptiondel.DescriptionHandler) {
	subrouter := router.PathPrefix("/battle").Subrouter()

	subrouter.HandleFunc("/generate_description", descriptionHandler.GenerateDescription).Methods("POST")
}
