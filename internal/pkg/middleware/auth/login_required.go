package auth

import (
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/gorilla/mux"
	"net/http"
)

func LoginRequiredMiddleware(uc authinterface.AuthUsecases) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			session, _ := r.Cookie("session_id")
			if session == nil {
				responses.SendErrResponse(w, responses.StatusUnauthorized, responses.ErrNotAuthorized)

				return
			}

			_, isAuth := uc.CheckAuth(ctx, session.Value)
			if !isAuth {
				responses.SendErrResponse(w, responses.StatusUnauthorized, responses.ErrNotAuthorized)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
