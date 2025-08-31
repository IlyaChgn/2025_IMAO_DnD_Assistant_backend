package recover

import (
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"net/http"

	responses "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				ctx := request.Context()
				l := logger.FromContext(ctx)

				var err error
				switch x := r.(type) {
				case string:
					err = fmt.Errorf("%s", x)
				case error:
					err = x
				default:
					err = fmt.Errorf("%#v", x)
				}

				l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer,
					err, nil)
				http.Error(writer, responses.ErrInternalServer, responses.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(writer, request)
	})
}
