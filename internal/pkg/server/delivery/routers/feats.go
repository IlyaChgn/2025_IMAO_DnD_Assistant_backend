package router

import (
	featsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/feats/delivery"
	"github.com/gorilla/mux"
)

func ServeFeatsRouter(router *mux.Router, handler *featsdel.FeatsHandler) {
	sub := router.PathPrefix("/reference/feats").Subrouter()

	sub.HandleFunc("/{engName}", handler.GetFeatByEngName).Methods("GET")
	sub.HandleFunc("", handler.GetFeats).Methods("GET")
}
