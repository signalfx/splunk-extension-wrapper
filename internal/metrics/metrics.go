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
