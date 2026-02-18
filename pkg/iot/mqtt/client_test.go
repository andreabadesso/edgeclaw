package mqtt

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFleetAlert_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	alert := FleetAlert{
		NodeID:     "edge-01",
		Severity:   "critical",
		SensorType: "temperature",
		Message:    "sensor exceeded threshold",
		Value:      98.6,
		Threshold:  95.0,
		Timestamp:  ts,
	}

	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded FleetAlert
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.NodeID != alert.NodeID {
		t.Errorf("NodeID = %q, want %q", decoded.NodeID, alert.NodeID)
	}
	if decoded.Severity != alert.Severity {
		t.Errorf("Severity = %q, want %q", decoded.Severity, alert.Severity)
	}
	if decoded.SensorType != alert.SensorType {
		t.Errorf("SensorType = %q, want %q", decoded.SensorType, alert.SensorType)
	}
	if decoded.Message != alert.Message {
		t.Errorf("Message = %q, want %q", decoded.Message, alert.Message)
	}
	if decoded.Value != alert.Value {
		t.Errorf("Value = %f, want %f", decoded.Value, alert.Value)
	}
	if decoded.Threshold != alert.Threshold {
		t.Errorf("Threshold = %f, want %f", decoded.Threshold, alert.Threshold)
	}
	if !decoded.Timestamp.Equal(alert.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, alert.Timestamp)
	}
}

func TestFleetAlert_JSONOmitsZeroOptionalFields(t *testing.T) {
	alert := FleetAlert{
		NodeID:     "edge-02",
		Severity:   "info",
		SensorType: "humidity",
		Message:    "normal reading",
		Timestamp:  time.Now().UTC(),
	}

	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	// value and threshold should be omitted when zero (omitempty)
	if _, exists := raw["value"]; exists {
		t.Error("expected 'value' field to be omitted when zero")
	}
	if _, exists := raw["threshold"]; exists {
		t.Error("expected 'threshold' field to be omitted when zero")
	}
}

func TestFleetAlert_JSONFromRawBytes(t *testing.T) {
	raw := `{
		"node_id": "edge-03",
		"severity": "warning",
		"sensor_type": "pressure",
		"message": "pressure rising",
		"value": 120.5,
		"threshold": 100.0,
		"timestamp": "2025-06-15T10:00:00Z"
	}`

	var alert FleetAlert
	if err := json.Unmarshal([]byte(raw), &alert); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if alert.NodeID != "edge-03" {
		t.Errorf("NodeID = %q, want %q", alert.NodeID, "edge-03")
	}
	if alert.Severity != "warning" {
		t.Errorf("Severity = %q, want %q", alert.Severity, "warning")
	}
	if alert.Value != 120.5 {
		t.Errorf("Value = %f, want %f", alert.Value, 120.5)
	}
	if alert.Threshold != 100.0 {
		t.Errorf("Threshold = %f, want %f", alert.Threshold, 100.0)
	}
}

func TestFullTopic_WithPrefix(t *testing.T) {
	c := &Client{topicPrefix: "edgeclaw/fleet"}
	got := c.fullTopic("alerts")
	want := "edgeclaw/fleet/alerts"
	if got != want {
		t.Errorf("fullTopic(%q) = %q, want %q", "alerts", got, want)
	}
}

func TestFullTopic_WithoutPrefix(t *testing.T) {
	c := &Client{topicPrefix: ""}
	got := c.fullTopic("alerts")
	want := "alerts"
	if got != want {
		t.Errorf("fullTopic(%q) = %q, want %q", "alerts", got, want)
	}
}

func TestFullTopic_NestedSuffix(t *testing.T) {
	c := &Client{topicPrefix: "fleet"}
	got := c.fullTopic("sensors/temperature")
	want := "fleet/sensors/temperature"
	if got != want {
		t.Errorf("fullTopic(%q) = %q, want %q", "sensors/temperature", got, want)
	}
}

func TestConfig_JSONTags(t *testing.T) {
	cfg := Config{
		BrokerURL:   "tcp://broker:1883",
		TopicPrefix: "edgeclaw",
		ClientID:    "node-01",
		Username:    "user",
		Password:    "pass",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	expected := map[string]string{
		"broker_url":   "tcp://broker:1883",
		"topic_prefix": "edgeclaw",
		"client_id":    "node-01",
		"username":     "user",
		"password":     "pass",
	}

	for key, wantVal := range expected {
		if got, ok := raw[key]; !ok {
			t.Errorf("missing JSON key %q", key)
		} else if got != wantVal {
			t.Errorf("JSON[%q] = %q, want %q", key, got, wantVal)
		}
	}
}

func TestConfig_OmitsEmptyOptionalFields(t *testing.T) {
	cfg := Config{
		BrokerURL:   "tcp://broker:1883",
		TopicPrefix: "edgeclaw",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	// client_id has omitempty
	if _, exists := raw["client_id"]; exists {
		t.Error("expected 'client_id' to be omitted when empty")
	}
}
