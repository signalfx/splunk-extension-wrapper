# OpenTelemetry Metrics Quick Start

## 🚀 Enable OpenTelemetry in 3 Steps

### 1. Build & Deploy Extension Layer

```bash
cd splunk-extension-wrapper
./build-layer.sh

# Upload to AWS Lambda
aws lambda publish-layer-version \
  --layer-name splunk-otel-extension \
  --zip-file fileb://bin/extension.zip \
  --compatible-runtimes nodejs18.x nodejs20.x python3.11 python3.12
```

### 2. Configure Lambda Function

```bash
# Enable OpenTelemetry
aws lambda update-function-configuration \
  --function-name YOUR_FUNCTION \
  --environment Variables="{
    USE_OTEL_METRICS=true,
    OTEL_EXPORTER_OTLP_ENDPOINT=your-collector:4317,
    OTEL_EXPORTER_OTLP_INSECURE=true,
    OTEL_LAMBDA_EMIT_SEMCONV=true,
    SPLUNK_EXTENSION_WRAPPER_ENABLED=true
  }"
```

### 3. Verify

```bash
# Invoke function
aws lambda invoke --function-name YOUR_FUNCTION response.json

# Check logs
aws logs tail /aws/lambda/YOUR_FUNCTION --follow | grep OpenTelemetry
```

**Expected output:**
```
[splunk-extension-wrapper] OpenTelemetry metrics enabled
[splunk-extension-wrapper] [INFO] OpenTelemetry MeterProvider initialized
[splunk-extension-wrapper] Telemetry API subscriber started successfully
```

---

## 📊 Environment Variables

### Required

| Variable | Example | Description |
|----------|---------|-------------|
| `USE_OTEL_METRICS` | `true` | **Enable OpenTelemetry** |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `collector:4317` | Collector endpoint |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_INSECURE` | `false` | Use insecure gRPC |
| `OTEL_LAMBDA_EMIT_SEMCONV` | `false` | Enable 6 additional FaaS metrics |

---

## 📈 Metrics You'll Get

### Core Metrics (Always)
- `lambda.function.invocation` - Invocation count
- `lambda.function.initialization` - Total initializations
- `lambda.function.initialization.latency` - Init duration
- `lambda.function.cold_starts` - On-demand cold starts
- `lambda.function.warm_starts` - SnapStart warm starts
- `lambda.function.response_size` - Response payload size
- `lambda.function.snapstart.restore_duration` - SnapStart restore time
- `lambda.function.shutdown` - Shutdowns
- `lambda.function.lifetime` - Environment lifetime

### FaaS Metrics (If `OTEL_LAMBDA_EMIT_SEMCONV=true`)
- `faas.invocations` - Successful invocations
- `faas.errors` - Failed invocations
- `faas.timeouts` - Timeouts
- `faas.init_duration` - Init duration (seconds)
- `faas.duration` - Invocation duration (seconds)
- `faas.mem_usage` - Memory usage (bytes)

---

## 🎯 Common Setups

### Local Testing
```bash
# Start collector
cd test/integration
docker-compose up -d

# Set env vars
USE_OTEL_METRICS=true
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_EXPORTER_OTLP_INSECURE=true
```

### AWS with EC2 Collector
```bash
USE_OTEL_METRICS=true
OTEL_EXPORTER_OTLP_ENDPOINT=10.0.1.100:4317  # EC2 private IP
OTEL_EXPORTER_OTLP_INSECURE=true
```

### Splunk Observability Cloud
```bash
USE_OTEL_METRICS=true
OTEL_EXPORTER_OTLP_ENDPOINT=https://ingest.us1.signalfx.com:4317
OTEL_EXPORTER_OTLP_HEADERS=X-SF-Token=YOUR_TOKEN
OTEL_LAMBDA_EMIT_SEMCONV=true
```

---

## ❌ Disable OpenTelemetry (Rollback to SignalFx)

```bash
# Remove or set to false
aws lambda update-function-configuration \
  --function-name YOUR_FUNCTION \
  --environment Variables="{
    USE_OTEL_METRICS=false,
    SPLUNK_REALM=us1,
    SPLUNK_ACCESS_TOKEN=xxx
  }"
```

---

## 🔍 How to Know Which Metrics You're Getting

### SignalFx Metrics
**Logs show:**
```
SignalFx metrics enabled
SPLUNK_REALM: us1
```

**Env vars:**
```bash
SPLUNK_REALM=us1
SPLUNK_ACCESS_TOKEN=xxx
```

**No `USE_OTEL_METRICS` or set to `false`**

### OpenTelemetry Metrics
**Logs show:**
```
OpenTelemetry metrics enabled
OpenTelemetry MeterProvider initialized
Telemetry API subscriber started
```

**Env vars:**
```bash
USE_OTEL_METRICS=true
OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317
```

**Metrics include:**
- `faas.invocations` ← Only in OTel
- `faas.errors` ← Only in OTel
- `faas.timeouts` ← Only in OTel

---

## 🛠️ Troubleshooting

### Metrics not appearing?
```bash
# Check extension is using OTel
aws logs tail /aws/lambda/YOUR_FUNCTION | grep "OpenTelemetry metrics enabled"

# Check collector connectivity
aws logs tail /aws/lambda/YOUR_FUNCTION | grep "Failed to"
```

### Extension falls back to SignalFx?
```
Failed to initialize OpenTelemetry: connection refused
Falling back to SignalFx metrics
```
**Fix:** Check collector endpoint and security groups

### Telemetry API failed?
```
Failed to start telemetry subscriber
```
**Fix:** Check Lambda runtime supports Telemetry API (most do)

---

## ⚠️ Important Notes

- **Telemetry API Buffer:** AWS requires `maxItems` between 1000-10000 (we use 1000)
- **CloudWatch Logs:** Set `VERBOSE=true` to see all extension logs, or important messages go to stderr automatically

## 📚 Full Documentation

- [Package Documentation](../internal/otelmetrics/README.md)
- [Telemetry API](../internal/telemetry/README.md)
- [Integration Tests](../test/integration/README.md)

---

## ✅ Summary

| What | Command |
|------|---------|
| **Build** | `./build-layer.sh` |
| **Enable OTel** | Set `USE_OTEL_METRICS=true` |
| **Configure endpoint** | Set `OTEL_EXPORTER_OTLP_ENDPOINT` |
| **Enable FaaS metrics** | Set `OTEL_LAMBDA_EMIT_SEMCONV=true` |
| **Verify** | Check CloudWatch logs for "OpenTelemetry metrics enabled" |
| **Rollback** | Remove `USE_OTEL_METRICS` or set to `false` |

🎉 **You're all set!**

