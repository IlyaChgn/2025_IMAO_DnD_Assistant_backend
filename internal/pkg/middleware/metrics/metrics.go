package metrics

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

func CreateMetricsMiddleware(m metrics.HTTPMetrics) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fw := &fakeResponseWriter{
				ResponseWriter: w,
				code:           200,
			}
			start := time.Now()

			next.ServeHTTP(fw, r)

			duration := time.Since(start)
			code := fw.code
			route := mux.CurrentRoute(r)
			path, _ := route.GetPathTemplate()

			if path != "/metrics" {
				m.IncreaseHits(path, code)
				m.IncreaseDuration(path, code, duration)
			}
		})
	}
}

type fakeResponseWriter struct {
	http.ResponseWriter
	code int
}

func (fw *fakeResponseWriter) WriteHeader(code int) {
	fw.code = code
	fw.ResponseWriter.WriteHeader(code)
}
