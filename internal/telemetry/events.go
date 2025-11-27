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
	"encoding/json"
	"time"
)

// EventType represents the type of telemetry event
type EventType string

const (
	// Platform events
	EventPlatformInitStart    EventType = "platform.initStart"
	EventPlatformInitEnd      EventType = "platform.initEnd"
	EventPlatformStart        EventType = "platform.start"
	EventPlatformRuntimeDone  EventType = "platform.runtimeDone"
	EventPlatformReport       EventType = "platform.report"
	EventPlatformShutdown     EventType = "platform.shutdown"
	EventPlatformExtension    EventType = "platform.extension"
	EventPlatformTelemetry    EventType = "platform.telemetrySubscription"
	EventPlatformLogsDropped  EventType = "platform.logsDropped"
	
	// Function events
	EventFunction             EventType = "function"
	
	// Extension events
	EventExtension            EventType = "extension"
)

// TelemetryEvent represents a telemetry event from the Lambda Telemetry API
type TelemetryEvent struct {
	Time   time.Time `json:"time"`
	Type   EventType `json:"type"`
	Record Record    `json:"record"`
}

// Record is the event-specific data
type Record map[string]interface{}

// UnmarshalJSON implements custom unmarshaling for TelemetryEvent
// AWS Lambda Telemetry API sometimes sends 'record' as a JSON string instead of an object
func (e *TelemetryEvent) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal into a temporary struct
	type Alias TelemetryEvent
	aux := &struct {
		Record json.RawMessage `json:"record"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	// Try to unmarshal record as an object first
	var recordMap map[string]interface{}
	if err := json.Unmarshal(aux.Record, &recordMap); err == nil {
		e.Record = recordMap
		return nil
	}
	
	// If that fails, try to unmarshal as a string (escaped JSON)
	var recordStr string
	if err := json.Unmarshal(aux.Record, &recordStr); err != nil {
		// If both fail, return the original error
		return err
	}
	
	// Unmarshal the string content as JSON
	if err := json.Unmarshal([]byte(recordStr), &recordMap); err != nil {
		return err
	}
	
	e.Record = recordMap
	return nil
}

// PlatformInitStartEvent represents platform.initStart event
type PlatformInitStartEvent struct {
	InitializationType string    `json:"initializationType"`
	Phase              string    `json:"phase"`
	RuntimeVersion     string    `json:"runtimeVersion,omitempty"`
	RuntimeVersionArn  string    `json:"runtimeVersionArn,omitempty"`
}

// PlatformInitEndEvent represents platform.initEnd event
type PlatformInitEndEvent struct {
	InitializationType string `json:"initializationType"`
	Phase              string `json:"phase"`
	Status             string `json:"status,omitempty"`
}

// PlatformStartEvent represents platform.start event
type PlatformStartEvent struct {
	RequestID string `json:"requestId"`
	Version   string `json:"version,omitempty"`
}

// PlatformRuntimeDoneEvent represents platform.runtimeDone event
type PlatformRuntimeDoneEvent struct {
	RequestID string                     `json:"requestId"`
	Status    string                     `json:"status"`
	Metrics   *PlatformRuntimeDoneMetrics `json:"metrics,omitempty"`
}

// PlatformRuntimeDoneMetrics contains metrics from runtimeDone event
type PlatformRuntimeDoneMetrics struct {
	DurationMs       float64 `json:"durationMs"`
	ProducedBytes    int64   `json:"producedBytes,omitempty"`
}

// PlatformReportEvent represents platform.report event
type PlatformReportEvent struct {
	RequestID string               `json:"requestId"`
	Status    string               `json:"status,omitempty"`
	Metrics   PlatformReportMetrics `json:"metrics"`
}

// PlatformReportMetrics contains detailed metrics from report event
type PlatformReportMetrics struct {
	DurationMs        float64 `json:"durationMs"`
	BilledDurationMs  int64   `json:"billedDurationMs"`
	MemorySizeMB      int64   `json:"memorySizeMB"`
	MaxMemoryUsedMB   int64   `json:"maxMemoryUsedMB"`
	InitDurationMs    float64 `json:"initDurationMs,omitempty"`
	RestoreDurationMs float64 `json:"restoreDurationMs,omitempty"` // SnapStart restore duration
}

// PlatformShutdownEvent represents platform.shutdown event
type PlatformShutdownEvent struct {
	ShutdownReason string `json:"shutdownReason"`
}

// SubscriptionRequest represents the request to subscribe to telemetry streams
type SubscriptionRequest struct {
	SchemaVersion string              `json:"schemaVersion"`
	Types         []string            `json:"types"`
	Buffering     BufferingConfig     `json:"buffering"`
	Destination   DestinationConfig   `json:"destination"`
}

// BufferingConfig specifies buffering settings for telemetry
type BufferingConfig struct {
	MaxItems  int `json:"maxItems"`
	MaxBytes  int `json:"maxBytes"`
	TimeoutMS int `json:"timeoutMs"`
}

// DestinationConfig specifies where telemetry should be sent
type DestinationConfig struct {
	Protocol string `json:"protocol"`
	URI      string `json:"URI"`
}

