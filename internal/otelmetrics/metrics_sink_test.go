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
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestNewMetricsSink(t *testing.T) {
	// Create a test meter provider
	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")

	// Create metrics sink
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	if sink == nil {
		t.Fatal("Metrics sink is nil")
	}

	// Verify instruments are created
	if sink.invocation == nil {
		t.Error("invocation instrument is nil")
	}
	if sink.initialization == nil {
		t.Error("initialization instrument is nil")
	}
	if sink.initializationLatency == nil {
		t.Error("initializationLatency instrument is nil")
	}
	if sink.shutdown == nil {
		t.Error("shutdown instrument is nil")
	}
	if sink.lifetime == nil {
		t.Error("lifetime instrument is nil")
	}
}

func TestMetricsSinkWithSemconv(t *testing.T) {
	// Enable semconv
	os.Setenv("OTEL_LAMBDA_EMIT_SEMCONV", "true")
	defer os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")

	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")

	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	if !sink.emitSemconv {
		t.Error("emitSemconv should be true")
	}

	// Verify FaaS instruments are created
	if sink.faasInvocations == nil {
		t.Error("faasInvocations instrument is nil")
	}
	if sink.faasErrors == nil {
		t.Error("faasErrors instrument is nil")
	}
	if sink.faasTimeouts == nil {
		t.Error("faasTimeouts instrument is nil")
	}
	if sink.faasInitDuration == nil {
		t.Error("faasInitDuration instrument is nil")
	}
	if sink.faasInvokeDuration == nil {
		t.Error("faasInvokeDuration instrument is nil")
	}
	if sink.faasMemUsage == nil {
		t.Error("faasMemUsage instrument is nil")
	}
}

func TestMetricsSinkRecordLifecycle(t *testing.T) {
	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Test initialization lifecycle
	sink.RecordInitStart(ctx, now, "on-demand")
	if sink.initStartTime.IsZero() {
		t.Error("initStartTime should be set after RecordInitStart")
	}
	if !sink.environmentStarted {
		t.Error("environmentStarted should be true after RecordInitStart")
	}

	time.Sleep(10 * time.Millisecond)
	sink.RecordInitEnd(ctx, now.Add(100*time.Millisecond), "on-demand")

	// Test invocation lifecycle
	sink.RecordStart(ctx, now.Add(200*time.Millisecond), "request-1")
	sink.RecordRuntimeDone(ctx, now.Add(300*time.Millisecond), "request-1", "success", 100, 0)

	// Test report with metrics
	reportMetrics := ReportMetrics{
		DurationMs:       100.5,
		BilledDurationMs: 101,
		MemorySizeMB:     512,
		MaxMemoryUsedMB:  256,
		InitDurationMs:   100.0,
	}
	sink.RecordReport(ctx, now.Add(350*time.Millisecond), "request-1", reportMetrics)

	if sink.lastLifetimeMs == 0 {
		t.Error("lastLifetimeMs should be non-zero after RecordReport")
	}

	// Test shutdown
	sink.RecordShutdown(ctx, now.Add(1*time.Second), "spindown")
}

func TestMetricsSinkStatusHandling(t *testing.T) {
	os.Setenv("OTEL_LAMBDA_EMIT_SEMCONV", "true")
	defer os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")

	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Test different statuses
	testCases := []struct {
		name   string
		status string
	}{
		{"success", "success"},
		{"error", "error"},
		{"failure", "failure"},
		{"timeout", "timeout"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			sink.RecordRuntimeDone(ctx, now, "request-"+tc.name, tc.status, 100, 0)
		})
	}
}

func TestMetricsSinkLifetimeDelta(t *testing.T) {
	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Initialize
	sink.RecordInitStart(ctx, now, "on-demand")

	// First report at 100ms
	reportMetrics1 := ReportMetrics{
		DurationMs:      100.0,
		MaxMemoryUsedMB: 128,
	}
	sink.RecordReport(ctx, now.Add(100*time.Millisecond), "request-1", reportMetrics1)

	firstLifetime := sink.lastLifetimeMs
	if firstLifetime == 0 {
		t.Error("First lifetime should be non-zero")
	}

	// Second report at 200ms - should add delta
	reportMetrics2 := ReportMetrics{
		DurationMs:      50.0,
		MaxMemoryUsedMB: 150,
	}
	sink.RecordReport(ctx, now.Add(200*time.Millisecond), "request-2", reportMetrics2)

	secondLifetime := sink.lastLifetimeMs
	if secondLifetime <= firstLifetime {
		t.Errorf("Second lifetime (%d) should be greater than first (%d)", secondLifetime, firstLifetime)
	}

	expectedDelta := int64(100) // 200ms - 100ms
	actualDelta := secondLifetime - firstLifetime
	if actualDelta != expectedDelta {
		t.Errorf("Expected delta of %dms, got %dms", expectedDelta, actualDelta)
	}
}

