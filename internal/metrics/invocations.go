package metrics

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
)

const invocations = "lambda.function.invocation"

type invocationsCounter struct {
	invocations int64
}

func (ic *invocationsCounter) invoked() {
	ic.invocations++
}

func (ic *invocationsCounter) counter() *datapoint.Datapoint {
	defer func() { ic.invocations = 0 }()
	return sfxclient.Counter(
		invocations,
		nil,
		ic.invocations,
	)
}

func (ic *invocationsCounter) Datapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		ic.counter(),
	}
}
