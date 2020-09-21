package metrics

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
	"log"
	"sync/atomic"
	"time"
)

const invocations = "lambda.extension.function.invocation"
const sandboxStart = "lambda.extension.environment.initialization"
const sandboxEnd = "lambda.extension.environment.shutdown"
const sandboxDuration = "lambda.extension.environment.duration"
const sandboxActive = "lambda.extension.environment.active"

const dimShutdownCause = "cause"
const dimFunctionName = "name"
const dimFunctionVersion = "version"
const dimAwsUniqueId = "AWSUniqueId"

type metrics struct {
	adhocDps chan *datapoint.Datapoint

	startTime time.Time
	endTime   time.Time

	invocations int64
}

func newMetrics() metrics {
	return metrics{
		adhocDps: make(chan *datapoint.Datapoint, 3),
	}
}

func (m *metrics) markStart() {
	m.startTime = time.Now()
	m.adhocDps <- m.startCounter()
}

func (m *metrics) markEnd(cause string) {
	m.endTime = time.Now()
	m.adhocDps <- m.endCounter(cause)
	m.adhocDps <- m.envDuration()
}

func (m *metrics) Invoked() {
	atomic.AddInt64(&m.invocations, 1)
}

func (m metrics) startCounter() *datapoint.Datapoint {
	dp := sfxclient.Counter(sandboxStart, nil, 1)
	dp.Timestamp = m.startTime
	return dp
}

func (m metrics) endCounter(cause string) *datapoint.Datapoint {
	dp := sfxclient.Counter(sandboxEnd, map[string]string{dimShutdownCause: cause}, 1)
	dp.Timestamp = m.endTime
	return dp
}

func (m metrics) envDuration() *datapoint.Datapoint {
	dur := m.endTime.Sub(m.startTime)
	dp := sfxclient.Gauge(sandboxDuration, nil, dur.Milliseconds())
	dp.Timestamp = m.endTime
	return dp
}

func (m *metrics) invocationsCounter() *datapoint.Datapoint {
	return sfxclient.Counter(
		invocations,
		nil,
		atomic.SwapInt64(&m.invocations, 0),
	)
}

func activeCounter() *datapoint.Datapoint {
	return sfxclient.Gauge(sandboxActive, nil, 1)
}

func (m *metrics) Datapoints() []*datapoint.Datapoint {
	dps := []*datapoint.Datapoint{
		m.invocationsCounter(),
		activeCounter(),
	}

DRAIN:
	for {
		select {
		case dp := <-m.adhocDps:
			log.Printf("drainig adhoc dps: %v", dp)
			dps = append(dps, dp)
		default:
			log.Printf("nothing to drain...")
			break DRAIN
		}
	}

	return dps
}
