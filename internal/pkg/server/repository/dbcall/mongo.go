package dbcall

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"time"
)

func MongoCall[T any](fnName string, m *metrics.DBMetrics, dbFn func() (T, error)) (T, error) {
	m.IncreaseHits(fnName)
	start := time.Now()

	result, err := dbFn()
	m.IncreaseDuration(fnName, time.Since(start))

	if err != nil {
		m.IncreaseErrs(fnName)
	}

	return result, err
}
