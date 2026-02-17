# EdgeClaw Architecture

## Overview

EdgeClaw is an IoT monitoring system built on top of
[PicoClaw](https://github.com/sipeed/picoclaw), Sipeed's ultra-lightweight
AI assistant. Rather than implementing a custom agent, EdgeClaw extends
PicoClaw with IoT-specific tools, workspace templates, and deployment
infrastructure for running autonomous sensor monitoring fleets on embedded
hardware.

PicoClaw provides:
- Agent loop with multi-turn tool calling
- Heartbeat scheduler for periodic analysis
- Bidirectional chat channels (Telegram, Discord, etc.)
- Session persistence and memory
- Subagent spawning for long-running tasks
- Cron scheduling
- Provider abstraction for multiple LLM backends

EdgeClaw adds:
- TimescaleDB query tool for time-series sensor data
- MQTT client for fleet-wide communication over Tailscale
- Fleet alert broadcasting tool
- Brain node provider with automatic edge-local fallback
- IoT monitoring skill with anomaly detection techniques
- Deployment infrastructure (Docker Compose, Yocto recipes, systemd)


## System Components

```
[External Sensors / Ingestion]
         |
         v
[TimescaleDB on Each Edge Bot]
         |
         v
[PicoClaw Agent + EdgeClaw Tools]  <----->  [Tailscale VPN]  <----->  [Brain Node: Ollama + Qwen]
         |                                         |
         v                                         v
[Analysis: Outliers, Trends]              [Other Edge Bots via MQTT]
         |
         v
[Alerts: Telegram / Exec / MQTT Fleet]
         |
         v
[OTA Updates via Mender over Tailscale]
```

### Edge Bots

Each edge bot is a Raspberry Pi 5 (16GB) running:

1. **PicoClaw** -- The core agent runtime, handling the heartbeat loop,
   tool calling, LLM interaction, and chat channels.
2. **EdgeClaw binary** -- Initializes IoT subsystems (TimescaleDB client,
   MQTT client) and registers custom tools into PicoClaw's ToolRegistry.
3. **TimescaleDB** -- Local PostgreSQL with the TimescaleDB extension,
   storing sensor readings in hypertables with automatic compression
   and retention policies.
4. **Tailscale** -- VPN overlay for secure, zero-config access to the
   brain node and MQTT broker.

### Brain Node

A more powerful device (mini-PC, 32GB+ RAM, optional GPU) running:

- **Ollama** serving Qwen3-30B-A3B-Instruct (4-bit quantized)
- Accessible to all edge bots via Tailscale DNS
  (e.g., `http://qwen-brain.tailscale:11434/v1`)

### Fleet Coordination

- **MQTT Broker** (Mosquitto) runs on a stable node, accessible over Tailscale.
- Edge bots publish insights and alerts to shared topics.
- Topic structure: `edgeclaw/fleet/alerts`, `edgeclaw/fleet/insights`,
  `edgeclaw/fleet/commands/{node_id}`.

### Management Layer

- **Yocto** for building minimal, reproducible OS images.
- **Mender** for OTA updates over Tailscale (phased rollouts by device group).


## Data Flow

1. External processes (MQTT gateways, Telegraf, sensor scripts) ingest
   readings into the local TimescaleDB `sensors` hypertable.
2. PicoClaw's heartbeat fires every N minutes (default: 5).
3. The heartbeat reads `HEARTBEAT.md` for task definitions.
4. PicoClaw invokes the `query_timescaledb` tool to fetch recent data.
5. Data is sent to the brain node (Qwen via Ollama) for analysis,
   using the iot-monitor skill for domain-specific guidance.
6. If the brain is unreachable, the provider falls back to a lighter
   local model for degraded-mode analysis.
7. The brain returns structured analysis (outliers, trends, correlations).
8. If anomalies are detected, PicoClaw invokes the `fleet_alert` tool
   to broadcast via MQTT and notify operators via Telegram.
9. Other edge bots receive fleet alerts and can incorporate them into
   their own analysis context.


## Custom Tools

| Tool               | Description                                                |
|--------------------|------------------------------------------------------------|
| query_timescaledb  | Execute read-only SQL queries against the local sensor DB  |
| mqtt_publish       | Publish arbitrary messages to fleet MQTT topics            |
| fleet_alert        | Broadcast structured alerts with severity and sensor info  |

These tools are registered at startup by the EdgeClaw binary and become
available to PicoClaw's agent loop for use during heartbeat analysis
and interactive chat sessions.


## Workspace Structure

EdgeClaw provides PicoClaw workspace templates that define the agent's
behavior in an IoT monitoring context:

| File                              | Purpose                                    |
|-----------------------------------|--------------------------------------------|
| `HEARTBEAT.md`                    | Periodic analysis tasks (what to check)    |
| `IDENTITY.md`                     | Agent self-concept and role                |
| `SOUL.md`                         | Core behavioral philosophy                 |
| `AGENTS.md`                       | Operational rules and constraints          |
| `skills/iot-monitor/SKILL.md`     | Domain knowledge for sensor analysis       |


## Provider Architecture

```
[PicoClaw Agent]
       |
       v
[EdgeClaw Ollama Provider]
       |
       +---> [Brain Node: Qwen-30B via Tailscale]  (primary)
       |
       +---> [Local Ollama: Qwen-0.6B]             (fallback)
```

The Ollama provider in `pkg/providers/ollama/` implements automatic fallback:
if the brain node request fails (timeout, network error), the provider
retries with the configured local model. This ensures the agent remains
operational during network partitions, at the cost of reduced analysis
quality.


## Deployment

### Development

```bash
# Start infrastructure services.
make compose-up

# Pull the Qwen model into Ollama.
docker exec edgeclaw-ollama ollama pull qwen3:0.6b

# Install PicoClaw workspace templates.
make workspace-init

# Install PicoClaw config.
make config-init

# Build and run EdgeClaw.
make run
```

### Production (Yocto)

Yocto recipes are provided in `deploy/yocto/` for three meta-layers:

- **meta-edgeclaw** -- The EdgeClaw binary, workspace templates, and systemd service.
- **meta-timescaledb** -- TimescaleDB container management and schema.
- **meta-tailscale** -- Tailscale VPN daemon and auto-authentication.

The systemd service starts EdgeClaw (which initializes IoT subsystems),
then launches PicoClaw in gateway mode for persistent operation with
heartbeats and channels.


## Security

- **Workspace sandboxing**: PicoClaw's `restrict_to_workspace: true` limits
  file operations.
- **SQL safety**: The `query_timescaledb` tool only allows SELECT queries.
- **Exec whitelist**: The PicoClaw exec tool can be restricted to specific
  commands via configuration.
- **Network**: Tailscale provides end-to-end encryption and ACLs
  (e.g., edge bots can only access the brain node on port 11434).
- **Database**: TimescaleDB credentials are per-node; role-based access
  limits the agent to read operations.
- **MQTT**: Topic-level ACLs in Mosquitto restrict bots to their prefix.
