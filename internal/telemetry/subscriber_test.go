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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/splunk/lambda-extension/internal/otelmetrics"
)

// MockMetricsSink implements TelemetryMetricsSink for testing
type MockMetricsSink struct {
	mu sync.Mutex
	
	initStartCalls      []time.Time
	initEndCalls        []time.Time
	startCalls          []StartCall
	runtimeDoneCalls    []RuntimeDoneCall
	reportCalls         []ReportCall
	shutdownCalls       []ShutdownCall
}

type StartCall struct {
	Timestamp time.Time
	RequestID string
}

type RuntimeDoneCall struct {
	Timestamp time.Time
	RequestID string
	Status    string
	DurationMs int64
}

type ReportCall struct {
	Timestamp time.Time
	RequestID string
	Metrics   otelmetrics.ReportMetrics
}

type ShutdownCall struct {
	Timestamp time.Time
	Reason    string
}

func NewMockMetricsSink() *MockMetricsSink {
	return &MockMetricsSink{
		initStartCalls:   make([]time.Time, 0),
		initEndCalls:     make([]time.Time, 0),
		startCalls:       make([]StartCall, 0),
		runtimeDoneCalls: make([]RuntimeDoneCall, 0),
		reportCalls:      make([]ReportCall, 0),
		shutdownCalls:    make([]ShutdownCall, 0),
	}
}

func (m *MockMetricsSink) RecordInitStart(ctx context.Context, timestamp time.Time, initializationType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initStartCalls = append(m.initStartCalls, timestamp)
}

func (m *MockMetricsSink) RecordInitEnd(ctx context.Context, timestamp time.Time, initializationType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initEndCalls = append(m.initEndCalls, timestamp)
}

func (m *MockMetricsSink) RecordStart(ctx context.Context, timestamp time.Time, requestID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalls = append(m.startCalls, StartCall{Timestamp: timestamp, RequestID: requestID})
}

func (m *MockMetricsSink) RecordRuntimeDone(ctx context.Context, timestamp time.Time, requestID string, status string, durationMs int64, producedBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runtimeDoneCalls = append(m.runtimeDoneCalls, RuntimeDoneCall{
		Timestamp:  timestamp,
		RequestID:  requestID,
		Status:     status,
		DurationMs: durationMs,
	})
}

func (m *MockMetricsSink) RecordReport(ctx context.Context, timestamp time.Time, requestID string, metrics otelmetrics.ReportMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reportCalls = append(m.reportCalls, ReportCall{
		Timestamp: timestamp,
		RequestID: requestID,
		Metrics:   metrics,
	})
}

func (m *MockMetricsSink) RecordShutdown(ctx context.Context, timestamp time.Time, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdownCalls = append(m.shutdownCalls, ShutdownCall{Timestamp: timestamp, Reason: reason})
}

func (m *MockMetricsSink) GetCalls() ([]time.Time, []time.Time, []StartCall, []RuntimeDoneCall, []ReportCall, []ShutdownCall) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.initStartCalls, m.initEndCalls, m.startCalls, m.runtimeDoneCalls, m.reportCalls, m.shutdownCalls
}

// Test helpers to create synthetic events
func createInitStartEvent(timestamp time.Time) TelemetryEvent {
	return TelemetryEvent{
		Time: timestamp,
		Type: EventPlatformInitStart,
		Record: Record{
			"initializationType": "on-demand",
			"phase":              "init",
		},
	}
}

func createInitEndEvent(timestamp time.Time) TelemetryEvent {
	return TelemetryEvent{
		Time: timestamp,
		Type: EventPlatformInitEnd,
		Record: Record{
			"initializationType": "on-demand",
			"phase":              "init",
		},
	}
}

func createStartEvent(timestamp time.Time, requestID string) TelemetryEvent {
	return TelemetryEvent{
		Time: timestamp,
		Type: EventPlatformStart,
		Record: Record{
			"requestId": requestID,
		},
	}
}

func createRuntimeDoneEvent(timestamp time.Time, requestID, status string, durationMs float64) TelemetryEvent {
	return TelemetryEvent{
		Time: timestamp,
		Type: EventPlatformRuntimeDone,
		Record: Record{
			"requestId": requestID,
			"status":    status,
			"metrics": map[string]interface{}{
				"durationMs": durationMs,
			},
		},
	}
}

