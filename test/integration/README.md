# Integration Tests

This directory contains integration tests that verify the complete OTel metrics pipeline with a real OpenTelemetry Collector.

## Prerequisites

- Docker and Docker Compose installed
- Go 1.23+ installed

## Running Integration Tests

### 1. Start the OTel Collector

```bash
cd test/integration
docker-compose up -d
```

This starts a local OTel Collector that:
- Listens on `localhost:4317` for OTLP gRPC
- Exports metrics to `/tmp/otel-metrics.json`
- Logs metrics to stdout

### 2. Run the Integration Tests

```bash
# From the project root
go test -tags=integration ./test/integration/... -v
```

Or use the provided script:

```bash
./test/integration/run-integration-tests.sh
```

### 3. Stop the Collector

```bash
cd test/integration
docker-compose down
```

## What the Tests Verify

### TestIntegrationFullLifecycle
- Sets up OTel MeterProvider with OTLP exporter
- Creates MetricsSink
- Simulates complete Lambda lifecycle:
  - Initialization (cold start)
  - Multiple invocations (success, error, timeout)
  - Shutdown
- Verifies metrics are exported to collector
- Checks for all expected metrics:
  - `lambda.function.invocation`
  - `lambda.function.initialization`
  - `lambda.function.initialization.latency`
  - `lambda.function.shutdown`
  - `lambda.function.lifetime`
  - `faas.invocations`
  - `faas.errors`
  - `faas.timeouts`

### TestIntegrationWithTelemetrySubscriber
- Tests the full integration pattern
- Simulates telemetry events through MetricsSink
- Verifies end-to-end metric export

## Viewing Exported Metrics

The collector writes metrics to `/tmp/otel-metrics.json` in NDJSON format:

```bash
# View metrics in real-time
tail -f /tmp/otel-metrics.json | jq .

# Pretty-print all metrics
cat /tmp/otel-metrics.json | jq .
```

## Collector Configuration

The collector is configured in `otel-collector-config.yaml`:
- **Receivers**: OTLP gRPC (4317) and HTTP (4318)
- **Processors**: Batch processor (1s timeout)
- **Exporters**: File exporter and logging exporter

## Troubleshooting

### Collector not starting
```bash
# Check logs
docker-compose logs otel-collector

# Check if ports are available
lsof -i :4317
```

### Tests timing out
- Increase shutdown timeout in tests
- Check if collector is reachable: `curl localhost:4317`
- Verify Docker is running

### No metrics in output file
- Check collector logs for errors
- Verify `/tmp` is writable
- Ensure metrics are being sent (check test logs)

## Clean Up

```bash
# Stop collector
docker-compose down

# Remove metrics file
rm /tmp/otel-metrics.json

# Remove Docker volumes
docker-compose down -v
```

