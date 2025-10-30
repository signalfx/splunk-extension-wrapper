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
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// MetricsSink handles recording of Lambda telemetry events to OpenTelemetry metrics.
// It maintains state to compute durations and deltas for proper metric recording.
type MetricsSink struct {
	// Instruments
	invocation              metric.Int64Counter
	initialization          metric.Int64Counter
	initializationLatency   metric.Int64UpDownCounter
	shutdown                metric.Int64Counter
	lifetime                metric.Int64UpDownCounter
	coldStarts              metric.Int64Counter
	warmStarts              metric.Int64Counter
	responseSize            metric.Int64Histogram
	snapStartRestoreDuration metric.Float64Histogram

	// FaaS semantic convention instruments (optional)
	emitSemconv        bool
	faasInvocations    metric.Int64Counter
	faasErrors         metric.Int64Counter
	faasTimeouts       metric.Int64Counter
	faasInitDuration   metric.Float64Histogram
	faasInvokeDuration metric.Float64Histogram
	faasMemUsage       metric.Float64Histogram

	// State tracking
	mu                 sync.RWMutex
	initStartTime      time.Time
	initEndTime        time.Time
	invokeStartTime    time.Time
	lastLifetimeMs     int64
	environmentStarted bool
}

// MetricsSinkOption configures MetricsSink
type MetricsSinkOption func(*MetricsSink)

// WithSemconv enables FaaS semantic convention metrics
func WithSemconv(enabled bool) MetricsSinkOption {
	return func(ms *MetricsSink) {
		ms.emitSemconv = enabled
	}
}

