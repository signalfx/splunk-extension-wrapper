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
	"os"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

// TestSemconvDisabledByDefault ensures FaaS semconv metrics are not created by default
func TestSemconvDisabledByDefault(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")

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

	if sink.emitSemconv {
		t.Error("emitSemconv should be false by default")
	}

	// Verify FaaS instruments are nil
	if sink.faasInvocations != nil {
		t.Error("faasInvocations should be nil when semconv disabled")
	}
	if sink.faasErrors != nil {
		t.Error("faasErrors should be nil when semconv disabled")
	}
	if sink.faasTimeouts != nil {
		t.Error("faasTimeouts should be nil when semconv disabled")
	}
	if sink.faasInitDuration != nil {
		t.Error("faasInitDuration should be nil when semconv disabled")
	}
	if sink.faasInvokeDuration != nil {
		t.Error("faasInvokeDuration should be nil when semconv disabled")
	}
	if sink.faasMemUsage != nil {
		t.Error("faasMemUsage should be nil when semconv disabled")
	}
}

// TestSemconvEnabledWithEnvVar tests that setting OTEL_LAMBDA_EMIT_SEMCONV=true enables semconv
func TestSemconvEnabledWithEnvVar(t *testing.T) {
	os.Setenv("OTEL_LAMBDA_EMIT_SEMCONV", "true")
	defer os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")

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

	if !sink.emitSemconv {
		t.Error("emitSemconv should be true when OTEL_LAMBDA_EMIT_SEMCONV=true")
	}

	// Verify FaaS instruments are created
	if sink.faasInvocations == nil {
		t.Error("faasInvocations should not be nil when semconv enabled")
	}
	if sink.faasErrors == nil {
		t.Error("faasErrors should not be nil when semconv enabled")
	}
	if sink.faasTimeouts == nil {
		t.Error("faasTimeouts should not be nil when semconv enabled")
	}
	if sink.faasInitDuration == nil {
		t.Error("faasInitDuration should not be nil when semconv enabled")
	}
	if sink.faasInvokeDuration == nil {
		t.Error("faasInvokeDuration should not be nil when semconv enabled")
	}
	if sink.faasMemUsage == nil {
		t.Error("faasMemUsage should not be nil when semconv enabled")
	}
}

// TestSemconvVariousEnvValues tests different environment variable values
func TestSemconvVariousEnvValues(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true", "true", true},
		{"1", "1", true},
		{"false", "false", false},
		{"0", "0", false},
		{"empty", "", false},
		{"invalid", "maybe", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value == "" {
				os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")
			} else {
				os.Setenv("OTEL_LAMBDA_EMIT_SEMCONV", tc.value)
			}
			defer os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")

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

			if sink.emitSemconv != tc.expected {
				t.Errorf("Expected emitSemconv=%v for value '%s', got %v", tc.expected, tc.value, sink.emitSemconv)
			}
		})
	}
}

