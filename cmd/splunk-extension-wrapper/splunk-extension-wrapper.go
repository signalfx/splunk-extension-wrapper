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

package main

import (
	"github.com/splunk/lambda-extension/internal/config"
	"github.com/splunk/lambda-extension/internal/extensionapi"
	"github.com/splunk/lambda-extension/internal/metrics"
	"github.com/splunk/lambda-extension/internal/ossignal"
	"github.com/splunk/lambda-extension/internal/otelmetrics"
	"github.com/splunk/lambda-extension/internal/shutdown"
	"github.com/splunk/lambda-extension/internal/telemetry"
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

// the correct value is set by the go linker (it's done during build using "ldflags")
var gitVersion string

const enabledKey = "SPLUNK_EXTENSION_WRAPPER_ENABLED"
const extensionNameKey = "SPLUNK_EXTENSION_WRAPPER_NAME"
const otelEnabledKey = "USE_OTEL_METRICS"

func enabled() bool {
	s := strings.ToLower(os.Getenv(enabledKey))
	return s != "0" && s != "false"
}

func otelEnabled() bool {
	s := strings.ToLower(os.Getenv(otelEnabledKey))
	return s == "1" || s == "true"
}

func main() {
	ctx := context.Background()
	enabled := enabled()
	useOtel := otelEnabled()

	configuration := config.New()

	initLogging(&configuration)

	ossignal.Watch()

	// Initialize SignalFx metrics (legacy)
	var m *metrics.MetricEmitter = nil
	if enabled && !useOtel {
		m = metrics.New()
		// Log to stderr so it always appears in CloudWatch
		fmt.Fprintln(os.Stderr, "[splunk-extension-wrapper] SignalFx metrics enabled")
	}

	// Initialize OpenTelemetry metrics
	var otelProvider *otelmetrics.Provider = nil
	var telemetrySub *telemetry.TelemetrySubscriber = nil
	if enabled && useOtel {
		var err error
		otelProvider, err = otelmetrics.Setup(ctx)
		if err != nil {
			// Log to stderr so it always appears in CloudWatch
			fmt.Fprintf(os.Stderr, "[splunk-extension-wrapper] Failed to initialize OpenTelemetry: %v\n", err)
			fmt.Fprintln(os.Stderr, "[splunk-extension-wrapper] Falling back to SignalFx metrics")
			m = metrics.New()
			useOtel = false
		} else {
			// Log to stderr so it always appears in CloudWatch
			fmt.Fprintln(os.Stderr, "[splunk-extension-wrapper] OpenTelemetry metrics enabled")
		}
	}

	shutdownCondition := registerApiAndStartMainLoop(enabled, useOtel, m, otelProvider, &telemetrySub, &configuration)

	// Shutdown metrics systems
	if shutdownCondition.IsError() {
		log.SetOutput(os.Stderr)
	}

	log.Println("shutdown reason:", shutdownCondition.Reason())
	log.Println("shutdown message:", shutdownCondition.Message())

	// Shutdown telemetry subscriber first
	if telemetrySub != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := telemetrySub.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down telemetry subscriber: %v", err)
		}
	}

	// Shutdown OpenTelemetry provider
	if otelProvider != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := otelProvider.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down OpenTelemetry: %v", err)
		}
	}

	// Shutdown SignalFx metrics
	if m != nil {
		m.Shutdown(shutdownCondition)
	}
}

func registerApiAndStartMainLoop(enabled bool, useOtel bool, m *metrics.MetricEmitter, otelProvider *otelmetrics.Provider, telemetrySub **telemetry.TelemetrySubscriber, configuration *config.Configuration) (sc shutdown.Condition) {
	var api *extensionapi.RegisteredApi

	defer func() {
		if r := recover(); r != nil {
			log.SetOutput(os.Stderr)
			sc = shutdown.Internal(fmt.Sprintf("%v", r))
			if api != nil {
				api.ExitError(sc.Reason())
			}
		}
	}()

	api, sc = extensionapi.Register(enabled, extensionName(), configuration)

	// Initialize Telemetry API subscriber after registration (need ExtensionID)
	if sc == nil && useOtel && otelProvider != nil {
		ctx := context.Background()
		
		// Create meter and metrics sink
		meter := otelProvider.MeterProvider().Meter("github.com/splunk/lambda-extension")
		metricsSink, err := otelmetrics.NewMetricsSink(meter)
		if err != nil {
			log.Printf("Failed to create metrics sink: %v", err)
		} else {
			// Create and start telemetry subscriber
			*telemetrySub = telemetry.NewTelemetrySubscriber(telemetry.Config{
				ExtensionID: api.ExtensionID(),
				MetricsSink: metricsSink,
			})
			
			if err := (*telemetrySub).Start(ctx); err != nil {
				// Log to stderr so it always appears in CloudWatch
				fmt.Fprintf(os.Stderr, "[splunk-extension-wrapper] Failed to start telemetry subscriber: %v\n", err)
				*telemetrySub = nil
			} else {
				// Log to stderr so it always appears in CloudWatch
				fmt.Fprintln(os.Stderr, "[splunk-extension-wrapper] Telemetry API subscriber started successfully")
			}
		}
	}

	if sc == nil {
		sc = mainLoop(api, m, configuration)
	}

	if sc != nil && sc.IsError() && api != nil {
		api.ExitError(sc.Reason())
	}

	return
}

func mainLoop(api *extensionapi.RegisteredApi, m *metrics.MetricEmitter, configuration *config.Configuration) (sc shutdown.Condition) {
	if m != nil {
		m.SetFunction(api.FunctionName, api.FunctionVersion)
	}

	var event *extensionapi.Event
	event, sc = api.NextEvent()

	for sc == nil {
		if m != nil {
			sc = m.Invoked(event.InvokedFunctionArn, configuration.SplunkFailFast)
		}
		if sc == nil {
			event, sc = api.NextEvent()
		}
	}

	return
}

func initLogging(configuration *config.Configuration) {
	en := extensionName()
	log.SetPrefix("[" + en + "] ")
	log.SetFlags(log.Lmsgprefix)

	log.Printf("%v, version: %v", extensionName(), gitVersion)

	if !configuration.Verbose {
		log.SetOutput(ioutil.Discard)
	}

	log.Printf("lambda region: %v", os.Getenv("AWS_REGION"))
	log.Printf("lambda runtime: %v", os.Getenv("AWS_EXECUTION_ENV"))

	log.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))
	log.Println("NumCPU", runtime.NumCPU())
	log.Println("goroutines on start", runtime.NumGoroutine())

	scanner := bufio.NewScanner(strings.NewReader(configuration.String()))
	for scanner.Scan() {
		log.Print(scanner.Text())
	}
}

func extensionName() string {
	name := os.Getenv(extensionNameKey)
	if name == "" {
		name = path.Base(os.Args[0])
	}
	return name
}
