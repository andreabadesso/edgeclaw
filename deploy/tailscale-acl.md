# Tailscale ACL Policy for EdgeClaw

## Overview

The file `tailscale-acl.json` defines a least-privilege network policy for the
EdgeClaw IoT fleet.  It restricts which nodes can talk to which services and
on which ports, using Tailscale's tag-based ACL system.

## Node Types and Tags

| Tag               | Node Type     | Description                                        |
|-------------------|---------------|----------------------------------------------------|
| `tag:edge-bot`    | Edge Bot      | Raspberry Pi 5 / Sipeed board running PicoClaw     |
| `tag:brain`       | Brain Node    | Mini-PC running Ollama with Qwen models            |
| `tag:mqtt-broker` | MQTT Broker   | Mosquitto broker for fleet communication           |
| `tag:mender`      | Mender Server | OTA update server for fleet image deployments      |

Admin users (`group:admin`) own all tags and have unrestricted access.

## Traffic Matrix

| Source           | Destination      | Ports  | Purpose                          |
|------------------|------------------|--------|----------------------------------|
| `group:admin`    | `*`              | `*`    | Full management access           |
| `tag:edge-bot`   | `tag:brain`      | 11434  | Ollama LLM inference API         |
| `tag:edge-bot`   | `tag:mqtt-broker`| 1883   | Fleet MQTT pub/sub               |
| `tag:edge-bot`   | `tag:mender`     | 443    | OTA update downloads (HTTPS)     |
| `tag:edge-bot`   | `tag:edge-bot`   | 1883   | Peer-to-peer MQTT (mesh)         |
| `tag:brain`      | `tag:edge-bot`   | --     | **Denied** (brain cannot initiate) |

All other traffic is denied by default.

## SSH Access

Only `group:admin` devices can SSH (port 22) into fleet nodes.  Both root and
non-root users are permitted targets to support initial provisioning and
day-to-day debugging.

## DNS

MagicDNS is enabled so that every Tailscale node gets a stable hostname.
EdgeClaw configs reference services by name rather than IP:

- `qwen-brain.tailscale` -- Brain node Ollama API
- `mosquitto.tailscale` -- MQTT broker

The Tailscale DNS resolver (`100.100.100.100`) is configured as the nameserver.

## Auto Approvers

Edge bots may advertise subnet routes for local sensor networks behind them
(e.g., `10.x.x.x` or `192.168.x.x` ranges).  These routes are auto-approved
so newly provisioned bots can join the fleet without manual admin intervention.

## Applying the Policy

### Via the Tailscale Admin Console

1. Open <https://login.tailscale.com/admin/acls>.
2. Replace the contents with `deploy/tailscale-acl.json`.
3. Click **Save**.

### Via the Tailscale CLI (requires API key)

```bash
# Validate syntax first.
tailscale acl test --file deploy/tailscale-acl.json

# Push the policy.
tailscale acl set --file deploy/tailscale-acl.json
```

## Enrolling Nodes

Each node type is enrolled with a pre-authenticated key and the correct tag.
Generate auth keys in the Tailscale admin console under **Settings > Keys**
with the appropriate tag pre-approved.

### Edge Bot

```bash
tailscale up \
  --authkey=tskey-auth-XXXX \
  --advertise-tags=tag:edge-bot \
  --hostname=edge-bot-$(hostname -s)
```

### Brain Node

```bash
tailscale up \
  --authkey=tskey-auth-XXXX \
  --advertise-tags=tag:brain \
  --hostname=qwen-brain
```

### MQTT Broker

```bash
tailscale up \
  --authkey=tskey-auth-XXXX \
  --advertise-tags=tag:mqtt-broker \
  --hostname=mosquitto
```

### Mender Server

```bash
tailscale up \
  --authkey=tskey-auth-XXXX \
  --advertise-tags=tag:mender \
  --hostname=mender
```

## Yocto Integration

The `meta-tailscale` Yocto layer in `deploy/yocto/meta-tailscale/` includes a
systemd service that runs `tailscale up` at boot with a pre-provisioned auth
key baked into the image (stored in `/etc/tailscale/authkey`, readable only by
root).  The tag is set based on the machine class defined at build time.
