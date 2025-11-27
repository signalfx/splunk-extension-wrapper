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

package telemetry

import (
	"context"
	"log"
	"time"

	"github.com/splunk/lambda-extension/internal/otelmetrics"
)

// ExampleUsage demonstrates how to set up and use the telemetry subscriber
// with OpenTelemetry metrics.
//
// This is not a test, but rather documentation showing the intended usage pattern.
func ExampleUsage() {
	ctx := context.Background()

	// Step 1: Set up OpenTelemetry MeterProvider
	provider, err := otelmetrics.Setup(ctx)
	if err != nil {
		log.Fatalf("Failed to setup OTel provider: %v", err)
	}
	defer provider.Shutdown(ctx)

	// Step 2: Create metrics sink using the meter from the provider
	meter := provider.MeterProvider().Meter("github.com/splunk/lambda-extension")
	metricsSink, err := otelmetrics.NewMetricsSink(meter)
	if err != nil {
		log.Fatalf("Failed to create metrics sink: %v", err)
	}

	// Step 3: Create telemetry subscriber
	// Note: extensionID would come from the Lambda Extension API registration
	subscriber := NewTelemetrySubscriber(Config{
		ExtensionID: "your-extension-id-from-registration",
		MetricsSink: metricsSink,
	})

	// Step 4: Start the telemetry subscriber
	if err := subscriber.Start(ctx); err != nil {
		log.Fatalf("Failed to start telemetry subscriber: %v", err)
	}

	// Step 5: The subscriber now listens for telemetry events and
	// automatically records metrics via the MetricsSink

	// ... Your extension's main loop runs here ...

	// Step 6: Shutdown gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := subscriber.Shutdown(shutdownCtx); err != nil {
		log.Printf("Telemetry subscriber shutdown error: %v", err)
	}
}

// Example of how environment variables control behavior:
//
// Required for Lambda:
//   AWS_LAMBDA_RUNTIME_API - Lambda runtime API endpoint
//   AWS_LAMBDA_FUNCTION_NAME - Function name (for resource attributes)
//   AWS_LAMBDA_FUNCTION_VERSION - Function version
//   AWS_REGION - AWS region
//
// OpenTelemetry configuration (all optional):
//   OTEL_EXPORTER_OTLP_ENDPOINT - OTLP gRPC endpoint (default: 127.0.0.1:4317)
//   OTEL_EXPORTER_OTLP_INSECURE - Use insecure connection (default: false)
//   OTEL_LAMBDA_EMIT_SEMCONV - Emit FaaS semantic convention metrics (default: false)
//
// Telemetry API configuration (optional):
//   Default listener: 127.0.0.1:4243
//   Configure via Config struct when creating TelemetrySubscriber

