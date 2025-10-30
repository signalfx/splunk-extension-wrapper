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
)

func TestSetupAndShutdown(t *testing.T) {
	// Set up test environment
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test-function")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	defer func() {
		os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
		os.Unsetenv("AWS_LAMBDA_FUNCTION_VERSION")
		os.Unsetenv("AWS_REGION")
		os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This test will fail if there's no OTLP collector running,
	// but it should still initialize the provider without error
	provider, err := Setup(ctx)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	if provider.MeterProvider() == nil {
		t.Fatal("MeterProvider is nil")
	}

	// Create instruments
	instruments, err := NewInstruments(provider)
	if err != nil {
		t.Fatalf("NewInstruments failed: %v", err)
	}

	if instruments.Invocation() == nil {
		t.Error("Invocation instrument is nil")
	}

	if instruments.Initialization() == nil {
		t.Error("Initialization instrument is nil")
	}

	if instruments.InitializationLatency() == nil {
		t.Error("InitializationLatency instrument is nil")
	}

	if instruments.Shutdown() == nil {
		t.Error("Shutdown instrument is nil")
	}

	if instruments.Lifetime() == nil {
		t.Error("Lifetime instrument is nil")
	}

	// Test that semconv instruments are nil by default
	if instruments.EmitSemconv() {
		t.Error("EmitSemconv should be false by default")
	}

	if instruments.FaasInvocations() != nil {
		t.Error("FaasInvocations should be nil when semconv is disabled")
	}

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	if err := provider.Shutdown(shutdownCtx); err != nil {
		// Shutdown may fail if there's no OTLP collector running, which is expected in tests
		t.Logf("Shutdown returned error (expected if no collector running): %v", err)
	}
}

func TestSemconvEnabled(t *testing.T) {
	// Set up test environment with semconv enabled
	os.Setenv("OTEL_LAMBDA_EMIT_SEMCONV", "true")
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test-function")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	defer func() {
		os.Unsetenv("OTEL_LAMBDA_EMIT_SEMCONV")
		os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
		os.Unsetenv("AWS_LAMBDA_FUNCTION_VERSION")
		os.Unsetenv("AWS_REGION")
		os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	provider, err := Setup(ctx)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer provider.Shutdown(context.Background())

	instruments, err := NewInstruments(provider)
	if err != nil {
		t.Fatalf("NewInstruments failed: %v", err)
	}

	if !instruments.EmitSemconv() {
		t.Error("EmitSemconv should be true when OTEL_LAMBDA_EMIT_SEMCONV=true")
	}

	if instruments.FaasInvocations() == nil {
		t.Error("FaasInvocations should not be nil when semconv is enabled")
	}

	if instruments.FaasErrors() == nil {
		t.Error("FaasErrors should not be nil when semconv is enabled")
	}

	if instruments.FaasTimeouts() == nil {
		t.Error("FaasTimeouts should not be nil when semconv is enabled")
	}

	if instruments.FaasInitDuration() == nil {
		t.Error("FaasInitDuration should not be nil when semconv is enabled")
	}

	if instruments.FaasInvokeDuration() == nil {
		t.Error("FaasInvokeDuration should not be nil when semconv is enabled")
	}
}

func TestRecordMetrics(t *testing.T) {
	// Set up test environment
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test-function")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	defer func() {
		os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
		os.Unsetenv("AWS_LAMBDA_FUNCTION_VERSION")
		os.Unsetenv("AWS_REGION")
		os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")
	}()

	ctx := context.Background()

	provider, err := Setup(ctx)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer provider.Shutdown(ctx)

	instruments, err := NewInstruments(provider)
	if err != nil {
		t.Fatalf("NewInstruments failed: %v", err)
	}

	// Test recording metrics (this should not panic)
	instruments.Invocation().Add(ctx, 1)
	instruments.Initialization().Add(ctx, 1)
	instruments.InitializationLatency().Add(ctx, 150)
	instruments.Shutdown().Add(ctx, 1)
	instruments.Lifetime().Add(ctx, 5000)

	// No assertions here - we're just verifying that recording doesn't panic
	// The actual export would require a running OTLP collector
}

// TestResourceBuildingWithMissingEnvVars tests resource creation with missing env vars
func TestResourceBuildingWithMissingEnvVars(t *testing.T) {
	testCases := []struct {
		name        string
		setEnvVars  func()
		expectError bool
	}{
		{
			name: "all env vars present",
			setEnvVars: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test-function")
				os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
				os.Setenv("AWS_REGION", "us-east-1")
			},
			expectError: false,
		},
		{
			name: "missing function name",
			setEnvVars: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
				os.Setenv("AWS_REGION", "us-east-1")
			},
			expectError: false, // Should use defaults
		},
		{
			name: "missing function version",
			setEnvVars: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test-function")
				os.Unsetenv("AWS_LAMBDA_FUNCTION_VERSION")
				os.Setenv("AWS_REGION", "us-east-1")
			},
			expectError: false, // Should use defaults
		},
		{
			name: "missing region",
			setEnvVars: func() {
				os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test-function")
				os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
				os.Unsetenv("AWS_REGION")
			},
			expectError: false, // Should use defaults
		},
		{
			name: "all missing",
			setEnvVars: func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Unsetenv("AWS_LAMBDA_FUNCTION_VERSION")
				os.Unsetenv("AWS_REGION")
			},
			expectError: false, // Should use all defaults
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean slate
			os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
			os.Unsetenv("AWS_LAMBDA_FUNCTION_VERSION")
			os.Unsetenv("AWS_REGION")
			
			// Set up test environment
			tc.setEnvVars()
			os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
			
			defer func() {
				os.Unsetenv("AWS_LAMBDA_FUNCTION_NAME")
				os.Unsetenv("AWS_LAMBDA_FUNCTION_VERSION")
				os.Unsetenv("AWS_REGION")
				os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Setup should succeed even with missing env vars (uses defaults)
			provider, err := Setup(ctx)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if provider != nil {
				provider.Shutdown(ctx)
			}
		})
	}
}

