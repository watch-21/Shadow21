import QtQuick
import Shadow21.Desktop
import QtQuick.Layouts

Item {
    id: root

    required property var backend
    required property var workspace
    readonly property bool inactive: !backend.anonsurfActive && workspace.modeStatus.phase === "inactive"
    readonly property bool proxyMode: !inactive && !backend.anonsurfActive && (workspace.mode === "proxy" || workspace.modeStatus.mode === "proxy")
    property bool safeKill: false
    readonly property bool iconOnly: false

    implicitHeight: content.implicitHeight

    ColumnLayout {
        id: content

        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        spacing: Theme.spacingSmall

        GridLayout {
            Layout.fillWidth: true
            columns: root.width < 650 ? 2 : 4
            columnSpacing: Theme.spacingSmall
            rowSpacing: Theme.spacingSmall

            ActionButton {
                Layout.fillWidth: true
                text: AppLanguage.tr("New Identity")
                iconName: "identity"
                iconOnly: root.iconOnly
                tooltip: AppLanguage.tr("Requests new Tor circuits for new connections. The exit IP may stay the same; existing connections are unchanged.")
                enabled: root.backend.statusAvailable && root.backend.anonsurfActive && root.backend.torActive && !root.backend.busy
                onClicked: root.backend.changeIdentity()
            }

            ActionButton {
                Layout.fillWidth: true
                text: AppLanguage.tr("Verify IP")
                iconName: "globe"
                iconOnly: root.iconOnly
                tooltip: AppLanguage.tr("Checks the public IP used by this request and whether it belongs to a known Tor exit node.")
                enabled: root.backend.anonsurfActive && root.workspace.connectionActive && !root.backend.busy
                onClicked: root.backend.checkIP()
            }

            ActionButton {
                Layout.fillWidth: true
                text: AppLanguage.tr("Restart")
                iconName: "restart"
                iconOnly: root.iconOnly
                tooltip: AppLanguage.tr("Restarts Tor and rebuilds routing from the current shadow21 configuration.")
                enabled: root.backend.statusAvailable && root.backend.anonsurfActive && !root.backend.busy
                onClicked: root.backend.restart(root.safeKill)
            }

            ActionButton {
                Layout.fillWidth: true
                text: AppLanguage.tr("Refresh")
                iconName: "refresh"
                iconOnly: root.iconOnly
                tooltip: AppLanguage.tr("Reads the shadow21 status right away instead of waiting for the periodic update.")
                enabled: !root.backend.busy
                onClicked: root.backend.refresh()
            }
        }
    }

}
