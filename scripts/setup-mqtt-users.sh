#!/usr/bin/env bash
# setup-mqtt-users.sh — Generate Mosquitto password file for EdgeClaw.
#
# Usage:
#   ./scripts/setup-mqtt-users.sh [node-id ...]
#
# Examples:
#   ./scripts/setup-mqtt-users.sh                  # admin only
#   ./scripts/setup-mqtt-users.sh rpi-01 rpi-02    # admin + two edge bots
#
# The password file is written to deploy/mosquitto_passwd and is gitignored.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PASSWD_FILE="$REPO_ROOT/deploy/mosquitto_passwd"

# Ensure the deploy directory exists
mkdir -p "$REPO_ROOT/deploy"

# ---- Admin user ----
echo "Creating admin user..."
read -rsp "Enter password for admin: " ADMIN_PASS
echo

# Create the password file (overwrite if it exists) with the admin user
mosquitto_passwd -c -b "$PASSWD_FILE" admin "$ADMIN_PASS"
echo "  -> admin user created."

# ---- Edge-bot users ----
for NODE_ID in "$@"; do
    USERNAME="edge-bot-${NODE_ID}"
    echo "Creating user: $USERNAME ..."
    read -rsp "Enter password for $USERNAME: " BOT_PASS
    echo
    # Append (-b without -c) to the existing file
    mosquitto_passwd -b "$PASSWD_FILE" "$USERNAME" "$BOT_PASS"
    echo "  -> $USERNAME created."
done

echo ""
echo "Password file written to: $PASSWD_FILE"
echo "Mount it into the Mosquitto container at /mosquitto/config/passwd"
