// Package mqtt provides an MQTT client for fleet-wide communication between
// EdgeClaw nodes. Messages are routed over a Tailscale VPN to a shared
// Mosquitto broker, enabling edge bots to share insights, broadcast alerts,
// and receive commands from other nodes in the fleet.
package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// Config holds MQTT broker connection parameters.
type Config struct {
	BrokerURL   string `json:"broker_url"`
	TopicPrefix string `json:"topic_prefix"`
	ClientID    string `json:"client_id,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
}

// MessageHandler is called when a message arrives on a subscribed topic.
type MessageHandler func(topic string, payload []byte)

// Client wraps the Paho MQTT client for fleet communication.
type Client struct {
	client      paho.Client
	topicPrefix string

	mu       sync.Mutex
	handlers map[string]MessageHandler
}

// New creates and connects an MQTT client to the fleet broker.
func New(cfg Config) (*Client, error) {
	opts := paho.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetKeepAlive(30 * time.Second).
		SetCleanSession(false)

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}

	c := &Client{
		topicPrefix: cfg.TopicPrefix,
		handlers:    make(map[string]MessageHandler),
	}

	opts.SetDefaultPublishHandler(func(_ paho.Client, msg paho.Message) {
		c.mu.Lock()
		handler, ok := c.handlers[msg.Topic()]
		c.mu.Unlock()
		if ok {
			handler(msg.Topic(), msg.Payload())
		}
	})

	opts.SetOnConnectHandler(func(_ paho.Client) {
		log.Printf("[mqtt] connected to %s", cfg.BrokerURL)
		c.mu.Lock()
		topics := make(map[string]byte, len(c.handlers))
		for t := range c.handlers {
			topics[t] = 1
		}
		c.mu.Unlock()
		if len(topics) > 0 {
			c.client.SubscribeMultiple(topics, nil)
		}
	})

	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		log.Printf("[mqtt] connection lost: %v", err)
	})

	c.client = paho.NewClient(opts)

	token := c.client.Connect()
	if token.WaitTimeout(10 * time.Second); token.Error() != nil {
		return nil, fmt.Errorf("mqtt connect: %w", token.Error())
	}

	return c, nil
}

func (c *Client) fullTopic(suffix string) string {
	if c.topicPrefix == "" {
		return suffix
	}
	return c.topicPrefix + "/" + suffix
}

// Publish sends a JSON-encoded message to a topic under the configured prefix.
func (c *Client) Publish(topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	token := c.client.Publish(c.fullTopic(topic), 1, false, data)
	if token.WaitTimeout(5 * time.Second); token.Error() != nil {
		return fmt.Errorf("mqtt publish: %w", token.Error())
	}
	return nil
}

// Subscribe registers a handler for messages on a topic under the configured prefix.
func (c *Client) Subscribe(topic string, handler MessageHandler) error {
	full := c.fullTopic(topic)
	c.mu.Lock()
	c.handlers[full] = handler
	c.mu.Unlock()

	token := c.client.Subscribe(full, 1, nil)
	if token.WaitTimeout(5 * time.Second); token.Error() != nil {
		return fmt.Errorf("mqtt subscribe: %w", token.Error())
	}

	log.Printf("[mqtt] subscribed to %s", full)
	return nil
}

// PublishAlert is a convenience method for broadcasting a fleet-wide alert.
func (c *Client) PublishAlert(nodeID string, alert FleetAlert) error {
	alert.NodeID = nodeID
	alert.Timestamp = time.Now().UTC()
	return c.Publish("alerts", alert)
}

// Close disconnects from the broker.
func (c *Client) Close() {
	c.client.Disconnect(1000)
}

// FleetAlert is the standardized alert message shared across the fleet via MQTT.
type FleetAlert struct {
	NodeID     string    `json:"node_id"`
	Severity   string    `json:"severity"`
	SensorType string    `json:"sensor_type"`
	Message    string    `json:"message"`
	Value      float64   `json:"value,omitempty"`
	Threshold  float64   `json:"threshold,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}
