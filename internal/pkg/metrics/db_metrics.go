package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type DBMetrics struct {
	dbName    string
	totalHits *prometheus.CounterVec
	totalErrs *prometheus.CounterVec
	duration  *prometheus.HistogramVec
}

func NewDBMetrics(dbName string) (*DBMetrics, error) {
	var metrics DBMetrics

	metrics.dbName = dbName

	metrics.totalHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_total_hits_count", dbName),
			Help: fmt.Sprintf("Number of total %s hits", dbName),
		},
		[]string{"dbName", "function"})
	if err := prometheus.Register(metrics.totalHits); err != nil {
		return nil, err
	}

	metrics.totalErrs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_total_errors_count", dbName),
			Help: fmt.Sprintf("Number of total %s errors", dbName),
		},
		[]string{"dbName", "function"})
	if err := prometheus.Register(metrics.totalErrs); err != nil {
		return nil, err
	}

	metrics.duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_duration", dbName),
			Help:    fmt.Sprintf("Request time in %s DB", dbName),
			Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"dbName", "function"})
	if err := prometheus.Register(metrics.duration); err != nil {
		return nil, err
	}

	return &metrics, nil
}

func (m *DBMetrics) IncreaseHits(function string) {
	m.totalHits.WithLabelValues(m.dbName, function).Inc()
}

func (m *DBMetrics) IncreaseErrs(function string) {
	m.totalErrs.WithLabelValues(m.dbName, function).Inc()
}

func (m *DBMetrics) IncreaseDuration(function string, duration time.Duration) {
	m.duration.WithLabelValues(m.dbName, function).Observe(duration.Seconds())
}
