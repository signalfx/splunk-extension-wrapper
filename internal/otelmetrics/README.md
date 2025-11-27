# OpenTelemetry Metrics Package

This package provides first-class OpenTelemetry metrics support for the Lambda extension.

## Features

- **Automatic Setup**: Configures an OTel `MeterProvider` with OTLP gRPC exporter
- **Lambda-Specific Metrics**: Pre-configured instruments for Lambda lifecycle events
- **FaaS Semantic Conventions**: Optional support for standard FaaS metrics
- **Environment-Based Configuration**: Respects all standard `OTEL_*` environment variables

## Quick Start

```go
// 1. Setup OTel provider
provider, err := otelmetrics.Setup(ctx)
if err != nil {
    log.Fatal(err)
}
defer provider.Shutdown(ctx)

// 2. Create instruments
instruments, err := otelmetrics.NewInstruments(provider)
if err != nil {
    log.Fatal(err)
}

// 3. Use instruments to record metrics
instruments.Invocation().Add(ctx, 1)
instruments.InitializationLatency().Add(ctx, 150) // milliseconds
```

## Metrics

### Lambda-Specific Metrics (Always Available)

- `lambda.function.invocation` (Counter) - Number of invocations
- `lambda.function.initialization` (Counter) - Number of cold starts
- `lambda.function.initialization.latency` (UpDownCounter) - Cold start latency in ms
- `lambda.function.shutdown` (Counter) - Number of shutdowns
- `lambda.function.lifetime` (UpDownCounter) - Total environment lifetime in ms
- `lambda.function.cold_starts` (Counter) - On-demand cold starts
- `lambda.function.warm_starts` (Counter) - SnapStart warm starts
- `lambda.function.response_size` (Histogram) - Response payload size in bytes
- `lambda.function.snapstart.restore_duration` (Histogram) - SnapStart restore time in ms

### FaaS Semantic Convention Metrics (Optional)

Enable with `OTEL_LAMBDA_EMIT_SEMCONV=true`:

- `faas.invocations` (Counter) - Successful invocations
- `faas.errors` (Counter) - Failed invocations
- `faas.timeouts` (Counter) - Timed out invocations
- `faas.init_duration` (Histogram, seconds) - Init duration
- `faas.duration` (Histogram, seconds) - Invocation duration
- `faas.mem_usage` (Histogram, bytes) - Memory usage

## Configuration

Environment variables:

- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP endpoint (default: `127.0.0.1:4317`)
- `OTEL_EXPORTER_OTLP_INSECURE` - Use insecure connection (default: `false`)
- `OTEL_LAMBDA_EMIT_SEMCONV` - Enable FaaS semconv metrics (default: `false`)

## Integration

The `MetricsSink` interface allows integration with the Lambda Telemetry API.
See `internal/telemetry` package for automatic telemetry event processing.

