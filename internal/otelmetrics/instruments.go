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
	"fmt"
	"log"
	"os"
	"strconv"

	"go.opentelemetry.io/otel/metric"
)

const (
	// Lambda-specific metric names
	metricInvocation              = "lambda.function.invocation"
	metricInitialization          = "lambda.function.initialization"
	metricInitializationLatency   = "lambda.function.initialization.latency"
	metricShutdown                = "lambda.function.shutdown"
	metricLifetime                = "lambda.function.lifetime"
	metricColdStarts              = "lambda.function.cold_starts"
	metricWarmStarts              = "lambda.function.warm_starts"
	metricResponseSize            = "lambda.function.response_size"
	metricSnapStartRestoreDuration = "lambda.function.snapstart.restore_duration"

	// FaaS semantic convention metric names (optional)
	metricFaasInvocations   = "faas.invocations"
	metricFaasErrors        = "faas.errors"
	metricFaasTimeouts      = "faas.timeouts"
	metricFaasInitDuration  = "faas.init_duration"
	metricFaasInvokeDuration = "faas.duration"  // OTel semconv standard

	// Environment variable to enable FaaS semconv metrics
	envEmitSemconv = "OTEL_LAMBDA_EMIT_SEMCONV"
)

// Instruments holds all OpenTelemetry metric instruments for Lambda extension.
type Instruments struct {
	// Lambda-specific instruments (always available)
	invocation              metric.Int64Counter
	initialization          metric.Int64Counter
	initializationLatency   metric.Int64UpDownCounter
	shutdown                metric.Int64Counter
	lifetime                metric.Int64UpDownCounter
	coldStarts              metric.Int64Counter
	warmStarts              metric.Int64Counter
	responseSize            metric.Int64Histogram
	snapStartRestoreDuration metric.Float64Histogram

	// FaaS semantic convention instruments (optional)
	emitSemconv       bool
	faasInvocations   metric.Int64Counter
	faasErrors        metric.Int64Counter
	faasTimeouts      metric.Int64Counter
	faasInitDuration  metric.Float64Histogram
	faasInvokeDuration metric.Float64Histogram
}

