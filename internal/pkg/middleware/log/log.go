package log

import (
	mylogger "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/gorilla/mux"
	"net/http"
)

func CreateLogMiddleware(logger mylogger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := logger.WithContext(r.Context())
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
