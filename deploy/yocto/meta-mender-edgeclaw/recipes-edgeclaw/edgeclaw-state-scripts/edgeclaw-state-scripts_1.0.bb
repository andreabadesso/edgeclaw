SUMMARY = "EdgeClaw Mender State Scripts"
DESCRIPTION = "Mender state scripts that manage edgeclaw and picoclaw services \
during OTA updates. Stops services before artifact installation and restarts \
them after a successful commit."
LICENSE = "MIT"
LIC_FILES_CHKSUM = "file://${COMMON_LICENSE_DIR}/MIT;md5=0835ade698e0bcf8506ecda2f7b4f302"

SRC_URI = " \
    file://ArtifactInstall_Enter_00 \
    file://ArtifactCommit_Enter_00 \
"

inherit mender-state-scripts

do_install() {
    install -d ${D}${MENDER_STATE_SCRIPTS_DIR}
    install -m 0755 ${WORKDIR}/ArtifactInstall_Enter_00 \
        ${D}${MENDER_STATE_SCRIPTS_DIR}/ArtifactInstall_Enter_00
    install -m 0755 ${WORKDIR}/ArtifactCommit_Enter_00 \
        ${D}${MENDER_STATE_SCRIPTS_DIR}/ArtifactCommit_Enter_00
}
