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
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.28.0"
)

const (
	// Default OTLP endpoint for local collector
	defaultOTLPEndpoint = "127.0.0.1:4317"
	
	// Export interval for metrics
	exportInterval = 5 * time.Second
)

// Provider wraps an OpenTelemetry MeterProvider with lifecycle management.
type Provider struct {
	meterProvider *metric.MeterProvider
}

// Setup initializes and configures an OpenTelemetry MeterProvider.
// It reads configuration from standard OTEL_* environment variables:
//   - OTEL_EXPORTER_OTLP_ENDPOINT: OTLP gRPC endpoint (default: 127.0.0.1:4317)
//   - OTEL_EXPORTER_OTLP_HEADERS: Custom headers for OTLP requests
//   - OTEL_EXPORTER_OTLP_TIMEOUT: Export timeout
//   - Plus all other standard OpenTelemetry SDK environment variables
//
// Resource attributes are automatically derived from AWS Lambda environment:
//   - service.name: AWS Lambda function name
//   - service.version: AWS Lambda function version
//   - cloud.provider: aws
//   - cloud.region: AWS region
//   - faas.name: Lambda function name
//   - faas.version: Lambda function version
//
// The provider uses a PeriodicReader with 5s export interval.
func Setup(ctx context.Context) (*Provider, error) {
	// Build resource with Lambda-specific attributes
	res, err := buildResource(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to build resource: %v", err)
		// Continue with default resource
		res = resource.Default()
	}

	// Configure OTLP gRPC exporter
	// Endpoint defaults to 127.0.0.1:4317 but respects OTEL_EXPORTER_OTLP_ENDPOINT
	endpoint := getOTLPEndpoint()
	
	exporterOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
	}

	// Respect OTEL_EXPORTER_OTLP_INSECURE env var
	if isInsecure() {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, err
	}

	// Create periodic reader with 5s interval
	reader := metric.NewPeriodicReader(
		exporter,
		metric.WithInterval(exportInterval),
	)

	// Create MeterProvider
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)

	// Log to stderr so it always appears in CloudWatch logs
	fmt.Fprintf(os.Stderr, "[splunk-extension-wrapper] [INFO] OpenTelemetry MeterProvider initialized (endpoint: %s, interval: %s)\n", endpoint, exportInterval)

	return &Provider{
		meterProvider: meterProvider,
	}, nil
}

// MeterProvider returns the underlying OpenTelemetry MeterProvider.
func (p *Provider) MeterProvider() *metric.MeterProvider {
	return p.meterProvider
}

// Shutdown cleanly shuts down the MeterProvider, flushing any pending metrics.
// It should be called before application exit to ensure all metrics are exported.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.meterProvider != nil {
		return p.meterProvider.Shutdown(ctx)
	}
	return nil
}

// buildResource creates a Resource with AWS Lambda-specific attributes
func buildResource(ctx context.Context) (*resource.Resource, error) {
	// Get Lambda environment variables
	functionName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	functionVersion := os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")
	region := os.Getenv("AWS_REGION")

	attrs := []resource.Option{
		resource.WithAttributes(
			semconv.ServiceNameKey.String(functionName),
			semconv.ServiceVersionKey.String(functionVersion),
			semconv.CloudProviderAWS,
			semconv.CloudRegionKey.String(region),
			semconv.FaaSNameKey.String(functionName),
			semconv.FaaSVersionKey.String(functionVersion),
		),
	}

	// Merge with default resource (includes telemetry.sdk.* attributes)
	return resource.New(ctx, attrs...)
}

// getOTLPEndpoint returns the OTLP endpoint from environment or default
func getOTLPEndpoint() string {
	// Check OTEL_EXPORTER_OTLP_ENDPOINT first
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	
	// Check metrics-specific endpoint
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	
	return defaultOTLPEndpoint
}

// isInsecure checks if the connection should be insecure (no TLS)
func isInsecure() bool {
	insecure := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE")
	return insecure == "true" || insecure == "1"
}

