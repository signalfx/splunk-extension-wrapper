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
	"testing"
	"time"
)

// TestTelemetryEventUnmarshalJSON tests custom unmarshaling for both record formats
func TestTelemetryEventUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name        string
		jsonInput   string
		expectError bool
		validate    func(*testing.T, *TelemetryEvent)
	}{
		{
			name: "record as direct object",
			jsonInput: `{
				"time": "2024-01-01T00:00:00Z",
				"type": "platform.report",
				"record": {
					"requestId": "test-123",
					"metrics": {
						"durationMs": 250.5,
						"maxMemoryUsedMB": 128
					}
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *TelemetryEvent) {
				if event.Type != EventPlatformReport {
					t.Errorf("Expected type platform.report, got %s", event.Type)
				}
				if requestID, ok := event.Record["requestId"].(string); !ok || requestID != "test-123" {
					t.Errorf("Expected requestId test-123, got %v", event.Record["requestId"])
				}
			},
		},
		{
			name: "record as escaped JSON string",
			jsonInput: `{
				"time": "2024-01-01T00:00:00Z",
				"type": "platform.report",
				"record": "{\"requestId\":\"test-456\",\"metrics\":{\"durationMs\":150.0}}"
			}`,
			expectError: false,
			validate: func(t *testing.T, event *TelemetryEvent) {
				if requestID, ok := event.Record["requestId"].(string); !ok || requestID != "test-456" {
					t.Errorf("Expected requestId test-456, got %v", event.Record["requestId"])
				}
				if metrics, ok := event.Record["metrics"].(map[string]interface{}); ok {
					if duration, ok := metrics["durationMs"].(float64); !ok || duration != 150.0 {
						t.Errorf("Expected duration 150.0, got %v", metrics["durationMs"])
					}
				} else {
					t.Error("Expected metrics to be map")
				}
			},
		},
		{
			name: "record with nested objects",
			jsonInput: `{
				"time": "2024-01-01T00:00:00Z",
				"type": "platform.runtimeDone",
				"record": {
					"requestId": "test-789",
					"status": "success",
					"metrics": {
						"durationMs": 100.5,
						"producedBytes": 1024
					}
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *TelemetryEvent) {
				metrics, ok := event.Record["metrics"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected metrics to be a map")
				}
				if producedBytes, ok := metrics["producedBytes"].(float64); !ok || producedBytes != 1024 {
					t.Errorf("Expected producedBytes 1024, got %v", metrics["producedBytes"])
				}
			},
		},
		{
			name: "empty record",
			jsonInput: `{
				"time": "2024-01-01T00:00:00Z",
				"type": "platform.initStart",
				"record": {}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *TelemetryEvent) {
				if len(event.Record) != 0 {
					t.Errorf("Expected empty record, got %v", event.Record)
				}
			},
		},
		{
			name: "malformed JSON in string record",
			jsonInput: `{
				"time": "2024-01-01T00:00:00Z",
				"type": "platform.report",
				"record": "{this is not valid JSON}"
			}`,
			expectError: true,
			validate:    nil,
		},
		{
			name: "record as number (invalid)",
			jsonInput: `{
				"time": "2024-01-01T00:00:00Z",
				"type": "platform.report",
				"record": 12345
			}`,
			expectError: true,
			validate:    nil,
		},
		{
			name: "record as array (invalid)",
			jsonInput: `{
				"time": "2024-01-01T00:00:00Z",
				"type": "platform.report",
				"record": ["item1", "item2"]
			}`,
			expectError: true,
			validate:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var event TelemetryEvent
			err := json.Unmarshal([]byte(tc.jsonInput), &event)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if tc.validate != nil {
					tc.validate(t, &event)
				}
			}
		})
	}
}

// TestTelemetryEventTimeParsing tests that time field is parsed correctly
func TestTelemetryEventTimeParsing(t *testing.T) {
	jsonInput := `{
		"time": "2024-01-15T10:30:45.123Z",
		"type": "platform.start",
		"record": {"requestId": "test"}
	}`

	var event TelemetryEvent
	err := json.Unmarshal([]byte(jsonInput), &event)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:45.123Z")
	if !event.Time.Equal(expectedTime) {
		t.Errorf("Expected time %v, got %v", expectedTime, event.Time)
	}
}

// TestRecordWithSpecialCharacters tests handling of special characters in record
func TestRecordWithSpecialCharacters(t *testing.T) {
	jsonInput := `{
		"time": "2024-01-01T00:00:00Z",
		"type": "platform.shutdown",
		"record": {
			"shutdownReason": "spindown\nwith\nnewlines\tand\ttabs"
		}
	}`

	var event TelemetryEvent
	err := json.Unmarshal([]byte(jsonInput), &event)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	reason, ok := event.Record["shutdownReason"].(string)
	if !ok {
		t.Fatal("shutdownReason not found or not a string")
	}
	if reason != "spindown\nwith\nnewlines\tand\ttabs" {
		t.Errorf("Special characters not preserved: %s", reason)
	}
}

