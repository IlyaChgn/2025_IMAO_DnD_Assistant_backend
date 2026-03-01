package router

import (
	characterdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/delivery"
	"github.com/gorilla/mux"
)

func ServeCharacterBaseRouter(router *mux.Router, handler *characterdel.CharacterBaseHandler,
	loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/characters").Subrouter()
	subrouter.Use(loginRequiredMiddleware)

	subrouter.HandleFunc("", handler.List).Methods("GET")
	subrouter.HandleFunc("", handler.Create).Methods("POST")
	subrouter.HandleFunc("/import/lss", handler.ImportLSS).Methods("POST")
	subrouter.HandleFunc("/{id}/computed", handler.GetComputed).Methods("GET")
	subrouter.HandleFunc("/{id}/avatar", handler.UploadAvatar).Methods("POST")
	subrouter.HandleFunc("/{id}/avatar", handler.DeleteAvatar).Methods("DELETE")
	subrouter.HandleFunc("/{id}/hotbar-layout", handler.GetHotbarLayout).Methods("GET")
	subrouter.HandleFunc("/{id}/hotbar-layout", handler.SetHotbarLayout).Methods("PUT")
	subrouter.HandleFunc("/{id}", handler.GetByID).Methods("GET")
	subrouter.HandleFunc("/{id}", handler.Update).Methods("PUT")
	subrouter.HandleFunc("/{id}", handler.Delete).Methods("DELETE")
}
