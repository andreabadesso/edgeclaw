# Mender client configuration for EdgeClaw IoT fleet
# Deploys OTA updates over Tailscale VPN to Raspberry Pi 5 and Sipeed boards

FILESEXTRAPATHS:prepend := "${THISDIR}/files:"

SRC_URI += "file://mender.conf"

# Mender server accessible over Tailscale VPN
MENDER_SERVER_URL = "https://mender.tailscale:443"

# Device type for Raspberry Pi 5 targets
MENDER_DEVICE_TYPE = "raspberrypi5"

# Tenant token — replace with actual token from Mender server
MENDER_TENANT_TOKEN = "REPLACE_WITH_YOUR_TENANT_TOKEN"

# Poll intervals (seconds)
# Check for updates every 30 minutes
MENDER_UPDATE_POLL_INTERVAL_SECONDS = "1800"
# Report inventory every 8 hours
MENDER_INVENTORY_POLL_INTERVAL_SECONDS = "28800"
# Retry on failure every 5 minutes
MENDER_RETRY_POLL_INTERVAL_SECONDS = "300"

do_install:append() {
    install -d ${D}${sysconfdir}/mender
    install -m 0600 ${WORKDIR}/mender.conf ${D}${sysconfdir}/mender/mender.conf
}
