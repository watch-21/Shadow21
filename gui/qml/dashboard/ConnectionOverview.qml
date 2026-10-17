import QtQuick
import Shadow21.Desktop
import QtQuick.Layouts

Rectangle {
    id: root

    required property var backend
    required property var workspace
    readonly property bool proxyMode: !backend.anonsurfActive && (workspace.mode === "proxy" || workspace.modeStatus.mode === "proxy")
    readonly property bool inactive: !backend.anonsurfActive && workspace.modeStatus.phase === "inactive"
    readonly property var ipResult: !workspace.connectionActive ? ({}) : proxyMode ? workspace.proxyCheck : workspace.systemCheck
    readonly property string ipState: !workspace.connectionActive ? "idle" : proxyMode ? workspace.proxyIPState : workspace.systemIPState
    function stateText(state) {
        if (state === "checking") return AppLanguage.tr("Checking…");
        if (state === "ready") return AppLanguage.tr("Verified request");
        if (state === "failed") return AppLanguage.tr("Check failed");
        if (state === "stale") return AppLanguage.tr("Stale — check again");
        return AppLanguage.tr("Not checked");
    }
    // Four columns hold down to the width where each one would stop fitting a
    // readable value; below that two, never a single stacked column, which
    // would make the window grow much taller than it needs to be.
    readonly property int columns: width < 1000 ? 2 : 4

    implicitHeight: content.implicitHeight + Theme.spacingMedium * 2
    color: Theme.bgPanel
    border.width: 1
    border.color: Theme.borderLight

    GridLayout {
        id: content

        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: Theme.spacingMedium
        columns: root.columns
        columnSpacing: Theme.spacingLarge
        rowSpacing: Theme.spacingSmall

        StatusItem {
            Layout.fillWidth: true
            label: AppLanguage.tr("Tor service")
            tooltip: AppLanguage.tr("Tor service state. ACTIVE alone does not confirm that a request uses Tor.")
            value: AppLanguage.tr(!root.backend.statusAvailable ? "UNKNOWN" : root.backend.torActive ? "ACTIVE" : "INACTIVE")
            detail: AppLanguage.tr(!root.backend.statusAvailable ? "Unavailable" : root.backend.anonsurfActive ? "Managed by shadow21" : "System service")
            accentColor: root.backend.statusAvailable && root.backend.torActive ? Theme.neonCyan : Theme.textPrimary
        }

        StatusItem {
            Layout.fillWidth: true
            label: AppLanguage.tr("DNS setup")
            tooltip: AppLanguage.tr("DNS configuration inferred from the resolver and service state, not a live DNS routing check.")
            value: AppLanguage.tr(root.backend.dnsMode)
            detail: AppLanguage.tr(root.backend.dnsServers)
            accentColor: !root.backend.statusAvailable || !root.backend.anonsurfActive ? Theme.textPrimary : root.backend.dnsMode.indexOf("Tor") === 0 ? Theme.neonCyan : Theme.warning
        }

        StatusItem {
            Layout.fillWidth: true
            label: root.inactive ? AppLanguage.tr("Current external IP") : root.proxyMode ? AppLanguage.tr("Proxy IP") : AppLanguage.tr("Public IP")
            tooltip: AppLanguage.tr("IP observed by the Tor Project for this request. A Tor exit result does not verify every application's traffic.") + "\n" + (root.ipResult.ip || "")
            value: !root.workspace.connectionActive ? AppLanguage.tr("Start a mode to check IP") : root.ipResult.ip || root.stateText(root.ipState)
            detail: !root.workspace.connectionActive ? "" : (root.inactive ? AppLanguage.tr("Before connection") + " · " : root.proxyMode ? "SOCKS · " : AppLanguage.tr("System route") + " · ") + root.stateText(root.ipState) + (root.ipState === "ready" ? " · " + (root.ipResult.is_tor ? "Tor" : AppLanguage.tr("Direct")) : "")
            timestamp: root.ipResult.timestamp || root.ipResult.last_successful_timestamp || ""
            accentColor: root.ipState === "ready" ? (root.ipResult.is_tor ? Theme.neonCyan : Theme.warning) : Theme.textPrimary
        }

        Item {
            Layout.fillWidth: true
            Layout.alignment: Qt.AlignTop
            implicitHeight: startupContent.implicitHeight

            ColumnLayout {
                id: startupContent

                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                spacing: Theme.spacingTiny

                Text {
                    text: AppLanguage.tr("System startup")
                    color: Theme.textSecondary
                    font.family: Theme.uiFont
                    font.pixelSize: Theme.badgeSize
                    font.weight: Font.Medium
                }

                ToggleOption {
                    Layout.fillWidth: true
                    label: AppLanguage.tr(root.backend.bootEnabled ? "Enabled" : "Disabled")
                    tooltip: AppLanguage.tr("Starts shadow21 during boot. Traffic sent before routing is activated is not covered.")
                    checked: root.backend.bootEnabled
                    enabled: (root.workspace.mode === "system" || root.backend.anonsurfActive) && root.workspace.modeStatus.phase === "inactive" && !root.backend.busy
                    // The box follows the service, not the click: a refused
                    // authorisation leaves the real state on screen.
                    onToggled: {
                        if (checked !== root.backend.bootEnabled)
                            root.backend.setBootEnabled(checked);
                        checked = Qt.binding(() => root.backend.bootEnabled);
                    }
                }
            }
        }
    }
}
