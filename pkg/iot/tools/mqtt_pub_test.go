package tools

import (
	"context"
	"strings"
	"testing"
)

func TestMQTTPublish_Name(t *testing.T) {
	tool := NewMQTTPublish(nil)
	if name := tool.Name(); name == "" {
		t.Error("Name() returned empty string")
	}
	if name := tool.Name(); name != "mqtt_publish" {
		t.Errorf("Name() = %q, want %q", name, "mqtt_publish")
	}
}

func TestMQTTPublish_Description(t *testing.T) {
	tool := NewMQTTPublish(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(desc, "MQTT") {
		t.Error("Description() should mention MQTT")
	}
}

func TestMQTTPublish_Execute_MissingTopic(t *testing.T) {
	tool := NewMQTTPublish(nil)

	// No topic key
	_, err := tool.Execute(context.Background(), map[string]string{"payload": "hello"})
	if err == nil {
		t.Error("expected error for missing topic, got nil")
	}
	if !strings.Contains(err.Error(), "missing required argument: topic") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "missing required argument: topic")
	}

	// Empty topic
	_, err = tool.Execute(context.Background(), map[string]string{"topic": "", "payload": "hello"})
	if err == nil {
		t.Error("expected error for empty topic, got nil")
	}
}

func TestMQTTPublish_Execute_MissingPayload(t *testing.T) {
	tool := NewMQTTPublish(nil)

	// No payload key
	_, err := tool.Execute(context.Background(), map[string]string{"topic": "test/topic"})
	if err == nil {
		t.Error("expected error for missing payload, got nil")
	}
	if !strings.Contains(err.Error(), "missing required argument: payload") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "missing required argument: payload")
	}
}
