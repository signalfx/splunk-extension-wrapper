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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/splunk/lambda-extension/internal/otelmetrics"
)

const (
	// Default listener configuration
	defaultListenerHost = "0.0.0.0"  // Listen on all interfaces
	defaultListenerPort = "4243"
	
	// AWS Lambda requires 'sandbox.localdomain' in Telemetry API subscription
	telemetrySubscriptionHost = "sandbox.localdomain"
	
	// Subscription configuration
	schemaVersion   = "2022-12-13"
	bufferMaxItems  = 1000  // AWS requires minimum 1000, maximum 10000
	bufferMaxBytes  = 256 * 1024  // 256KB
	bufferTimeoutMS = 500
	
	// Telemetry API endpoint
	telemetryAPIEndpoint = "http://%s/2022-07-01/telemetry"
)

// TelemetrySubscriber manages the Lambda Telemetry API subscription
type TelemetrySubscriber struct {
	listenerHost string
	listenerPort string
	extensionID  string
	metricsSink  otelmetrics.TelemetryMetricsSink
	
	server   *http.Server
	state    *executionState
	mu       sync.RWMutex
	
	ctx    context.Context
	cancel context.CancelFunc
}

// executionState tracks the current Lambda execution environment state
type executionState struct {
	initStartTime   time.Time
	initEndTime     time.Time
	firstInvokeTime time.Time
	lastRequestID   string
	lastReport      *PlatformReportMetrics
}

// Config holds configuration for the TelemetrySubscriber
type Config struct {
	ListenerHost string
	ListenerPort string
	ExtensionID  string
	MetricsSink  otelmetrics.TelemetryMetricsSink
}

