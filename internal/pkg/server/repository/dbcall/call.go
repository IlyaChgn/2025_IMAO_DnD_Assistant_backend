package dbcall

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"time"
)

func DBCall[T any](fnName string, m *metrics.DBMetrics, dbFn func() (T, error)) (T, error) {
	m.IncreaseHits(fnName)

	start := time.Now()
	defer m.IncreaseDuration(fnName, time.Since(start))

	result, err := dbFn()

	if err != nil {
		m.IncreaseErrs(fnName)
	}

	return result, err
}

func ErrOnlyDBCall(fnName string, m *metrics.DBMetrics, dbFn func() error) error {
	_, err := DBCall[any](fnName, m, func() (any, error) {
		err := dbFn()

		return nil, err
	})

	return err
}
