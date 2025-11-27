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
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

// TestGaugeEmulationWithDeltas tests that UpDownCounter properly emulates gauge behavior
// by adding deltas instead of absolute values
func TestGaugeEmulationWithDeltas(t *testing.T) {
	// Create a manual reader to inspect metrics
	reader := metric.NewManualReader()
	res := resource.Default()
	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Initialize environment
	sink.RecordInitStart(ctx, now, "on-demand")

	// First report at 100ms - should add 100ms to lifetime
	report1 := ReportMetrics{
		DurationMs:      50.0,
		MaxMemoryUsedMB: 128,
	}
	sink.RecordReport(ctx, now.Add(100*time.Millisecond), "req-1", report1)

	// Force a metric collection
	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Find lifetime metric
	lifetimeValue := findUpDownCounterValue(t, data, "lambda.function.lifetime")
	if lifetimeValue != 100 {
		t.Errorf("Expected lifetime value of 100ms after first report, got %d", lifetimeValue)
	}

	// Second report at 250ms - should add delta of 150ms (250-100)
	report2 := ReportMetrics{
		DurationMs:      75.0,
		MaxMemoryUsedMB: 150,
	}
	sink.RecordReport(ctx, now.Add(250*time.Millisecond), "req-2", report2)

	// Collect again
	err = reader.Collect(ctx, &data)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Lifetime should now be 100 + 150 = 250ms
	lifetimeValue = findUpDownCounterValue(t, data, "lambda.function.lifetime")
	if lifetimeValue != 250 {
		t.Errorf("Expected lifetime value of 250ms after second report, got %d", lifetimeValue)
	}

	// Third report at 500ms - should add delta of 250ms (500-250)
	report3 := ReportMetrics{
		DurationMs:      100.0,
		MaxMemoryUsedMB: 175,
	}
	sink.RecordReport(ctx, now.Add(500*time.Millisecond), "req-3", report3)

	// Collect again
	err = reader.Collect(ctx, &data)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Lifetime should now be 250 + 250 = 500ms
	lifetimeValue = findUpDownCounterValue(t, data, "lambda.function.lifetime")
	if lifetimeValue != 500 {
		t.Errorf("Expected lifetime value of 500ms after third report, got %d", lifetimeValue)
	}
}

// TestGaugeEmulationMultipleReports verifies consistent delta calculation across many reports
func TestGaugeEmulationMultipleReports(t *testing.T) {
	reader := metric.NewManualReader()
	res := resource.Default()
	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)
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

	// Simulate 10 reports at 100ms intervals
	for i := 1; i <= 10; i++ {
		report := ReportMetrics{
			DurationMs:      float64(i * 10),
			MaxMemoryUsedMB: int64(100 + i*10),
		}
		sink.RecordReport(ctx, now.Add(time.Duration(i*100)*time.Millisecond), "req-"+string(rune(i)), report)
	}

	// Collect final metrics
	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// After 10 reports at 100ms intervals, lifetime should be 1000ms
	lifetimeValue := findUpDownCounterValue(t, data, "lambda.function.lifetime")
	expectedLifetime := int64(1000)
	if lifetimeValue != expectedLifetime {
		t.Errorf("Expected lifetime value of %dms after 10 reports, got %d", expectedLifetime, lifetimeValue)
	}
}

// TestInitializationLatencyDelta tests that initialization latency is recorded as delta
func TestInitializationLatencyDelta(t *testing.T) {
	reader := metric.NewManualReader()
	res := resource.Default()
	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	sink, err := NewMetricsSink(meter)
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Record init cycle
	sink.RecordInitStart(ctx, now, "on-demand")
	sink.RecordInitEnd(ctx, now.Add(150*time.Millisecond), "on-demand")

	// Collect metrics
	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Initialization latency should be 150ms
	latencyValue := findUpDownCounterValue(t, data, "lambda.function.initialization.latency")
	if latencyValue != 150 {
		t.Errorf("Expected initialization latency of 150ms, got %d", latencyValue)
	}

	// Initialization counter should be 1
	initCount := findCounterValue(t, data, "lambda.function.initialization")
	if initCount != 1 {
		t.Errorf("Expected initialization count of 1, got %d", initCount)
	}
}

// TestZeroDeltaDoesNotUpdate ensures zero deltas don't add to lifetime
func TestZeroDeltaDoesNotUpdate(t *testing.T) {
	reader := metric.NewManualReader()
	res := resource.Default()
	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)
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
	report1 := ReportMetrics{MaxMemoryUsedMB: 128}
	sink.RecordReport(ctx, now.Add(100*time.Millisecond), "req-1", report1)

	// Report at same timestamp (zero delta) - should not update
	sink.RecordReport(ctx, now.Add(100*time.Millisecond), "req-2", report1)

	// Collect metrics
	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Should still be 100ms (no delta added)
	lifetimeValue := findUpDownCounterValue(t, data, "lambda.function.lifetime")
	if lifetimeValue != 100 {
		t.Errorf("Expected lifetime of 100ms with zero delta, got %d", lifetimeValue)
	}
}

// Helper functions to find metric values

func findUpDownCounterValue(t *testing.T, data metricdata.ResourceMetrics, name string) int64 {
	for _, sm := range data.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					if len(sum.DataPoints) > 0 {
						return sum.DataPoints[0].Value
					}
				}
			}
		}
	}
	t.Logf("Warning: metric %s not found", name)
	return 0
}

func findCounterValue(t *testing.T, data metricdata.ResourceMetrics, name string) int64 {
	for _, sm := range data.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					if len(sum.DataPoints) > 0 {
						return sum.DataPoints[0].Value
					}
				}
			}
		}
	}
	t.Logf("Warning: metric %s not found", name)
	return 0
}

