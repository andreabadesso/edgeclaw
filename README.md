# EdgeClaw

**Autonomous IoT monitoring fleet powered by PicoClaw and local AI.**

EdgeClaw extends [PicoClaw](https://github.com/sipeed/picoclaw) with
IoT-specific tools to run a fleet of lightweight AI agents that monitor
sensor data, detect anomalies, and coordinate responses -- all locally,
on embedded hardware.

```
Sensors --> TimescaleDB --> PicoClaw + EdgeClaw Tools --> Brain Node (Qwen)
                                      |
                                      +--> Alerts (Telegram, MQTT Fleet)
```

---

## What This Is

A fleet of edge devices (Raspberry Pi 5, Sipeed boards) each running
PicoClaw as an autonomous monitoring agent. Each node:

- Queries its local **TimescaleDB** for time-series sensor data
- Sends analysis requests to a centralized **brain node** running Qwen
  via Ollama, accessed transparently over **Tailscale**
- Detects anomalies using statistical methods (IQR, Z-score, trend analysis)
- Broadcasts alerts to the fleet via **MQTT** and notifies operators
  via **Telegram**
- Falls back to a lightweight local model when the brain is unreachable

EdgeClaw does not reimplement an agent. It provides custom tools, workspace
templates, and deployment infrastructure that plug into PicoClaw's existing
runtime -- heartbeats, channels, sessions, tool calling, and subagents
all come from PicoClaw.

---

## Architecture

| Component        | Role                                                    | Technology                    |
|------------------|---------------------------------------------------------|-------------------------------|
| Edge Bot         | PicoClaw agent + EdgeClaw IoT tools                     | Go, Raspberry Pi 5 (16GB)    |
| Brain Node       | Centralized LLM inference                               | Ollama, Qwen3-30B-A3B        |
| Data Storage     | Time-series sensor data per edge node                   | TimescaleDB (PostgreSQL)      |
| Fleet Comms      | Inter-bot alerts, insights, commands                    | MQTT (Mosquitto) over VPN     |
| Networking       | Secure, zero-config mesh between all nodes              | Tailscale                     |
| OS / Deployment  | Minimal embedded images, OTA updates                    | Yocto Linux, Mender           |
| User Interface   | Bidirectional chat, alert notifications                 | PicoClaw channels (Telegram)  |

See [docs/architecture.md](docs/architecture.md) for the full design.

---

## Custom Tools

EdgeClaw registers these tools into PicoClaw's ToolRegistry:

| Tool                | What it does                                           |
|---------------------|--------------------------------------------------------|
| `query_timescaledb` | Run read-only SQL against the local sensor database    |
| `mqtt_publish`      | Publish messages to the fleet MQTT broker              |
| `fleet_alert`       | Broadcast structured alerts to all nodes and operators |

These are invoked by PicoClaw's agent loop during heartbeat analysis
cycles and interactive chat sessions. The agent decides when and how
to use them based on the heartbeat tasks and conversation context.

---

## Project Structure

```
edgeclaw/
  cmd/edgeclaw/main.go            Entry point: initializes IoT subsystems,
                                   registers tools, delegates to PicoClaw
  pkg/
    iot/
      timescale/client.go          TimescaleDB connection pool and query interface
      mqtt/client.go               MQTT fleet client (Paho, over Tailscale)
      tools/
        query_db.go                query_timescaledb tool
        mqtt_pub.go                mqtt_publish tool
        fleet_alert.go             fleet_alert tool
    providers/
      ollama/provider.go           Brain node provider with edge-local fallback

  workspace/                       PicoClaw workspace templates
    HEARTBEAT.md                   Periodic analysis task definitions
    IDENTITY.md                    Agent persona
    SOUL.md                        Behavioral philosophy
    AGENTS.md                      Operational rules and constraints
    skills/iot-monitor/SKILL.md    IoT monitoring domain knowledge

  configs/
    config.example.json            PicoClaw config with EdgeClaw extensions

  deploy/
    docker-compose.yml             Local dev stack (TimescaleDB, Mosquitto, Ollama)
    schema.sql                     TimescaleDB hypertable schema
    mosquitto.conf                 MQTT broker configuration
    yocto/                         Yocto recipes for production deployment
      meta-edgeclaw/               EdgeClaw binary and workspace
      meta-timescaledb/            TimescaleDB container management
      meta-tailscale/              Tailscale VPN daemon

  docs/
    architecture.md                Full architecture documentation
```

---

## Getting Started

### Prerequisites

- Go 1.22+
- Docker and Docker Compose (for local development)
- PicoClaw installed (`go install github.com/sipeed/picoclaw/cmd/picoclaw@latest`)

### 1. Start Infrastructure

```bash
make compose-up
```

This brings up TimescaleDB (with the sensor schema applied automatically),
Mosquitto, and Ollama.

### 2. Pull a Model

```bash
docker exec edgeclaw-ollama ollama pull qwen3:0.6b
```

For the full brain node experience, pull the larger model on beefier hardware:

```bash
ollama pull qwen3-30b-a3b-instruct
```

### 3. Install Workspace and Config

```bash
make workspace-init
make config-init
```

Edit `~/.picoclaw/config.json` with your provider credentials, database
password, and Telegram bot token.

### 4. Build and Run

```bash
make build
./bin/edgeclaw -config ~/.picoclaw/config.json
```

In a separate terminal, start PicoClaw in gateway mode for persistent
operation with heartbeats and channels:

```bash
picoclaw gateway
```

### 5. Seed Test Data (Optional)

Insert sample sensor readings to test the analysis pipeline:

```sql
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '5 min',  'energy',           450.0,  'meter-01'),
  (NOW() - INTERVAL '4 min',  'energy',           460.0,  'meter-01'),
  (NOW() - INTERVAL '3 min',  'energy',          1200.0,  'meter-01'),  -- spike
  (NOW() - INTERVAL '2 min',  'water_tank_level',  18.0,  'tank-01'),  -- critical
  (NOW() - INTERVAL '1 min',  'pressure',           2.1,  'valve-01');
```

---

## Configuration

The config file extends PicoClaw's standard format with an `edgeclaw` section:

```json
{
  "providers": {
    "qwen_brain": {
      "api_base": "http://qwen-brain.tailscale:11434/v1",
      "model": "qwen3-30b-a3b-instruct",
      "fallback_api_base": "http://localhost:11434/v1",
      "fallback_model": "qwen3-0.6b"
    }
  },
  "heartbeat": {
    "enabled": true,
    "interval": 5
  },
  "edgeclaw": {
    "node_id": "edge-bot-01",
    "database": {
      "host": "localhost",
      "port": 5432,
      "user": "edgeclaw",
      "password": "changeme",
      "dbname": "edgeclaw"
    },
    "mqtt": {
      "broker_url": "tcp://mosquitto.tailscale:1883",
      "topic_prefix": "edgeclaw/fleet"
    }
  }
}
```

See [configs/config.example.json](configs/config.example.json) for all options.

---

## Deployment

### Development

Use Docker Compose for infrastructure and run EdgeClaw + PicoClaw directly:

```bash
make compose-up    # TimescaleDB, Mosquitto, Ollama
make run           # EdgeClaw (IoT tools)
picoclaw gateway   # PicoClaw agent runtime
```

### Production (Yocto + Mender)

Build minimal Linux images with the provided Yocto meta-layers:

| Layer              | Contents                                   |
|--------------------|--------------------------------------------|
| meta-edgeclaw      | Binary, workspace templates, systemd unit  |
| meta-timescaledb   | Container-managed TimescaleDB + schema     |
| meta-tailscale     | Tailscale VPN daemon with auto-auth        |

The systemd service initializes EdgeClaw's IoT subsystems, then launches
PicoClaw in gateway mode. OTA updates are delivered via Mender over
Tailscale.

### Cross-Compilation

```bash
make build-arm64      # Raspberry Pi 5, other ARM64 boards
make build-riscv64    # Sipeed Lichee Pi 4A, other RISC-V boards
```

---

## Heartbeat Cycle

Every 5 minutes (configurable), PicoClaw executes the tasks defined in
`HEARTBEAT.md`:

1. Query recent sensor data via `query_timescaledb`
2. Analyze for outliers (IQR, Z-score) and trends
3. Correlate cross-sensor anomalies (e.g., energy spike + pressure drop)
4. Check thresholds (water tank < 20% = critical)
5. Broadcast detected anomalies via `fleet_alert`
6. Notify operators via Telegram

The iot-monitor skill provides the agent with domain-specific knowledge
about sensor types, analysis techniques, and alert thresholds.

---

## Hardware

| Component     | Specification                     | Estimated Cost |
|---------------|-----------------------------------|----------------|
| Edge Bot      | Raspberry Pi 5, 16GB RAM          | $50-100        |
| Brain Node    | Mini-PC, 32GB+ RAM, optional GPU  | $200-500       |
| Storage       | SSD or high-endurance SD card     | $20-50         |
| Networking    | WiFi/Ethernet + Tailscale overlay | --             |
| Fleet of 10   |                                   | $700-1500      |

Alternatives: Sipeed Lichee Pi 4A (RISC-V, cheaper), 4-8GB Pis for
lighter workloads with local fallback models only.

---

## License

MIT