// NewTelemetrySubscriber creates a new TelemetrySubscriber
func NewTelemetrySubscriber(cfg Config) *TelemetrySubscriber {
	if cfg.ListenerHost == "" {
		cfg.ListenerHost = defaultListenerHost
	}
	if cfg.ListenerPort == "" {
		cfg.ListenerPort = defaultListenerPort
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TelemetrySubscriber{
		listenerHost: cfg.ListenerHost,
		listenerPort: cfg.ListenerPort,
		extensionID:  cfg.ExtensionID,
		metricsSink:  cfg.MetricsSink,
		state:        &executionState{},
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins listening for telemetry events and subscribes to the Telemetry API
func (ts *TelemetrySubscriber) Start(ctx context.Context) error {
	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", ts.handleTelemetry)
	
	ts.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", ts.listenerHost, ts.listenerPort),
		Handler: mux,
	}
	
	// Start HTTP server in background
	errChan := make(chan error, 1)
	go func() {
		log.Printf("[INFO] Starting telemetry listener on %s:%s", ts.listenerHost, ts.listenerPort)
		if err := ts.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("telemetry listener failed: %w", err)
		}
	}()
	
	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Subscribe to Telemetry API
	if err := ts.subscribe(); err != nil {
		ts.server.Close()
		return fmt.Errorf("failed to subscribe to telemetry API: %w", err)
	}
	
	// Check for startup errors
	select {
	case err := <-errChan:
		return err
	default:
		log.Println("[INFO] Telemetry subscriber started successfully")
		return nil
	}
}

// Shutdown gracefully shuts down the telemetry subscriber
func (ts *TelemetrySubscriber) Shutdown(ctx context.Context) error {
	log.Println("[INFO] Shutting down telemetry subscriber")
	ts.cancel()
	
	if ts.server != nil {
		return ts.server.Shutdown(ctx)
	}
	return nil
}

// subscribe sends a subscription request to the Lambda Telemetry API
func (ts *TelemetrySubscriber) subscribe() error {
	runtimeAPI := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	if runtimeAPI == "" {
		return fmt.Errorf("AWS_LAMBDA_RUNTIME_API environment variable not set")
	}
	
	subscriptionURL := fmt.Sprintf(telemetryAPIEndpoint, runtimeAPI)
	
	subscription := SubscriptionRequest{
		SchemaVersion: schemaVersion,
		Types:         []string{"platform"},  // Only subscribe to platform events (metrics data)
		Buffering: BufferingConfig{
			MaxItems:  bufferMaxItems,
			MaxBytes:  bufferMaxBytes,
			TimeoutMS: bufferTimeoutMS,
		},
		Destination: DestinationConfig{
			Protocol: "HTTP",
			// AWS Lambda requires 'sandbox.localdomain' hostname in subscription
			URI:      fmt.Sprintf("http://%s:%s", telemetrySubscriptionHost, ts.listenerPort),
		},
	}
	
	body, err := json.Marshal(subscription)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription request: %w", err)
	}
	
	req, err := http.NewRequest(http.MethodPut, subscriptionURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create subscription request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Lambda-Extension-Identifier", ts.extensionID)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send subscription request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("subscription failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	log.Printf("[INFO] Successfully subscribed to Lambda Telemetry API")
	return nil
}

// handleTelemetry handles incoming telemetry events
func (ts *TelemetrySubscriber) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read telemetry request body: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	// Parse events
	var events []TelemetryEvent
	if err := json.Unmarshal(body, &events); err != nil {
		log.Printf("[ERROR] Failed to parse telemetry events: %v", err)
		// Don't return error to Lambda, just acknowledge receipt
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Process each event
	for _, event := range events {
		ts.processEvent(event)
	}
	
	w.WriteHeader(http.StatusOK)
}

// processEvent processes a single telemetry event
func (ts *TelemetrySubscriber) processEvent(event TelemetryEvent) {
	ctx := ts.ctx
	
	switch event.Type {
	case EventPlatformInitStart:
		ts.handleInitStart(ctx, event)
	case EventPlatformInitEnd:
		ts.handleInitEnd(ctx, event)
	case EventPlatformStart:
		ts.handleStart(ctx, event)
	case EventPlatformRuntimeDone:
		ts.handleRuntimeDone(ctx, event)
	case EventPlatformReport:
		ts.handleReport(ctx, event)
	case EventPlatformShutdown:
		ts.handleShutdown(ctx, event)
	default:
		// Ignore other event types
	}
}

// handleInitStart processes platform.initStart events
func (ts *TelemetrySubscriber) handleInitStart(ctx context.Context, event TelemetryEvent) {
	initializationType, _ := event.Record["initializationType"].(string)
	
	ts.mu.Lock()
	ts.state.initStartTime = event.Time
	ts.mu.Unlock()
	
	if ts.metricsSink != nil {
		ts.metricsSink.RecordInitStart(ctx, event.Time, initializationType)
	}
}

// handleInitEnd processes platform.initEnd events
func (ts *TelemetrySubscriber) handleInitEnd(ctx context.Context, event TelemetryEvent) {
	initializationType, _ := event.Record["initializationType"].(string)
	
	ts.mu.Lock()
	ts.state.initEndTime = event.Time
	ts.mu.Unlock()
	
	if ts.metricsSink != nil {
		ts.metricsSink.RecordInitEnd(ctx, event.Time, initializationType)
	}
}

// handleStart processes platform.start events
func (ts *TelemetrySubscriber) handleStart(ctx context.Context, event TelemetryEvent) {
	requestID, _ := event.Record["requestId"].(string)
	
	ts.mu.Lock()
	if ts.state.firstInvokeTime.IsZero() {
		ts.state.firstInvokeTime = event.Time
	}
	ts.state.lastRequestID = requestID
	ts.mu.Unlock()
	
	if ts.metricsSink != nil {
		ts.metricsSink.RecordStart(ctx, event.Time, requestID)
	}
}

// handleRuntimeDone processes platform.runtimeDone events
func (ts *TelemetrySubscriber) handleRuntimeDone(ctx context.Context, event TelemetryEvent) {
	requestID, _ := event.Record["requestId"].(string)
	status, _ := event.Record["status"].(string)
	
	var durationMs int64
	var producedBytes int64
	if metrics, ok := event.Record["metrics"].(map[string]interface{}); ok {
		if durationMsFloat, ok := metrics["durationMs"].(float64); ok {
			durationMs = int64(durationMsFloat)
		}
		if producedBytesFloat, ok := metrics["producedBytes"].(float64); ok {
			producedBytes = int64(producedBytesFloat)
		}
	}
	
	if ts.metricsSink != nil {
		ts.metricsSink.RecordRuntimeDone(ctx, event.Time, requestID, status, durationMs, producedBytes)
	}
}

// handleReport processes platform.report events
func (ts *TelemetrySubscriber) handleReport(ctx context.Context, event TelemetryEvent) {
	requestID, _ := event.Record["requestId"].(string)
	
	var reportMetrics otelmetrics.ReportMetrics
	
	if metrics, ok := event.Record["metrics"].(map[string]interface{}); ok {
		if durationMs, ok := metrics["durationMs"].(float64); ok {
			reportMetrics.DurationMs = durationMs
		}
		if billedDur, ok := metrics["billedDurationMs"].(float64); ok {
			reportMetrics.BilledDurationMs = int64(billedDur)
		}
		if memSize, ok := metrics["memorySizeMB"].(float64); ok {
			reportMetrics.MemorySizeMB = int64(memSize)
		}
		if maxMem, ok := metrics["maxMemoryUsedMB"].(float64); ok {
			reportMetrics.MaxMemoryUsedMB = int64(maxMem)
		}
		if initDur, ok := metrics["initDurationMs"].(float64); ok {
			reportMetrics.InitDurationMs = initDur
		}
		if restoreDur, ok := metrics["restoreDurationMs"].(float64); ok {
			reportMetrics.RestoreDurationMs = restoreDur
		}
		
		// Store report for later use
		ts.mu.Lock()
		ts.state.lastReport = &PlatformReportMetrics{
			DurationMs:        reportMetrics.DurationMs,
			BilledDurationMs:  reportMetrics.BilledDurationMs,
			MemorySizeMB:      reportMetrics.MemorySizeMB,
			MaxMemoryUsedMB:   reportMetrics.MaxMemoryUsedMB,
			InitDurationMs:    reportMetrics.InitDurationMs,
			RestoreDurationMs: reportMetrics.RestoreDurationMs,
		}
		ts.mu.Unlock()
	}
	
	if ts.metricsSink != nil {
		ts.metricsSink.RecordReport(ctx, event.Time, requestID, reportMetrics)
	}
}

// handleShutdown processes platform.shutdown events
func (ts *TelemetrySubscriber) handleShutdown(ctx context.Context, event TelemetryEvent) {
	reason, _ := event.Record["shutdownReason"].(string)
	
	if ts.metricsSink != nil {
		ts.metricsSink.RecordShutdown(ctx, event.Time, reason)
	}
}

// GetState returns the current execution state (for debugging/monitoring)
func (ts *TelemetrySubscriber) GetState() executionState {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return *ts.state
}

