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
	if err := prometheus.Register(metrics.totalHits); err != nil {
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

func (m *DBMetrics) IncreaseDuration(function string, duration *time.Duration) {
	m.duration.WithLabelValues(m.dbName, function).Observe(duration.Seconds())
}
