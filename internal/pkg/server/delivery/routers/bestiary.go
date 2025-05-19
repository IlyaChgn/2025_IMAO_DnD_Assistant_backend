package router

import (
	bestiarydel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/delivery"
	"github.com/gorilla/mux"
)

func ServeBestiaryRouter(router *mux.Router, bestiaryHandler *bestiarydel.BestiaryHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/bestiary").Subrouter()

	subrouter.HandleFunc("/list", bestiaryHandler.GetCreaturesList).Methods("POST")
	subrouter.HandleFunc("/{name}", bestiaryHandler.GetCreatureByName).Methods("GET")
	subrouter.HandleFunc("/generated_creature", bestiaryHandler.AddGeneratedCreature).Methods("POST")
	subrouter.HandleFunc("/statblock-image", bestiaryHandler.UploadCreatureStatblockImage).
		Methods("POST")
	subrouter.HandleFunc("/creature-generation-prompt", bestiaryHandler.SubmitCreatureGenerationPrompt).
		Methods("POST")

	subrouterLoginRequired := subrouter.PathPrefix("/usr_content").Subrouter()
	subrouterLoginRequired.Use(loginRequiredMiddleware)

	subrouterLoginRequired.HandleFunc("/list", bestiaryHandler.GetUserCreaturesList).Methods("POST")
	subrouterLoginRequired.HandleFunc("/{name}", bestiaryHandler.GetUserCreatureByName).Methods("GET")
}
