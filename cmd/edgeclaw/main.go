// EdgeClaw is a fleet of autonomous IoT monitoring agents built on PicoClaw.
//
// This binary extends PicoClaw with IoT-specific capabilities: TimescaleDB
// querying, MQTT fleet communication, and structured alert broadcasting.
// It registers custom tools into a local registry and exposes them over an
// HTTP API that PicoClaw consumes as external tools. All agent orchestration
// (heartbeats, channels, sessions, LLM interaction) is delegated to
// PicoClaw's core runtime.
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
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andreabadesso/edgeclaw/pkg/iot/mqtt"
	"github.com/andreabadesso/edgeclaw/pkg/iot/timescale"
	"github.com/andreabadesso/edgeclaw/pkg/iot/tools"
	"github.com/andreabadesso/edgeclaw/pkg/registry"
)

var version = "dev"

// edgeclawConfig is the IoT-specific section of the PicoClaw config.
type edgeclawConfig struct {
	NodeID   string           `json:"node_id"`
	Database timescale.Config `json:"database"`
	MQTT     mqtt.Config      `json:"mqtt"`
	ToolPort int              `json:"tool_port,omitempty"`
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

	// Default tool server port.
	if cfg.ToolPort == 0 {
		cfg.ToolPort = 7331
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

	// --- Register IoT tools into the local registry ---

	reg := registry.New()

	reg.Register(tools.NewQueryTimescaleDB(dbClient))

	if mqttClient != nil {
		reg.Register(tools.NewMQTTPublish(mqttClient))
		reg.Register(tools.NewFleetAlert(mqttClient, cfg.NodeID))
	}

	// --- Start the tool HTTP server for PicoClaw integration ---

	addr := fmt.Sprintf(":%d", cfg.ToolPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: registry.HTTPHandler(reg),
	}

	// Start serving in a separate goroutine.
	go func() {
		log.Printf("[toolserver] listening on http://127.0.0.1%s", addr)
		log.Printf("[toolserver] PicoClaw external tool URL: http://127.0.0.1%s/tools", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[toolserver] listen error: %v", err)
		}
	}()

	log.Printf("edgeclaw node %s initialized -- %d tool(s) registered", cfg.NodeID, len(reg.All()))
	log.Printf("configure PicoClaw to use external tools at http://127.0.0.1%s", addr)

	// Block until shutdown signal.
	<-ctx.Done()

	// Gracefully shut down the HTTP server.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[toolserver] shutdown error: %v", err)
	}

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