// TestSemconvMetricsRecorded verifies that FaaS semconv metrics are actually recorded
func TestSemconvMetricsRecorded(t *testing.T) {
	os.Setenv("OTEL_LAMBDA_EMIT_SEMCONV", "true")
	defer os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")

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
	sink.RecordInitEnd(ctx, now.Add(100*time.Millisecond), "on-demand")

	// Record successful invocation
	sink.RecordStart(ctx, now.Add(200*time.Millisecond), "req-success")
	sink.RecordRuntimeDone(ctx, now.Add(450*time.Millisecond), "req-success", "success", 250, 0)

	// Record error invocation
	sink.RecordStart(ctx, now.Add(500*time.Millisecond), "req-error")
	sink.RecordRuntimeDone(ctx, now.Add(650*time.Millisecond), "req-error", "error", 150, 0)

	// Record timeout invocation
	sink.RecordStart(ctx, now.Add(700*time.Millisecond), "req-timeout")
	sink.RecordRuntimeDone(ctx, now.Add(900*time.Millisecond), "req-timeout", "timeout", 200, 0)

	// Record report with memory
	report := ReportMetrics{
		DurationMs:      250.0,
		MaxMemoryUsedMB: 128,
	}
	sink.RecordReport(ctx, now.Add(1*time.Second), "req-success", report)

	// Collect metrics
	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Verify FaaS metrics exist
	metrics := collectMetricNames(data)

	expectedMetrics := []string{
		"faas.invocations",
		"faas.errors",
		"faas.timeouts",
		"faas.init_duration",
		"faas.duration",
		"faas.mem_usage",
	}

	for _, expected := range expectedMetrics {
		if !contains(metrics, expected) {
			t.Errorf("Expected metric %s not found. Available metrics: %v", expected, metrics)
		}
	}

	// Verify counts
	invocations := findCounterValue(t, data, "faas.invocations")
	if invocations != 3 {
		t.Errorf("Expected 3 faas.invocations (success+error+timeout), got %d", invocations)
	}

	errors := findCounterValue(t, data, "faas.errors")
	if errors != 1 {
		t.Errorf("Expected 1 faas.error, got %d", errors)
	}

	timeouts := findCounterValue(t, data, "faas.timeouts")
	if timeouts != 1 {
		t.Errorf("Expected 1 faas.timeout, got %d", timeouts)
	}
}

// TestSemconvMetricsNotRecordedWhenDisabled ensures FaaS metrics are not recorded when disabled
func TestSemconvMetricsNotRecordedWhenDisabled(t *testing.T) {
	os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")

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

	// Record full lifecycle
	sink.RecordInitStart(ctx, now, "on-demand")
	sink.RecordInitEnd(ctx, now.Add(100*time.Millisecond), "on-demand")
	sink.RecordStart(ctx, now.Add(200*time.Millisecond), "req-1")
	sink.RecordRuntimeDone(ctx, now.Add(450*time.Millisecond), "req-1", "success", 250, 0)
	report := ReportMetrics{
		DurationMs:      250.0,
		MaxMemoryUsedMB: 128,
	}
	sink.RecordReport(ctx, now.Add(500*time.Millisecond), "req-1", report)

	// Collect metrics
	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Verify FaaS metrics do NOT exist
	metrics := collectMetricNames(data)

	forbiddenMetrics := []string{
		"faas.invocations",
		"faas.errors",
		"faas.timeouts",
		"faas.init_duration",
		"faas.duration",
		"faas.mem_usage",
	}

	for _, forbidden := range forbiddenMetrics {
		if contains(metrics, forbidden) {
			t.Errorf("Metric %s should not exist when semconv disabled. Available metrics: %v", forbidden, metrics)
		}
	}

	// Verify Lambda-specific metrics still exist
	expectedMetrics := []string{
		"lambda.function.invocation",
		"lambda.function.initialization",
		"lambda.function.initialization.latency",
		"lambda.function.lifetime",
	}

	for _, expected := range expectedMetrics {
		if !contains(metrics, expected) {
			t.Errorf("Expected Lambda metric %s not found. Available metrics: %v", expected, metrics)
		}
	}
}

// TestSemconvOptionOverridesEnv tests that WithSemconv option overrides environment
func TestSemconvOptionOverridesEnv(t *testing.T) {
	// Set env to false
	os.Setenv("OTEL_LAMBDA_EMIT_SEMCONV", "false")
	defer os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")

	reader := metric.NewManualReader()
	res := resource.Default()
	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("test")
	
	// Override with option to enable
	sink, err := NewMetricsSink(meter, WithSemconv(true))
	if err != nil {
		t.Fatalf("Failed to create metrics sink: %v", err)
	}

	if !sink.emitSemconv {
		t.Error("Expected emitSemconv=true when explicitly set via option")
	}

	// Verify instruments are created
	if sink.faasInvocations == nil {
		t.Error("faasInvocations should be created when enabled via option")
	}
}

// Helper functions

func collectMetricNames(data metricdata.ResourceMetrics) []string {
	names := make([]string, 0)
	for _, sm := range data.ScopeMetrics {
		for _, m := range sm.Metrics {
			names = append(names, m.Name)
		}
	}
	return names
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

