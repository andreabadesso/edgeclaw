package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/andreabadesso/edgeclaw/pkg/iot/mqtt"
)

// FleetAlert allows PicoClaw to broadcast a structured alert to all nodes in
// the fleet via MQTT. This is the primary mechanism for escalating anomalies
// detected during heartbeat analysis cycles.
type FleetAlert struct {
	client *mqtt.Client
	nodeID string
}

// NewFleetAlert creates a fleet alert tool bound to a specific node.
func NewFleetAlert(client *mqtt.Client, nodeID string) *FleetAlert {
	return &FleetAlert{client: client, nodeID: nodeID}
}

func (t *FleetAlert) Name() string { return "fleet_alert" }

func (t *FleetAlert) Description() string {
	return "Broadcast a structured alert to all nodes in the EdgeClaw fleet. " +
		"Arguments: 'severity' (info|warning|critical), 'sensor_type' (e.g. energy, water, pressure), " +
		"'message' (description of the issue), 'value' (optional, current reading), " +
		"'threshold' (optional, the exceeded threshold). " +
		"Use when anomalies are detected that other nodes or operators should know about."
}

func (t *FleetAlert) Execute(ctx context.Context, args map[string]string) (string, error) {
	severity := args["severity"]
	if severity == "" {
		severity = "info"
	}

	sensorType := args["sensor_type"]
	if sensorType == "" {
		return "", fmt.Errorf("missing required argument: sensor_type")
	}

	message := args["message"]
	if message == "" {
		return "", fmt.Errorf("missing required argument: message")
	}

	alert := mqtt.FleetAlert{
		Severity:   severity,
		SensorType: sensorType,
		Message:    message,
	}

	if v, ok := args["value"]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			alert.Value = f
		}
	}
	if v, ok := args["threshold"]; ok && v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			alert.Threshold = f
		}
	}

	if err := t.client.PublishAlert(t.nodeID, alert); err != nil {
		return "", fmt.Errorf("fleet alert: %w", err)
	}

	return fmt.Sprintf("fleet alert dispatched: [%s] %s", severity, message), nil
}
