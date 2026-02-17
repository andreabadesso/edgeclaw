package tools

import (
	"context"
	"fmt"

	"github.com/andreabadesso/edgeclaw/pkg/iot/mqtt"
)

// MQTTPublish allows PicoClaw to publish messages to the fleet MQTT broker.
type MQTTPublish struct {
	client *mqtt.Client
}

// NewMQTTPublish creates a new MQTT publish tool.
func NewMQTTPublish(client *mqtt.Client) *MQTTPublish {
	return &MQTTPublish{client: client}
}

func (t *MQTTPublish) Name() string { return "mqtt_publish" }

func (t *MQTTPublish) Description() string {
	return "Publish a message to the fleet MQTT broker (routed over Tailscale). " +
		"Accepts 'topic' (relative to the configured prefix) and 'payload' (string). " +
		"Use for sharing insights with other edge bots, broadcasting alerts, " +
		"or triggering actions on remote nodes."
}

func (t *MQTTPublish) Execute(ctx context.Context, args map[string]string) (string, error) {
	topic, ok := args["topic"]
	if !ok || topic == "" {
		return "", fmt.Errorf("missing required argument: topic")
	}

	payload, ok := args["payload"]
	if !ok {
		return "", fmt.Errorf("missing required argument: payload")
	}

	if err := t.client.Publish(topic, payload); err != nil {
		return "", fmt.Errorf("mqtt publish: %w", err)
	}

	return fmt.Sprintf("published to %s", topic), nil
}
