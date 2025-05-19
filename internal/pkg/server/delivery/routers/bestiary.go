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

	subrouterLoginRequired := subrouter.PathPrefix("").Subrouter()
	subrouterLoginRequired.Use(loginRequiredMiddleware)

	subrouterLoginRequired.HandleFunc("/generated_creature", bestiaryHandler.AddGeneratedCreature).
		Methods("POST")
	subrouterLoginRequired.HandleFunc("/statblock-image", bestiaryHandler.UploadCreatureStatblockImage).
		Methods("POST")
	subrouterLoginRequired.HandleFunc("/creature-generation-prompt", bestiaryHandler.SubmitCreatureGenerationPrompt).
		Methods("POST")

	subrouterLoginRequired.HandleFunc("/usr_content/list", bestiaryHandler.GetUserCreaturesList).
		Methods("POST")
	subrouterLoginRequired.HandleFunc("/usr_content/{name}", bestiaryHandler.GetUserCreatureByName).
		Methods("GET")
}