// NewMetricsSink creates a new MetricsSink with the given meter and options.
// It creates all necessary metric instruments and returns a ready-to-use sink.
func NewMetricsSink(meter metric.Meter, opts ...MetricsSinkOption) (*MetricsSink, error) {
	if meter == nil {
		return nil, fmt.Errorf("meter cannot be nil")
	}

	ms := &MetricsSink{
		emitSemconv: shouldEmitSemconvMetrics(),
	}

	// Apply options
	for _, opt := range opts {
		opt(ms)
	}

	var err error

	// Create Lambda-specific instruments
	ms.invocation, err = meter.Int64Counter(
		"lambda.function.invocation",
		metric.WithDescription("Number of Lambda function invocations"),
		metric.WithUnit("{invocation}"),
	)
	if err != nil {
		return nil, err
	}

	ms.initialization, err = meter.Int64Counter(
		"lambda.function.initialization",
		metric.WithDescription("Number of Lambda environment initializations (cold starts)"),
		metric.WithUnit("{initialization}"),
	)
	if err != nil {
		return nil, err
	}

	ms.initializationLatency, err = meter.Int64UpDownCounter(
		"lambda.function.initialization.latency",
		metric.WithDescription("Lambda cold start initialization latency"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	ms.shutdown, err = meter.Int64Counter(
		"lambda.function.shutdown",
		metric.WithDescription("Number of Lambda environment shutdowns"),
		metric.WithUnit("{shutdown}"),
	)
	if err != nil {
		return nil, err
	}

	ms.lifetime, err = meter.Int64UpDownCounter(
		"lambda.function.lifetime",
		metric.WithDescription("Total lifetime of Lambda environment"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	ms.coldStarts, err = meter.Int64Counter(
		"lambda.function.cold_starts",
		metric.WithDescription("Number of cold starts (on-demand initialization)"),
		metric.WithUnit("{cold_start}"),
	)
	if err != nil {
		return nil, err
	}

	ms.warmStarts, err = meter.Int64Counter(
		"lambda.function.warm_starts",
		metric.WithDescription("Number of warm starts (snap-start initialization)"),
		metric.WithUnit("{warm_start}"),
	)
	if err != nil {
		return nil, err
	}

	ms.responseSize, err = meter.Int64Histogram(
		"lambda.function.response_size",
		metric.WithDescription("Lambda function response payload size"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	ms.snapStartRestoreDuration, err = meter.Float64Histogram(
		"lambda.function.snapstart.restore_duration",
		metric.WithDescription("SnapStart restore duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	// Create FaaS semantic convention instruments if enabled
	if ms.emitSemconv {
		if err := ms.createSemconvInstruments(meter); err != nil {
			log.Printf("[WARN] Failed to create FaaS semconv instruments: %v", err)
			ms.emitSemconv = false
		} else {
			// Log to stderr so it always appears in CloudWatch logs
			fmt.Fprintln(os.Stderr, "[splunk-extension-wrapper] [INFO] FaaS semantic convention metrics enabled in MetricsSink")
		}
	}

	return ms, nil
}

// createSemconvInstruments creates FaaS semantic convention metric instruments
func (ms *MetricsSink) createSemconvInstruments(meter metric.Meter) error {
	var err error

	ms.faasInvocations, err = meter.Int64Counter(
		"faas.invocations",
		metric.WithDescription("Number of FaaS invocations"),
		metric.WithUnit("{invocation}"),
	)
	if err != nil {
		return err
	}

	ms.faasErrors, err = meter.Int64Counter(
		"faas.errors",
		metric.WithDescription("Number of FaaS invocation errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	ms.faasTimeouts, err = meter.Int64Counter(
		"faas.timeouts",
		metric.WithDescription("Number of FaaS invocation timeouts"),
		metric.WithUnit("{timeout}"),
	)
	if err != nil {
		return err
	}

	ms.faasInitDuration, err = meter.Float64Histogram(
		"faas.init_duration",
		metric.WithDescription("FaaS function initialization duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	ms.faasInvokeDuration, err = meter.Float64Histogram(
		"faas.duration",
		metric.WithDescription("FaaS function invocation duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	ms.faasMemUsage, err = meter.Float64Histogram(
		"faas.mem_usage",
		metric.WithDescription("FaaS function memory usage"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	return nil
}

// RecordInitStart records when Lambda environment initialization starts
// initializationType is "on-demand" for cold starts or "snap-start" for warm starts
func (ms *MetricsSink) RecordInitStart(ctx context.Context, timestamp time.Time, initializationType string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.initStartTime = timestamp
	ms.environmentStarted = true
}

// RecordInitEnd records when Lambda environment initialization completes
// It computes the initialization duration and records relevant metrics.
// initializationType is "on-demand" for cold starts or "snap-start" for warm starts
func (ms *MetricsSink) RecordInitEnd(ctx context.Context, timestamp time.Time, initializationType string) {
	ms.mu.Lock()
	initStartTime := ms.initStartTime
	ms.initEndTime = timestamp
	ms.mu.Unlock()

	if initStartTime.IsZero() {
		log.Printf("[WARN] RecordInitEnd called but initStartTime not set")
		return
	}

	// Compute initialization duration
	durationMs := timestamp.Sub(initStartTime).Milliseconds()
	durationSeconds := float64(durationMs) / 1000.0

	// Increment initialization counter
	ms.initialization.Add(ctx, 1)

	// Record initialization latency (as delta to UpDownCounter)
	ms.initializationLatency.Add(ctx, durationMs)

	// Record cold/warm start based on initialization type
	if initializationType == "snap-start" {
		ms.warmStarts.Add(ctx, 1)
		log.Printf("[DEBUG] Warm start (snap-start): init duration %dms", durationMs)
	} else {
		// Default to cold start for "on-demand" or any other type
		ms.coldStarts.Add(ctx, 1)
		log.Printf("[DEBUG] Cold start (on-demand): init duration %dms", durationMs)
	}

	// Record FaaS init duration if enabled
	if ms.emitSemconv && ms.faasInitDuration != nil {
		ms.faasInitDuration.Record(ctx, durationSeconds)
	}

	log.Printf("[DEBUG] Init duration: %dms (%.3fs)", durationMs, durationSeconds)
}

// RecordStart records when a Lambda invocation starts
func (ms *MetricsSink) RecordStart(ctx context.Context, timestamp time.Time, requestID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.invokeStartTime = timestamp
}

// RecordRuntimeDone records when a Lambda invocation completes
// It increments appropriate counters based on the invocation status.
// producedBytes is the response payload size in bytes
func (ms *MetricsSink) RecordRuntimeDone(ctx context.Context, timestamp time.Time, requestID string, status string, durationMs int64, producedBytes int64) {
	// Always increment invocation counter
	ms.invocation.Add(ctx, 1)

	// Record response size if present
	if producedBytes > 0 {
		ms.responseSize.Record(ctx, producedBytes)
		log.Printf("[DEBUG] Response size: %d bytes", producedBytes)
	}

	// Handle status-specific metrics
	switch status {
	case "success":
		// Increment semconv invocations on success
		if ms.emitSemconv && ms.faasInvocations != nil {
			ms.faasInvocations.Add(ctx, 1)
		}

	case "error", "failure":
		// Increment both invocations and errors
		if ms.emitSemconv {
			if ms.faasInvocations != nil {
				ms.faasInvocations.Add(ctx, 1)
			}
			if ms.faasErrors != nil {
				ms.faasErrors.Add(ctx, 1)
			}
		}

	case "timeout":
		// Increment both invocations and timeouts
		if ms.emitSemconv {
			if ms.faasInvocations != nil {
				ms.faasInvocations.Add(ctx, 1)
			}
			if ms.faasTimeouts != nil {
				ms.faasTimeouts.Add(ctx, 1)
			}
		}
	}

	// Record invocation duration if we have start time
	ms.mu.RLock()
	invokeStartTime := ms.invokeStartTime
	ms.mu.RUnlock()

	if !invokeStartTime.IsZero() && durationMs > 0 {
		durationSeconds := float64(durationMs) / 1000.0
		if ms.emitSemconv && ms.faasInvokeDuration != nil {
			ms.faasInvokeDuration.Record(ctx, durationSeconds)
		}
	}

	log.Printf("[DEBUG] RuntimeDone: requestID=%s, status=%s, duration=%dms", requestID, status, durationMs)
}

// RecordReport records metrics from the platform.report event
// It handles lifetime gauge emulation and memory usage recording.
func (ms *MetricsSink) RecordReport(ctx context.Context, timestamp time.Time, requestID string, metrics ReportMetrics) {
	ms.mu.Lock()
	initStartTime := ms.initStartTime
	lastLifetimeMs := ms.lastLifetimeMs
	ms.mu.Unlock()

	// Compute current lifetime
	if !initStartTime.IsZero() {
		currentLifetimeMs := timestamp.Sub(initStartTime).Milliseconds()
		
		// Compute delta for lifetime gauge emulation
		deltaLifetimeMs := currentLifetimeMs - lastLifetimeMs
		if deltaLifetimeMs > 0 {
			ms.lifetime.Add(ctx, deltaLifetimeMs)
			
			ms.mu.Lock()
			ms.lastLifetimeMs = currentLifetimeMs
			ms.mu.Unlock()
			
			log.Printf("[DEBUG] Lifetime: current=%dms, delta=%dms", currentLifetimeMs, deltaLifetimeMs)
		}
	}

	// Record invocation duration from report if available
	if metrics.DurationMs > 0 && ms.emitSemconv && ms.faasInvokeDuration != nil {
		durationSeconds := metrics.DurationMs / 1000.0
		ms.faasInvokeDuration.Record(ctx, durationSeconds)
	}

	// Record memory usage if available
	if metrics.MaxMemoryUsedMB > 0 && ms.emitSemconv && ms.faasMemUsage != nil {
		// Convert MB to bytes
		memoryBytes := float64(metrics.MaxMemoryUsedMB * 1024 * 1024)
		ms.faasMemUsage.Record(ctx, memoryBytes)
		log.Printf("[DEBUG] Memory usage: %dMB (%.0f bytes)", metrics.MaxMemoryUsedMB, memoryBytes)
	}

	// Record SnapStart restore duration if available
	if metrics.RestoreDurationMs > 0 {
		ms.snapStartRestoreDuration.Record(ctx, metrics.RestoreDurationMs)
		log.Printf("[DEBUG] SnapStart restore duration: %.2fms", metrics.RestoreDurationMs)
	}
}

// RecordShutdown records when the Lambda environment is shutting down
func (ms *MetricsSink) RecordShutdown(ctx context.Context, timestamp time.Time, reason string) {
	ms.shutdown.Add(ctx, 1)
	
	log.Printf("[DEBUG] Shutdown recorded: reason=%s", reason)
}

// ReportMetrics contains metrics from platform.report event
type ReportMetrics struct {
	DurationMs        float64
	BilledDurationMs  int64
	MemorySizeMB      int64
	MaxMemoryUsedMB   int64
	InitDurationMs    float64
	RestoreDurationMs float64 // SnapStart restore duration
}

// shouldEmitSemconvMetrics checks if FaaS semantic convention metrics should be emitted
func shouldEmitSemconvMetrics() bool {
	val := os.Getenv("OTEL_LAMBDA_EMIT_SEMCONV")
	if val == "" {
		return false
	}
	
	if enabled, err := strconv.ParseBool(val); err == nil {
		return enabled
	}
	
	return false
}

