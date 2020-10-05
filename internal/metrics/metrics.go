package metrics

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
	"time"
)

const environmentStart = "lambda.function.initialization"
const environmentStartDuration = "lambda.function.initialization.latency"
const environmentShutdown = "lambda.function.shutdown"
const environmentLifetime = "lambda.function.lifetime"

type environmentMetrics struct {
	adhocDps []*datapoint.Datapoint

	startTime       time.Time
	firstInvocation time.Time
	endTime         time.Time
}

func (em *environmentMetrics) markStart() {
	em.startTime = time.Now()
	em.adhocDps = append(em.adhocDps, em.startCounter())
}

func (em *environmentMetrics) markFirstInvocation() {
	em.firstInvocation = time.Now()
	em.adhocDps = append(em.adhocDps, em.startLatency())
}

func (em *environmentMetrics) markEnd(cause string) {
	em.endTime = time.Now()
	em.adhocDps = append(em.adhocDps, em.endCounter(cause), em.envDuration())
}

func (em environmentMetrics) startCounter() *datapoint.Datapoint {
	return sfxclient.Counter(environmentStart, nil, 1)
}

func (em environmentMetrics) startLatency() *datapoint.Datapoint {
	dur := em.firstInvocation.Sub(em.startTime)
	return sfxclient.Gauge(environmentStartDuration, nil, dur.Milliseconds())
}

func (em environmentMetrics) endCounter(cause string) *datapoint.Datapoint {
	return sfxclient.Counter(environmentShutdown, map[string]string{dimShutdownCause: cause}, 1)
}

func (em environmentMetrics) envDuration() *datapoint.Datapoint {
	dur := em.endTime.Sub(em.startTime)
	return sfxclient.Gauge(environmentLifetime, nil, dur.Milliseconds())
}

func (em *environmentMetrics) Datapoints() []*datapoint.Datapoint {
	defer func() { em.adhocDps = nil }()
	return em.adhocDps
}
