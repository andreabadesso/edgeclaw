SUMMARY = "Tailscale VPN"
DESCRIPTION = "Tailscale mesh VPN for secure inter-node communication in the EdgeClaw fleet"
LICENSE = "BSD-3-Clause"
LIC_FILES_CHKSUM = "file://LICENSE;md5=placeholder"

SRC_URI = "https://pkgs.tailscale.com/stable/tailscale_${PV}_arm64.tgz"
SRC_URI[sha256sum] = "placeholder"

S = "${WORKDIR}/tailscale_${PV}_arm64"

do_install() {
    install -d ${D}${bindir}
    install -m 0755 ${S}/tailscale ${D}${bindir}/tailscale
    install -m 0755 ${S}/tailscaled ${D}${bindir}/tailscaled

    install -d ${D}${systemd_system_unitdir}
    install -m 0644 ${S}/systemd/tailscaled.service \
        ${D}${systemd_system_unitdir}/tailscaled.service

    install -d ${D}${sysconfdir}/default
    install -m 0644 ${S}/systemd/tailscaled.defaults \
        ${D}${sysconfdir}/default/tailscaled
}

inherit systemd
SYSTEMD_SERVICE:${PN} = "tailscaled.service"
SYSTEMD_AUTO_ENABLE = "enable"
