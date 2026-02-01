package usecases

import (
	"time"

	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
)

const sessionStringLen = 32

// randSessionIDGen is the production SessionIDGenerator.
type randSessionIDGen struct{}

func NewRandSessionIDGen() *randSessionIDGen { return &randSessionIDGen{} }

func (g *randSessionIDGen) NewSessionID() string { return utils.RandString(sessionStringLen) }

// realTimerFactory is the production TimerFactory wrapping time.AfterFunc.
type realTimerFactory struct{}

func NewRealTimerFactory() *realTimerFactory { return &realTimerFactory{} }

func (f *realTimerFactory) AfterFunc(d time.Duration, fn func()) tableinterfaces.SessionTimer {
	return &realTimer{timer: time.AfterFunc(d, fn)}
}

type realTimer struct {
	timer *time.Timer
}

func (t *realTimer) Stop() bool                 { return t.timer.Stop() }
func (t *realTimer) Reset(d time.Duration) bool { return t.timer.Reset(d) }
