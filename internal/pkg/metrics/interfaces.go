package metrics

import "time"

type DBMetrics interface {
	IncreaseHits(function string)
	IncreaseErrs(function string)
	IncreaseDuration(function string, duration time.Duration)
}

type HTTPMetrics interface {
	IncreaseHits(path string, code int)
	IncreaseDuration(path string, code int, duration time.Duration)
}