// NewInstruments creates and initializes all metric instruments using the provided MeterProvider.
// It creates instruments for Lambda-specific metrics and optionally creates FaaS semantic
// convention instruments if OTEL_LAMBDA_EMIT_SEMCONV=true.
func NewInstruments(provider *Provider) (*Instruments, error) {
	meter := provider.MeterProvider().Meter("github.com/splunk/lambda-extension")

	instruments := &Instruments{
		emitSemconv: shouldEmitSemconv(),
	}

	var err error

	// Create Lambda-specific instruments
	instruments.invocation, err = meter.Int64Counter(
		metricInvocation,
		metric.WithDescription("Number of Lambda function invocations"),
		metric.WithUnit("{invocation}"),
	)
	if err != nil {
		return nil, err
	}

	instruments.initialization, err = meter.Int64Counter(
		metricInitialization,
		metric.WithDescription("Number of Lambda environment initializations"),
		metric.WithUnit("{initialization}"),
	)
	if err != nil {
		return nil, err
	}

	instruments.initializationLatency, err = meter.Int64UpDownCounter(
		metricInitializationLatency,
		metric.WithDescription("Lambda cold start initialization latency"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	instruments.shutdown, err = meter.Int64Counter(
		metricShutdown,
		metric.WithDescription("Number of Lambda environment shutdowns"),
		metric.WithUnit("{shutdown}"),
	)
	if err != nil {
		return nil, err
	}

	instruments.lifetime, err = meter.Int64UpDownCounter(
		metricLifetime,
		metric.WithDescription("Total lifetime of Lambda environment"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	instruments.coldStarts, err = meter.Int64Counter(
		metricColdStarts,
		metric.WithDescription("Number of cold starts (on-demand initialization)"),
		metric.WithUnit("{cold_start}"),
	)
	if err != nil {
		return nil, err
	}

	instruments.warmStarts, err = meter.Int64Counter(
		metricWarmStarts,
		metric.WithDescription("Number of warm starts (snap-start initialization)"),
		metric.WithUnit("{warm_start}"),
	)
	if err != nil {
		return nil, err
	}

	instruments.responseSize, err = meter.Int64Histogram(
		metricResponseSize,
		metric.WithDescription("Lambda function response payload size"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	instruments.snapStartRestoreDuration, err = meter.Float64Histogram(
		metricSnapStartRestoreDuration,
		metric.WithDescription("SnapStart restore duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	// Create FaaS semantic convention instruments if enabled
	if instruments.emitSemconv {
		if err := instruments.createSemconvInstruments(meter); err != nil {
			log.Printf("[WARN] Failed to create FaaS semconv instruments: %v", err)
		} else {
			// Log to stderr so it always appears in CloudWatch logs
			fmt.Fprintln(os.Stderr, "[splunk-extension-wrapper] [INFO] FaaS semantic convention metrics enabled")
		}
	}

	return instruments, nil
}

// createSemconvInstruments creates the optional FaaS semantic convention instruments
func (i *Instruments) createSemconvInstruments(meter metric.Meter) error {
	var err error

	i.faasInvocations, err = meter.Int64Counter(
		metricFaasInvocations,
		metric.WithDescription("Number of FaaS invocations"),
		metric.WithUnit("{invocation}"),
	)
	if err != nil {
		return err
	}

	i.faasErrors, err = meter.Int64Counter(
		metricFaasErrors,
		metric.WithDescription("Number of FaaS invocation errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	i.faasTimeouts, err = meter.Int64Counter(
		metricFaasTimeouts,
		metric.WithDescription("Number of FaaS invocation timeouts"),
		metric.WithUnit("{timeout}"),
	)
	if err != nil {
		return err
	}

	i.faasInitDuration, err = meter.Float64Histogram(
		metricFaasInitDuration,
		metric.WithDescription("FaaS function initialization duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	i.faasInvokeDuration, err = meter.Float64Histogram(
		metricFaasInvokeDuration,
		metric.WithDescription("FaaS function invocation duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	return nil
}

// Invocation returns the lambda.function.invocation counter.
// This metric tracks the number of Lambda function invocations.
func (i *Instruments) Invocation() metric.Int64Counter {
	return i.invocation
}

// Initialization returns the lambda.function.initialization counter.
// This metric tracks the number of Lambda environment initializations (cold starts).
func (i *Instruments) Initialization() metric.Int64Counter {
	return i.initialization
}

// InitializationLatency returns the lambda.function.initialization.latency up-down counter.
// This metric tracks the cold start initialization latency in milliseconds.
// Use as a gauge by setting the value directly.
func (i *Instruments) InitializationLatency() metric.Int64UpDownCounter {
	return i.initializationLatency
}

// Shutdown returns the lambda.function.shutdown counter.
// This metric tracks the number of Lambda environment shutdowns.
func (i *Instruments) Shutdown() metric.Int64Counter {
	return i.shutdown
}

// Lifetime returns the lambda.function.lifetime up-down counter.
// This metric tracks the total lifetime of the Lambda environment in milliseconds.
// Use as a gauge by setting the value directly.
func (i *Instruments) Lifetime() metric.Int64UpDownCounter {
	return i.lifetime
}

// ColdStarts returns the lambda.function.cold_starts counter.
// This metric tracks the number of cold starts (on-demand initialization).
func (i *Instruments) ColdStarts() metric.Int64Counter {
	return i.coldStarts
}

// WarmStarts returns the lambda.function.warm_starts counter.
// This metric tracks the number of warm starts (snap-start initialization).
func (i *Instruments) WarmStarts() metric.Int64Counter {
	return i.warmStarts
}

// ResponseSize returns the lambda.function.response_size histogram.
// This metric tracks the Lambda function response payload size in bytes.
func (i *Instruments) ResponseSize() metric.Int64Histogram {
	return i.responseSize
}

// SnapStartRestoreDuration returns the lambda.function.snapstart.restore_duration histogram.
// This metric tracks the SnapStart restore duration in milliseconds.
func (i *Instruments) SnapStartRestoreDuration() metric.Float64Histogram {
	return i.snapStartRestoreDuration
}

// FaasInvocations returns the faas.invocations counter (if enabled).
// Returns nil if OTEL_LAMBDA_EMIT_SEMCONV is not set to true.
func (i *Instruments) FaasInvocations() metric.Int64Counter {
	if !i.emitSemconv {
		return nil
	}
	return i.faasInvocations
}

// FaasErrors returns the faas.errors counter (if enabled).
// Returns nil if OTEL_LAMBDA_EMIT_SEMCONV is not set to true.
func (i *Instruments) FaasErrors() metric.Int64Counter {
	if !i.emitSemconv {
		return nil
	}
	return i.faasErrors
}

// FaasTimeouts returns the faas.timeouts counter (if enabled).
// Returns nil if OTEL_LAMBDA_EMIT_SEMCONV is not set to true.
func (i *Instruments) FaasTimeouts() metric.Int64Counter {
	if !i.emitSemconv {
		return nil
	}
	return i.faasTimeouts
}

// FaasInitDuration returns the faas.init_duration histogram (if enabled).
// Values should be recorded in seconds.
// Returns nil if OTEL_LAMBDA_EMIT_SEMCONV is not set to true.
func (i *Instruments) FaasInitDuration() metric.Float64Histogram {
	if !i.emitSemconv {
		return nil
	}
	return i.faasInitDuration
}

// FaasInvokeDuration returns the faas.duration histogram (if enabled).
// Values should be recorded in seconds.
// Returns nil if OTEL_LAMBDA_EMIT_SEMCONV is not set to true.
func (i *Instruments) FaasInvokeDuration() metric.Float64Histogram {
	if !i.emitSemconv {
		return nil
	}
	return i.faasInvokeDuration
}

// EmitSemconv returns whether FaaS semantic convention metrics are enabled.
func (i *Instruments) EmitSemconv() bool {
	return i.emitSemconv
}

// shouldEmitSemconv checks if FaaS semantic convention metrics should be emitted
func shouldEmitSemconv() bool {
	val := os.Getenv(envEmitSemconv)
	if val == "" {
		return false
	}
	
	// Try parsing as boolean
	if enabled, err := strconv.ParseBool(val); err == nil {
		return enabled
	}
	
	return false
}

