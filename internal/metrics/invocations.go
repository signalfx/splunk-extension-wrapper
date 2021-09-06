// Copyright Splunk Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
