SUMMARY = "EdgeClaw IoT Monitoring Agent"
DESCRIPTION = "IoT extensions for PicoClaw: TimescaleDB querying, MQTT fleet communication, and alert broadcasting"
LICENSE = "MIT"
LIC_FILES_CHKSUM = "file://LICENSE;md5=placeholder"

SRC_URI = "git://github.com/andreabadesso/edgeclaw.git;branch=main;protocol=https"
SRCREV = "${AUTOREV}"

S = "${WORKDIR}/git"

DEPENDS = "go-native"
RDEPENDS:${PN} = "picoclaw"

inherit go-mod systemd

GO_IMPORT = "github.com/andreabadesso/edgeclaw"

do_compile() {
    cd ${S}
    oe_runmake build-arm64
}

do_install() {
    # Binary.
    install -d ${D}${bindir}
    install -m 0755 ${S}/bin/edgeclaw-arm64 ${D}${bindir}/edgeclaw

    # PicoClaw workspace templates.
    install -d ${D}${datadir}/edgeclaw/workspace/skills/iot-monitor
    install -m 0644 ${S}/workspace/HEARTBEAT.md ${D}${datadir}/edgeclaw/workspace/
    install -m 0644 ${S}/workspace/IDENTITY.md ${D}${datadir}/edgeclaw/workspace/
    install -m 0644 ${S}/workspace/SOUL.md ${D}${datadir}/edgeclaw/workspace/
    install -m 0644 ${S}/workspace/AGENTS.md ${D}${datadir}/edgeclaw/workspace/
    install -m 0644 ${S}/workspace/skills/iot-monitor/SKILL.md \
        ${D}${datadir}/edgeclaw/workspace/skills/iot-monitor/

    # Configuration.
    install -d ${D}${sysconfdir}/picoclaw
    install -m 0640 ${S}/configs/config.example.json ${D}${sysconfdir}/picoclaw/config.json

    # Systemd service.
    install -d ${D}${systemd_system_unitdir}
    install -m 0644 ${WORKDIR}/edgeclaw.service ${D}${systemd_system_unitdir}/edgeclaw.service
}

SRC_URI += "file://edgeclaw.service"

SYSTEMD_SERVICE:${PN} = "edgeclaw.service"
SYSTEMD_AUTO_ENABLE = "enable"