func createReportEvent(timestamp time.Time, requestID string, durationMs, billedMs float64, maxMemMB, memSizeMB float64) TelemetryEvent {
	return TelemetryEvent{
		Time: timestamp,
		Type: EventPlatformReport,
		Record: Record{
			"requestId": requestID,
			"metrics": map[string]interface{}{
				"durationMs":       durationMs,
				"billedDurationMs": billedMs,
				"maxMemoryUsedMB":  maxMemMB,
				"memorySizeMB":     memSizeMB,
			},
		},
	}
}

func createShutdownEvent(timestamp time.Time, reason string) TelemetryEvent {
	return TelemetryEvent{
		Time: timestamp,
		Type: EventPlatformShutdown,
		Record: Record{
			"shutdownReason": reason,
		},
	}
}

// Tests

func TestTelemetrySubscriberProcessInitEvents(t *testing.T) {
	mockSink := NewMockMetricsSink()
	
	ts := &TelemetrySubscriber{
		metricsSink: mockSink,
		state:       &executionState{},
		ctx:         context.Background(),
	}
	
	now := time.Now()
	
	// Process init events
	ts.processEvent(createInitStartEvent(now))
	ts.processEvent(createInitEndEvent(now.Add(100 * time.Millisecond)))
	
	// Verify calls
	initStarts, initEnds, _, _, _, _ := mockSink.GetCalls()
	
	if len(initStarts) != 1 {
		t.Errorf("Expected 1 initStart call, got %d", len(initStarts))
	}
	if len(initEnds) != 1 {
		t.Errorf("Expected 1 initEnd call, got %d", len(initEnds))
	}
	
	if !initStarts[0].Equal(now) {
		t.Errorf("Expected initStart timestamp %v, got %v", now, initStarts[0])
	}
}

func TestTelemetrySubscriberProcessInvocationEvents(t *testing.T) {
	mockSink := NewMockMetricsSink()
	
	ts := &TelemetrySubscriber{
		metricsSink: mockSink,
		state:       &executionState{},
		ctx:         context.Background(),
	}
	
	now := time.Now()
	requestID := "test-request-123"
	
	// Process invocation events
	ts.processEvent(createStartEvent(now, requestID))
	ts.processEvent(createRuntimeDoneEvent(now.Add(250*time.Millisecond), requestID, "success", 250.0))
	
	// Verify calls
	_, _, starts, runtimeDones, _, _ := mockSink.GetCalls()
	
	if len(starts) != 1 {
		t.Errorf("Expected 1 start call, got %d", len(starts))
	}
	if len(runtimeDones) != 1 {
		t.Errorf("Expected 1 runtimeDone call, got %d", len(runtimeDones))
	}
	
	if starts[0].RequestID != requestID {
		t.Errorf("Expected requestID %s, got %s", requestID, starts[0].RequestID)
	}
	
	if runtimeDones[0].Status != "success" {
		t.Errorf("Expected status 'success', got %s", runtimeDones[0].Status)
	}
	if runtimeDones[0].DurationMs != 250 {
		t.Errorf("Expected duration 250ms, got %d", runtimeDones[0].DurationMs)
	}
}

func TestTelemetrySubscriberProcessReportEvent(t *testing.T) {
	mockSink := NewMockMetricsSink()
	
	ts := &TelemetrySubscriber{
		metricsSink: mockSink,
		state:       &executionState{},
		ctx:         context.Background(),
	}
	
	now := time.Now()
	requestID := "test-request-456"
	
	// Process report event
	ts.processEvent(createReportEvent(now, requestID, 250.5, 300.0, 128.0, 512.0))
	
	// Verify calls
	_, _, _, _, reports, _ := mockSink.GetCalls()
	
	if len(reports) != 1 {
		t.Errorf("Expected 1 report call, got %d", len(reports))
	}
	
	report := reports[0]
	if report.RequestID != requestID {
		t.Errorf("Expected requestID %s, got %s", requestID, report.RequestID)
	}
	if report.Metrics.DurationMs != 250.5 {
		t.Errorf("Expected duration 250.5ms, got %f", report.Metrics.DurationMs)
	}
	if report.Metrics.BilledDurationMs != 300 {
		t.Errorf("Expected billed duration 300ms, got %d", report.Metrics.BilledDurationMs)
	}
	if report.Metrics.MaxMemoryUsedMB != 128 {
		t.Errorf("Expected max memory 128MB, got %d", report.Metrics.MaxMemoryUsedMB)
	}
}

