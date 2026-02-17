SUMMARY = "TimescaleDB for EdgeClaw"
DESCRIPTION = "TimescaleDB time-series database for IoT sensor data, deployed as a container"
LICENSE = "Apache-2.0"
LIC_FILES_CHKSUM = "file://LICENSE;md5=placeholder"

RDEPENDS:${PN} = "podman"

SRC_URI = "file://edgeclaw-timescaledb.service \
           file://schema.sql"

do_install() {
    install -d ${D}${datadir}/edgeclaw
    install -m 0644 ${WORKDIR}/schema.sql ${D}${datadir}/edgeclaw/schema.sql

    install -d ${D}${systemd_system_unitdir}
    install -m 0644 ${WORKDIR}/edgeclaw-timescaledb.service \
        ${D}${systemd_system_unitdir}/edgeclaw-timescaledb.service
}

inherit systemd
SYSTEMD_SERVICE:${PN} = "edgeclaw-timescaledb.service"
SYSTEMD_AUTO_ENABLE = "enable"
