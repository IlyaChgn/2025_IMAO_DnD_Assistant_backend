package reqdata

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"net/http"
)

func RequestDataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctx = utils.SaveRequestData(ctx, r)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
