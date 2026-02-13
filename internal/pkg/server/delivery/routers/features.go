package router

import (
	featuresdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/features/delivery"
	"github.com/gorilla/mux"
)

func ServeFeaturesRouter(router *mux.Router, handler *featuresdel.FeaturesHandler) {
	sub := router.PathPrefix("/reference/features").Subrouter()

	// NOTE: route order matters — /by-class/{className} must come before /{id}
	sub.HandleFunc("/by-class/{className}", handler.GetFeaturesByClass).Methods("GET")
	sub.HandleFunc("/{id}", handler.GetFeatureByID).Methods("GET")
	sub.HandleFunc("", handler.GetFeatures).Methods("GET")
}
