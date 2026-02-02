package router

import (
	authdel "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth/delivery"
	"github.com/gorilla/mux"
)

func ServeAuthRouter(router *mux.Router, authHandler *authdel.AuthHandler, loginRequiredMiddleware mux.MiddlewareFunc) {
	subrouter := router.PathPrefix("/auth").Subrouter()

	subrouterLoginRequired := subrouter.PathPrefix("").Subrouter()
	subrouterLoginRequired.Use(loginRequiredMiddleware)
	subrouterLoginRequired.HandleFunc("/logout", authHandler.Logout).Methods("POST")
	subrouterLoginRequired.HandleFunc("/identities", authHandler.ListIdentities).Methods("GET")
	subrouterLoginRequired.HandleFunc("/link/{provider}", authHandler.LinkIdentity).Methods("POST")
	subrouterLoginRequired.HandleFunc("/unlink/{provider}", authHandler.UnlinkIdentity).Methods("DELETE")

	subrouter.HandleFunc("/login/{provider}", authHandler.Login).Methods("POST")
	subrouter.HandleFunc("/check", authHandler.CheckAuth).Methods("GET")
}
