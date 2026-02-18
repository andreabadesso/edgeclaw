package tools

import (
	"context"
	"strings"
	"testing"
)

func TestFleetAlert_Name(t *testing.T) {
	tool := NewFleetAlert(nil, "edge-01")
	if name := tool.Name(); name == "" {
		t.Error("Name() returned empty string")
	}
	if name := tool.Name(); name != "fleet_alert" {
		t.Errorf("Name() = %q, want %q", name, "fleet_alert")
	}
}

func TestFleetAlert_Description(t *testing.T) {
	tool := NewFleetAlert(nil, "edge-01")
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(desc, "alert") {
		t.Error("Description() should mention alert")
	}
}

func TestFleetAlert_Execute_MissingSensorType(t *testing.T) {
	tool := NewFleetAlert(nil, "edge-01")

	_, err := tool.Execute(context.Background(), map[string]string{
		"message": "something is wrong",
	})
	if err == nil {
		t.Error("expected error for missing sensor_type, got nil")
	}
	if !strings.Contains(err.Error(), "missing required argument: sensor_type") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "missing required argument: sensor_type")
	}
}

func TestFleetAlert_Execute_MissingMessage(t *testing.T) {
	tool := NewFleetAlert(nil, "edge-01")

	_, err := tool.Execute(context.Background(), map[string]string{
		"sensor_type": "temperature",
	})
	if err == nil {
		t.Error("expected error for missing message, got nil")
	}
	if !strings.Contains(err.Error(), "missing required argument: message") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "missing required argument: message")
	}
}

func TestFleetAlert_Execute_DefaultSeverity(t *testing.T) {
	// When severity is not provided, it defaults to "info". We can't easily
	// inspect the alert struct directly, but we can verify the validation
	// passes (it won't error on missing severity). The actual publish will
	// fail with a nil client, so we catch the panic.
	tool := NewFleetAlert(nil, "edge-01")

	func() {
		defer func() {
			recover() // nil mqtt client will panic
		}()
		_, err := tool.Execute(context.Background(), map[string]string{
			"sensor_type": "temperature",
			"message":     "test alert",
		})
		// If we get here without panic, the error should be about mqtt, not validation
		if err != nil && strings.Contains(err.Error(), "missing required argument") {
			t.Errorf("default severity should not cause a validation error: %v", err)
		}
	}()
}

func TestFleetAlert_Execute_ParsesValueAndThreshold(t *testing.T) {
	// Verify that value and threshold args are parsed without causing validation errors.
	// The actual publish will fail, so we just verify we get past validation.
	tool := NewFleetAlert(nil, "edge-01")

	func() {
		defer func() {
			recover() // nil mqtt client will panic
		}()
		_, err := tool.Execute(context.Background(), map[string]string{
			"sensor_type": "temperature",
			"message":     "threshold exceeded",
			"severity":    "critical",
			"value":       "98.6",
			"threshold":   "95.0",
		})
		if err != nil && strings.Contains(err.Error(), "missing required argument") {
			t.Errorf("unexpected validation error: %v", err)
		}
	}()
}

func TestFleetAlert_Execute_InvalidFloatIgnored(t *testing.T) {
	// Non-numeric values for value/threshold should be silently ignored
	// (ParseFloat returns an error, which the code ignores). Verify no panic
	// from the parsing itself.
	tool := NewFleetAlert(nil, "edge-01")

	func() {
		defer func() {
			recover() // nil mqtt client will panic
		}()
		_, err := tool.Execute(context.Background(), map[string]string{
			"sensor_type": "temperature",
			"message":     "test",
			"value":       "not-a-number",
			"threshold":   "also-not-a-number",
		})
		if err != nil && strings.Contains(err.Error(), "parse") {
			t.Errorf("invalid floats should be silently ignored, got: %v", err)
		}
	}()
}
