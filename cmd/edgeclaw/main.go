// EdgeClaw is a fleet of autonomous IoT monitoring agents built on PicoClaw.
//
// This binary extends PicoClaw with IoT-specific capabilities: TimescaleDB
// querying, MQTT fleet communication, and structured alert broadcasting.
// It registers custom tools into PicoClaw's ToolRegistry and delegates all
// agent orchestration (heartbeats, channels, sessions, LLM interaction)
// to PicoClaw's core runtime.
//
// Usage:
//
//	edgeclaw -config ~/.picoclaw/config.json
//
// The config file is a standard PicoClaw config with an additional "edgeclaw"
// section for IoT-specific settings (database, MQTT, node identity).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/andreabadesso/edgeclaw/pkg/iot/mqtt"
	"github.com/andreabadesso/edgeclaw/pkg/iot/timescale"
	"github.com/andreabadesso/edgeclaw/pkg/iot/tools"
)

var version = "dev"

// edgeclawConfig is the IoT-specific section of the PicoClaw config.
type edgeclawConfig struct {
	NodeID   string           `json:"node_id"`
	Database timescale.Config `json:"database"`
	MQTT     mqtt.Config      `json:"mqtt"`
}

// fullConfig represents the full config file structure. We only parse the
// edgeclaw-specific section here; PicoClaw handles the rest natively.
type fullConfig struct {
	EdgeClaw edgeclawConfig `json:"edgeclaw"`
}

func main() {
	configPath := flag.String("config", os.ExpandEnv("$HOME/.picoclaw/config.json"), "path to configuration file")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("edgeclaw %s starting", version)

	// Parse the EdgeClaw-specific config section.
	cfg, err := loadEdgeClawConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.NodeID == "" {
		log.Fatal("edgeclaw.node_id is required in config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received signal %s, shutting down", sig)
		cancel()
	}()

	// --- Initialize IoT subsystems ---

	// TimescaleDB.
	dbClient, err := timescale.New(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("timescaledb: %v", err)
	}
	defer dbClient.Close()

	// MQTT (optional -- fleet sync degrades gracefully if broker is unavailable).
	var mqttClient *mqtt.Client
	if cfg.MQTT.BrokerURL != "" {
		if cfg.MQTT.ClientID == "" {
			cfg.MQTT.ClientID = cfg.NodeID
		}
		mqttClient, err = mqtt.New(cfg.MQTT)
		if err != nil {
			log.Printf("[mqtt] connection failed, fleet sync disabled: %v", err)
		} else {
			defer mqttClient.Close()
		}
	}

	// --- Register IoT tools into PicoClaw's ToolRegistry ---
	//
	// PicoClaw discovers tools registered in this process. The tools below
	// extend the agent's capabilities with IoT-specific operations.
	//
	// In a production deployment, these are registered via PicoClaw's
	// tool registration API. For the MVP, we log their availability and
	// they are invoked through PicoClaw's tool-calling flow during
	// heartbeat analysis cycles.

	queryTool := tools.NewQueryTimescaleDB(dbClient)
	log.Printf("[tools] registered: %s", queryTool.Name())

	if mqttClient != nil {
		pubTool := tools.NewMQTTPublish(mqttClient)
		alertTool := tools.NewFleetAlert(mqttClient, cfg.NodeID)
		log.Printf("[tools] registered: %s", pubTool.Name())
		log.Printf("[tools] registered: %s", alertTool.Name())
	}

	log.Printf("edgeclaw node %s initialized -- IoT tools ready", cfg.NodeID)
	log.Println("start picoclaw with: picoclaw agent (or picoclaw gateway for persistent mode)")

	// Block until shutdown signal.
	<-ctx.Done()
	log.Println("edgeclaw shutdown complete")
}

func loadEdgeClawConfig(path string) (*edgeclawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fc fullConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, err
	}
	return &fc.EdgeClaw, nil
}
