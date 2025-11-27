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

package otelmetrics

import (
	"context"
	"time"
)

// TelemetryMetricsSink defines the interface for recording Lambda telemetry metrics.
// This interface is called by the telemetry subscriber to update metrics
// based on events from the AWS Lambda Telemetry API.
//
// This is a compatibility interface that wraps the new MetricsSink implementation.
type TelemetryMetricsSink interface {
	// RecordInitStart is called when platform.initStart event is received
	// initializationType is "on-demand" for cold starts or "snap-start" for warm starts
	RecordInitStart(ctx context.Context, timestamp time.Time, initializationType string)

	// RecordInitEnd is called when platform.initEnd event is received
	// initializationType is "on-demand" for cold starts or "snap-start" for warm starts
	RecordInitEnd(ctx context.Context, timestamp time.Time, initializationType string)

	// RecordStart is called when platform.start event is received
	RecordStart(ctx context.Context, timestamp time.Time, requestID string)

	// RecordRuntimeDone is called when platform.runtimeDone event is received
	// durationMs is the invocation duration in milliseconds
	// producedBytes is the response payload size in bytes
	// status indicates success, failure, or timeout
	RecordRuntimeDone(ctx context.Context, timestamp time.Time, requestID string, status string, durationMs int64, producedBytes int64)

	// RecordReport is called when platform.report event is received with detailed metrics
	RecordReport(ctx context.Context, timestamp time.Time, requestID string, metrics ReportMetrics)

	// RecordShutdown is called when platform.shutdown event is received
	// reason is the shutdown reason (e.g., "spindown", "timeout", "failure")
	RecordShutdown(ctx context.Context, timestamp time.Time, reason string)
}

// Note: The MetricsSink implementation has been moved to metrics_sink.go
// This file only contains the interface definition for backward compatibility.