func TestTelemetrySubscriberProcessShutdownEvent(t *testing.T) {
	mockSink := NewMockMetricsSink()
	
	ts := &TelemetrySubscriber{
		metricsSink: mockSink,
		state:       &executionState{},
		ctx:         context.Background(),
	}
	
	now := time.Now()
	reason := "spindown"
	
	// Process shutdown event
	ts.processEvent(createShutdownEvent(now, reason))
	
	// Verify calls
	_, _, _, _, _, shutdowns := mockSink.GetCalls()
	
	if len(shutdowns) != 1 {
		t.Errorf("Expected 1 shutdown call, got %d", len(shutdowns))
	}
	
	if shutdowns[0].Reason != reason {
		t.Errorf("Expected shutdown reason %s, got %s", reason, shutdowns[0].Reason)
	}
}

func TestTelemetrySubscriberCompleteLifecycle(t *testing.T) {
	mockSink := NewMockMetricsSink()
	
	ts := &TelemetrySubscriber{
		metricsSink: mockSink,
		state:       &executionState{},
		ctx:         context.Background(),
	}
	
	now := time.Now()
	
	// Simulate complete lifecycle
	ts.processEvent(createInitStartEvent(now))
	ts.processEvent(createInitEndEvent(now.Add(100 * time.Millisecond)))
	
	// First invocation
	ts.processEvent(createStartEvent(now.Add(200*time.Millisecond), "req-1"))
	ts.processEvent(createRuntimeDoneEvent(now.Add(450*time.Millisecond), "req-1", "success", 250.0))
	ts.processEvent(createReportEvent(now.Add(500*time.Millisecond), "req-1", 250.5, 300.0, 128.0, 512.0))
	
	// Second invocation
	ts.processEvent(createStartEvent(now.Add(600*time.Millisecond), "req-2"))
	ts.processEvent(createRuntimeDoneEvent(now.Add(750*time.Millisecond), "req-2", "error", 150.0))
	ts.processEvent(createReportEvent(now.Add(800*time.Millisecond), "req-2", 150.0, 200.0, 96.0, 512.0))
	
	// Shutdown
	ts.processEvent(createShutdownEvent(now.Add(1*time.Second), "timeout"))
	
	// Verify all calls were made
	initStarts, initEnds, starts, runtimeDones, reports, shutdowns := mockSink.GetCalls()
	
	if len(initStarts) != 1 {
		t.Errorf("Expected 1 initStart call, got %d", len(initStarts))
	}
	if len(initEnds) != 1 {
		t.Errorf("Expected 1 initEnd call, got %d", len(initEnds))
	}
	if len(starts) != 2 {
		t.Errorf("Expected 2 start calls, got %d", len(starts))
	}
	if len(runtimeDones) != 2 {
		t.Errorf("Expected 2 runtimeDone calls, got %d", len(runtimeDones))
	}
	if len(reports) != 2 {
		t.Errorf("Expected 2 report calls, got %d", len(reports))
	}
	if len(shutdowns) != 1 {
		t.Errorf("Expected 1 shutdown call, got %d", len(shutdowns))
	}
	
	// Verify statuses
	if runtimeDones[0].Status != "success" {
		t.Errorf("Expected first invocation status 'success', got %s", runtimeDones[0].Status)
	}
	if runtimeDones[1].Status != "error" {
		t.Errorf("Expected second invocation status 'error', got %s", runtimeDones[1].Status)
	}
}

func TestTelemetrySubscriberDifferentStatuses(t *testing.T) {
	testCases := []struct {
		name   string
		status string
	}{
		{"success", "success"},
		{"error", "error"},
		{"failure", "failure"},
		{"timeout", "timeout"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockSink := NewMockMetricsSink()
			
			ts := &TelemetrySubscriber{
				metricsSink: mockSink,
				state:       &executionState{},
				ctx:         context.Background(),
			}
			
			now := time.Now()
			requestID := "req-" + tc.name
			
			ts.processEvent(createStartEvent(now, requestID))
			ts.processEvent(createRuntimeDoneEvent(now.Add(100*time.Millisecond), requestID, tc.status, 100.0))
			
			_, _, _, runtimeDones, _, _ := mockSink.GetCalls()
			
			if len(runtimeDones) != 1 {
				t.Fatalf("Expected 1 runtimeDone call, got %d", len(runtimeDones))
			}
			
			if runtimeDones[0].Status != tc.status {
				t.Errorf("Expected status %s, got %s", tc.status, runtimeDones[0].Status)
			}
		})
	}
}

