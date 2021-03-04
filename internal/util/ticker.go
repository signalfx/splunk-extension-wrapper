package util

import (
	"github.com/splunk/lambda-extension/internal/config"
	"time"
)

type Ticker interface {
	Tick() bool
}

// According to AWS Lambda execution environment docs:
// Lambda freezes the execution environment when runtime and each extension has completed
//   and there are no pending events.
//
// The extension should only send metrics once after unfreeze and
// this ticker does not accumulate ticks when the execution environment is frozen.
type lossyTicker struct {
	tickAfter time.Time
	interval  time.Duration
	clock     clock
}

type ticksAlways struct{}

func NewTicker(config config.Configuration) Ticker {
	if config.FastIngest {
		return &ticksAlways{}
	}
	return newIntervalTicker(config.ReportingDelay, systemClock{})
}

func newIntervalTicker(interval time.Duration, clock clock) Ticker {
	return &lossyTicker{
		interval: interval,
		clock:    clock,
	}
}

func (ticker ticksAlways) Tick() bool {
	return true
}

func (ticker *lossyTicker) Tick() bool {
	now := ticker.clock.now()

	defer ticker.skipToNearest(now)

	if ticker.tickAfter.IsZero() {
		ticker.tickAfter = now
		return true
	}

	return now.After(ticker.tickAfter)
}

func (ticker *lossyTicker) skipToNearest(now time.Time) {
	if !ticker.tickAfter.After(now) {
		ticker.tickAfter = now.Add(ticker.interval)
	}
}
