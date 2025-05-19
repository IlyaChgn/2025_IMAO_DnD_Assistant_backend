package router

import (
	"net/http"

	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	"github.com/gorilla/mux"
)

func ServeBestiaryRouter(router *mux.Router, bestiaryHandler *bestiarydel.BestiaryHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/bestiary").Subrouter()

	subrouter.HandleFunc("/list", bestiaryHandler.GetCreaturesList).Methods("POST")
	subrouter.HandleFunc("/{name}", bestiaryHandler.GetCreatureByName).Methods("GET")

	subrouter.Handle("/generated_creature",
		loginRequiredMiddleware(http.HandlerFunc(bestiaryHandler.AddGeneratedCreature)),
	).Methods("POST")

	subrouter.Handle("/statblock-image",
		loginRequiredMiddleware(http.HandlerFunc(bestiaryHandler.UploadCreatureStatblockImage)),
	).Methods("POST")

	subrouter.Handle("/creature-generation-prompt",
		loginRequiredMiddleware(http.HandlerFunc(bestiaryHandler.SubmitCreatureGenerationPrompt)),
	).Methods("POST")
}
