package metrics

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
	"log"
	"time"
)

const environmentStart = "lambda.function.initialization"
const environmentShutdown = "lambda.function.shutdown"
const environmentLifetime = "lambda.function.lifetime"

type environmentMetrics struct {
	adhocDps chan *datapoint.Datapoint

	startTime time.Time
	endTime   time.Time
}

func newEnvironmentMetrics() environmentMetrics {
	return environmentMetrics{
		adhocDps: make(chan *datapoint.Datapoint, 3),
	}
}

func (em *environmentMetrics) markStart() {
	em.startTime = time.Now()
	em.adhocDps <- em.startCounter()
}

func (em *environmentMetrics) markEnd(cause string) {
	em.endTime = time.Now()
	em.adhocDps <- em.endCounter(cause)
	em.adhocDps <- em.envDuration()
}

func (em environmentMetrics) startCounter() *datapoint.Datapoint {
	dp := sfxclient.Counter(environmentStart, nil, 1)
	dp.Timestamp = em.startTime
	return dp
}

func (em environmentMetrics) endCounter(cause string) *datapoint.Datapoint {
	dp := sfxclient.Counter(environmentShutdown, map[string]string{dimShutdownCause: cause}, 1)
	dp.Timestamp = em.endTime
	return dp
}

func (em environmentMetrics) envDuration() *datapoint.Datapoint {
	dur := em.endTime.Sub(em.startTime)
	dp := sfxclient.Gauge(environmentLifetime, nil, dur.Milliseconds())
	dp.Timestamp = em.endTime
	return dp
}

func (em *environmentMetrics) Datapoints() []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint

	for {
		select {
		case dp := <-em.adhocDps:
			log.Printf("drainig adhoc dps: %v", dp)
			dps = append(dps, dp)
		default:
			log.Printf("nothing to drain...")
			return dps
		}
	}
}