// TestSubscriberHTTPErrors tests HTTP handler with invalid inputs
func TestSubscriberHTTPErrors(t *testing.T) {
	mockSink := NewMockMetricsSink()
	
	ts := &TelemetrySubscriber{
		metricsSink: mockSink,
		state:       &executionState{},
		ctx:         context.Background(),
	}
	
	testCases := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid JSON",
			method:         "POST",
			body:           `{this is not valid JSON}`,
			expectedStatus: 200, // Returns 200 even on error to not fail Lambda
		},
		{
			name:           "empty body",
			method:         "POST",
			body:           ``,
			expectedStatus: 200,
		},
		{
			name:           "wrong HTTP method",
			method:         "GET",
			body:           `[]`,
			expectedStatus: 405, // Method not allowed
		},
		{
			name:           "PUT method",
			method:         "PUT",
			body:           `[]`,
			expectedStatus: 405,
		},
		{
			name:           "malformed event array",
			method:         "POST",
			body:           `[{"time": "invalid", "type": "platform.report"}]`,
			expectedStatus: 200, // Handles gracefully
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTTP request
			req := &http.Request{
				Method: tc.method,
				Body:   ioutil.NopCloser(strings.NewReader(tc.body)),
			}
			
			// Create response recorder
			rr := &testResponseRecorder{statusCode: 200}
			
			// Call handler
			ts.handleTelemetry(rr, req)
			
			if rr.statusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.statusCode)
			}
		})
	}
}

// TestSubscriberStartFailure tests port already in use scenario
func TestSubscriberStartFailure(t *testing.T) {
	mockSink := NewMockMetricsSink()
	
	// Start first subscriber on default port
	ts1 := NewTelemetrySubscriber(Config{
		ExtensionID: "test-extension-1",
		MetricsSink: mockSink,
	})
	
	// Mock environment
	os.Setenv("AWS_LAMBDA_RUNTIME_API", "127.0.0.1:9001")
	defer os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
	
	ctx := context.Background()
	
	// First subscriber should succeed in creating server
	// (but will fail to subscribe since no runtime API is running)
	err1 := ts1.Start(ctx)
	if err1 == nil {
		defer ts1.Shutdown(ctx)
		t.Fatal("Expected error due to missing runtime API")
	}
	
	t.Logf("Expected failure (no runtime API): %v", err1)
	
	// Note: Testing actual port collision requires more setup,
	// but this verifies the error handling path
}

// TestSubscribeAPIFailure tests Telemetry API error responses
func TestSubscribeAPIFailure(t *testing.T) {
	mockSink := NewMockMetricsSink()
	
	ts := &TelemetrySubscriber{
		listenerHost: "127.0.0.1",
		listenerPort: "4243",
		extensionID:  "test-extension",
		metricsSink:  mockSink,
		state:        &executionState{},
	}
	
	testCases := []struct {
		name            string
		runtimeAPI      string
		expectError     bool
		errorContains   string
	}{
		{
			name:          "missing runtime API env var",
			runtimeAPI:    "",
			expectError:   true,
			errorContains: "AWS_LAMBDA_RUNTIME_API",
		},
		{
			name:          "invalid runtime API format",
			runtimeAPI:    "not-a-valid-url",
			expectError:   true,
			errorContains: "", // Will fail on connection
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.runtimeAPI == "" {
				os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
			} else {
				os.Setenv("AWS_LAMBDA_RUNTIME_API", tc.runtimeAPI)
			}
			defer os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
			
			err := ts.subscribe()
			
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if tc.expectError && err != nil && tc.errorContains != "" {
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tc.errorContains, err)
				}
			}
		})
	}
}

// testResponseRecorder is a simple HTTP response recorder for testing
type testResponseRecorder struct {
	statusCode int
	headers    http.Header
	body       []byte
}

func (r *testResponseRecorder) Header() http.Header {
	if r.headers == nil {
		r.headers = make(http.Header)
	}
	return r.headers
}

func (r *testResponseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}

func (r *testResponseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

