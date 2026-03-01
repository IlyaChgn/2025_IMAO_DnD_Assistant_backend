package router

import (
	spellsdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells/delivery"
	"github.com/gorilla/mux"
)

func ServeSpellsRouter(router *mux.Router, handler *spellsdel.SpellsHandler) {
	sub := router.PathPrefix("/reference/spells").Subrouter()

	// NOTE: route order matters — /by-class/{className} must come before /{id}
	sub.HandleFunc("/by-class/{className}", handler.GetSpellsByClass).Methods("GET")
	sub.HandleFunc("/{id}", handler.GetSpellByID).Methods("GET")
	sub.HandleFunc("", handler.GetSpells).Methods("GET")
}