// TestMetricsSinkConcurrentAccess tests race conditions in MetricsSink
func TestMetricsSinkConcurrentAccess(t *testing.T) {
	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Initialize first
	sink.RecordInitStart(ctx, now, "on-demand")
	sink.RecordInitEnd(ctx, now.Add(100*time.Millisecond), "on-demand")

	// Run concurrent operations
	const goroutines = 10
	const operationsPerGoroutine = 100

	done := make(chan bool, goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			for i := 0; i < operationsPerGoroutine; i++ {
				requestID := time.Now().Format("req-" + string(rune(id)) + "-" + string(rune(i)))
				
				sink.RecordStart(ctx, time.Now(), requestID)
				sink.RecordRuntimeDone(ctx, time.Now(), requestID, "success", 100, 0)
				
				report := ReportMetrics{
					DurationMs:      float64(100 + i),
					MaxMemoryUsedMB: int64(128 + i),
				}
				sink.RecordReport(ctx, time.Now(), requestID, report)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// If we get here without panic, test passes
	t.Log("Concurrent access completed without race conditions")
}

// TestMetricsSinkMemoryLeak tests handling of many sequential invocations
// Note: Current implementation uses single invokeStartTime (no per-request map),
// so there's no memory leak from request tracking. This test verifies that
// sequential invocations are handled correctly.
func TestMetricsSinkMemoryLeak(t *testing.T) {
	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Initialize
	sink.RecordInitStart(ctx, now, "on-demand")
	sink.RecordInitEnd(ctx, now.Add(100*time.Millisecond), "on-demand")

	// Record 1000 sequential invocations (simulating high throughput)
	const numRequests = 1000
	for i := 0; i < numRequests; i++ {
		requestID := fmt.Sprintf("req-%d", i)
		startTime := now.Add(time.Duration(i*100) * time.Millisecond)
		endTime := startTime.Add(50 * time.Millisecond)
		
		sink.RecordStart(ctx, startTime, requestID)
		sink.RecordRuntimeDone(ctx, endTime, requestID, "success", 50, 0)
		
		report := ReportMetrics{
			DurationMs:      50.0,
			MaxMemoryUsedMB: int64(128 + i%100),
		}
		sink.RecordReport(ctx, endTime, requestID, report)
	}

	// Verify last invoke start time was updated
	sink.mu.RLock()
	lastInvokeStartIsSet := !sink.invokeStartTime.IsZero()
	sink.mu.RUnlock()

	if !lastInvokeStartIsSet {
		t.Error("invokeStartTime should be set after processing invocations")
	}

	// Note: Current design uses single invokeStartTime field (overwrites on each invocation).
	// This is fine for Lambda's sequential invocation model within a single execution environment.
	t.Logf("Successfully processed %d sequential invocations without memory leak", numRequests)
}

// TestRecordInitEndWithoutInitStart tests error path when init state is missing
func TestRecordInitEndWithoutInitStart(t *testing.T) {
	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Call RecordInitEnd WITHOUT calling RecordInitStart first
	// This should log a warning and return early
	sink.RecordInitEnd(ctx, now, "on-demand")

	// Verify initStartTime is still zero
	sink.mu.Lock()
	isZero := sink.initStartTime.IsZero()
	sink.mu.Unlock()

	if !isZero {
		t.Error("initStartTime should still be zero after RecordInitEnd without RecordInitStart")
	}

	// Verify initialization counter was NOT incremented
	// (We can't easily assert this without metric inspection, but the function should return early)
	t.Log("RecordInitEnd without RecordInitStart handled gracefully")
}

// TestMetricsSinkWithNilMeter tests nil checks
func TestMetricsSinkWithNilMeter(t *testing.T) {
	// Test with nil meter should return error
	_, err := NewMetricsSink(nil)
	if err == nil {
		t.Error("Expected error when creating MetricsSink with nil meter")
	}
}

// TestEventsOutOfOrder tests handling of events in unexpected order
func TestEventsOutOfOrder(t *testing.T) {
	res := resource.Default()
	provider := metric.NewMeterProvider(metric.WithResource(res))
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Scenario 1: Start before Init
	sink.RecordStart(ctx, now, "req-1")
	sink.RecordInitStart(ctx, now.Add(100*time.Millisecond), "on-demand")
	
	// Should not panic - verify environment started flag
	sink.mu.Lock()
	envStarted := sink.environmentStarted
	sink.mu.Unlock()
	
	if !envStarted {
		t.Error("Environment should be marked as started after RecordInitStart")
	}

	// Scenario 2: RuntimeDone before Start
	sink.RecordRuntimeDone(ctx, now.Add(200*time.Millisecond), "req-unknown", "success", 100, 0)
	// Should handle gracefully (no panic)

	// Scenario 3: Report before Start
	report := ReportMetrics{
		DurationMs:      100.0,
		MaxMemoryUsedMB: 128,
	}
	sink.RecordReport(ctx, now.Add(300*time.Millisecond), "req-unknown", report)
	// Should handle gracefully

	// Scenario 4: Multiple InitStart calls
	sink.RecordInitStart(ctx, now.Add(400*time.Millisecond), "on-demand")
	sink.RecordInitStart(ctx, now.Add(500*time.Millisecond), "on-demand")
	// Should handle gracefully

	t.Log("Out-of-order events handled without panic")
}

