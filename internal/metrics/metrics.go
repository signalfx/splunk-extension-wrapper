package metrics

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
	"log"
	"sync/atomic"
	"time"
)

const invocations = "lambda.function.invocation"
const environmentStart = "lambda.function.initialization"
const environmentShutdown = "lambda.function.shutdown"
const environmentLifetime = "lambda.function.lifetime"

const dimShutdownCause = "cause"
const dimRegion = "aws_region"
const dimAccountId = "aws_account_id"
const dimFunctionName = "aws_function_name"
const dimFunctionVersion = "aws_function_version"
const dimArn = "aws_arn"
const dimQualifier = "aws_function_qualifier"
const dimRuntime = "aws_function_runtime"
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
	dp := sfxclient.Counter(environmentStart, nil, 1)
	dp.Timestamp = m.startTime
	return dp
}

func (m metrics) endCounter(cause string) *datapoint.Datapoint {
	dp := sfxclient.Counter(environmentShutdown, map[string]string{dimShutdownCause: cause}, 1)
	dp.Timestamp = m.endTime
	return dp
}

func (m metrics) envDuration() *datapoint.Datapoint {
	dur := m.endTime.Sub(m.startTime)
	dp := sfxclient.Gauge(environmentLifetime, nil, dur.Milliseconds())
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

func (m *metrics) Datapoints() []*datapoint.Datapoint {
	dps := []*datapoint.Datapoint{
		m.invocationsCounter(),
	}

	for {
		select {
		case dp := <-m.adhocDps:
			log.Printf("drainig adhoc dps: %v", dp)
			dps = append(dps, dp)
		default:
			log.Printf("nothing to drain...")
			return dps
		}
	}
}
