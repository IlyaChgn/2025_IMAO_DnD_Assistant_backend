package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type wsMetrics struct {
	sessions *prometheus.CounterVec
	conns    *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

type wsSessionMetrics struct {
	receivedMsgs *prometheus.CounterVec
	sentMsgs     *prometheus.CounterVec
}

func NewWSMetrics() (WSMetrics, error) {
	var metrics wsMetrics

	metrics.sessions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_sessions_count",
			Help: "Total WS session numbers",
		},
		[]string{})
	if err := prometheus.Register(metrics.sessions); err != nil {
		return nil, err
	}

	metrics.conns = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_conns_count",
			Help: "Number of WS connections",
		},
		[]string{})
	if err := prometheus.Register(metrics.conns); err != nil {
		return nil, err
	}

	metrics.duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ws_session_duration",
			Help:    "Websocket session duration",
			Buckets: []float64{15, 20, 25, 30, 45, 60, 90},
		},
		[]string{})
	if err := prometheus.Register(metrics.duration); err != nil {
		return nil, err
	}

	return &metrics, nil
}

func NewWSSessionMetrics() (WSSessionMetrics, error) {
	var metrics wsSessionMetrics

	metrics.receivedMsgs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_received_msgs_count",
			Help: "Number of received messages in WS connections",
		},
		[]string{})
	if err := prometheus.Register(metrics.receivedMsgs); err != nil {
		return nil, err
	}

	metrics.sentMsgs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_sent_msgs_count",
			Help: "Number of sent messages in WS connections",
		},
		[]string{})
	if err := prometheus.Register(metrics.sentMsgs); err != nil {
		return nil, err
	}

	return &metrics, nil
}

func (m *wsMetrics) IncSessions() {
	m.sessions.WithLabelValues().Inc()
}

func (m *wsMetrics) IncConns() {
	m.conns.WithLabelValues().Inc()
}

func (m *wsSessionMetrics) IncReceivedMsgs() {
	m.receivedMsgs.WithLabelValues().Inc()
}

func (m *wsSessionMetrics) IncSentMsgs() {
	m.sentMsgs.WithLabelValues().Inc()
}

func (m *wsMetrics) IncreaseDuration(duration time.Duration) {
	m.duration.WithLabelValues().Observe(duration.Minutes())
}
