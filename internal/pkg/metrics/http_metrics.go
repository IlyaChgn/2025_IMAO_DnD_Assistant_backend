package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

type httpMetrics struct {
	totalHits *prometheus.CounterVec
	duration  *prometheus.HistogramVec
}

func NewHTTPMetrics() (HTTPMetrics, error) {
	var metrics httpMetrics

	metrics.totalHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "total_hits_count",
			Help: "Number of total http requests",
		},
		[]string{"path", "code"})
	if err := prometheus.Register(metrics.totalHits); err != nil {
		return nil, err
	}

	metrics.duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_time",
			Help:    "Request time",
			Buckets: []float64{0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
		},
		[]string{"path", "code"})
	if err := prometheus.Register(metrics.duration); err != nil {
		return nil, err
	}

	return &metrics, nil
}

func (m *httpMetrics) IncreaseHits(path string, code int) {
	m.totalHits.WithLabelValues(path, strconv.Itoa(code)).Inc()
}

func (m *httpMetrics) IncreaseDuration(path string, code int, duration time.Duration) {
	m.duration.WithLabelValues(path, strconv.Itoa(code)).Observe(duration.Seconds())
}
