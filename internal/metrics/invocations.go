package metrics

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
	"sync/atomic"
)

const invocations = "lambda.function.invocation"

type invocationsCounter struct {
	invocations int64
}

func (ic *invocationsCounter) invoked() {
	atomic.AddInt64(&ic.invocations, 1)
}

func (ic *invocationsCounter) counter() *datapoint.Datapoint {
	return sfxclient.Counter(
		invocations,
		nil,
		atomic.SwapInt64(&ic.invocations, 0),
	)
}

func (ic *invocationsCounter) Datapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		ic.counter(),
	}
}
