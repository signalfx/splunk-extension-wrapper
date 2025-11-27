# Telemetry Package

This package provides integration with the AWS Lambda Telemetry API to collect detailed runtime metrics.

## Features

- **HTTP Listener**: Receives telemetry events on `127.0.0.1:4243`
- **Automatic Subscription**: Subscribes to platform, function, and extension event streams
- **Event Processing**: Handles all major Lambda lifecycle events
- **State Tracking**: Maintains execution environment state
- **Metrics Integration**: Pluggable `MetricsSink` for recording metrics

## Quick Start

```go
// 1. Create a metrics sink (e.g., from otelmetrics package)
metricsSink := otelmetrics.NewDefaultMetricsSink(instruments)

// 2. Create telemetry subscriber
subscriber := telemetry.NewTelemetrySubscriber(telemetry.Config{
    ExtensionID: extensionID, // from Lambda Extension API registration
    MetricsSink: metricsSink,
})

// 3. Start listening and subscribing
if err := subscriber.Start(ctx); err != nil {
    log.Fatal(err)
}

// 4. Shutdown gracefully when done
defer subscriber.Shutdown(ctx)
```

## Supported Events

### Platform Events

- `platform.initStart` - Environment initialization started
- `platform.initEnd` - Environment initialization completed
- `platform.start` - Function invocation started
- `platform.runtimeDone` - Function invocation completed
- `platform.report` - Detailed metrics report
- `platform.shutdown` - Environment shutdown

### Configuration

- Listener: `127.0.0.1:4243` (default)
- Buffer: 25 events, 256KB max, 500ms timeout
- Streams: platform, function, extension

## Architecture

```
Lambda Runtime
      │
      ├─> Telemetry API
      │         │
      │         v
      │   HTTP Listener (4243)
      │         │
      │         v
      │   Event Handlers
      │         │
      │         v
      │   MetricsSink
      │         │
      │         v
      │   OTel Metrics
      │         │
      │         v
      │   OTLP Exporter
```

## State Tracking

The subscriber maintains state for:
- Init start/end times
- First invocation time
- Last request ID
- Last report metrics (memory, duration, etc.)

Access via `subscriber.GetState()` for debugging.

