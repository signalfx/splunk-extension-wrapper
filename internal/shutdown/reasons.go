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

package shutdown

const (
	internalError = "internal"
	apiError      = "api"
	metricError   = "metric"
)

type Condition interface {
	Reason() string
	Message() string
	IsError() bool
}

type simple struct {
	reason  string
	message string
	error   bool
}

func newWithError(message, reason string) *simple {
	return &simple{message: message, reason: reason, error: true}
}

func (s simple) Reason() string {
	return s.reason
}

func (s simple) Message() string {
	return s.message
}

func (s simple) IsError() bool {
	return s.error
}

func Api(message string) Condition {
	return newWithError(message, apiError)
}

func Internal(message string) Condition {
	return newWithError(message, internalError)
}

func Metric(message string) Condition {
	return newWithError(message, metricError)
}

func Reason(reason string) Condition {
	return simple{reason: reason}
}
