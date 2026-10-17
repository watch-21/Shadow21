import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Shadow21.Desktop

Rectangle {
    id: root
    required property var workspace
    implicitHeight: content.implicitHeight + Theme.spacingMedium * 2
    color: Theme.bgPanel
    border.color: Theme.borderLight
    function size(value) {
        var n = Number(value || 0);
        if (n >= 1048576) return (n / 1048576).toFixed(2) + " MiB";
        if (n >= 1024) return (n / 1024).toFixed(1) + " KiB";
        return n.toFixed(0) + " B";
    }
    ColumnLayout {
        id: content
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: Theme.spacingMedium
        spacing: Theme.spacingSmall
        RowLayout {
            Layout.fillWidth: true
            Text {
                text: AppLanguage.tr("Network interface")
                color: Theme.textSecondary
                font.pixelSize: Theme.detailSize
            }
            ComboBox {
            palette.base: Theme.bgPanelAlt
            palette.button: Theme.bgPanelAlt
            palette.window: Theme.bgPanel
            palette.text: Theme.textPrimary
            palette.buttonText: Theme.textPrimary
            palette.windowText: Theme.textPrimary
            palette.highlight: Theme.accent
            palette.highlightedText: Theme.textOnAccent

                Layout.fillWidth: true
                model: root.workspace.interfaces
                currentIndex: root.workspace.interfaces.indexOf(root.workspace.networkInterface)
                Accessible.name: AppLanguage.tr("Network interface")
                onActivated: root.workspace.selectInterface(currentText)
            }
        }
        GridLayout {
            id: metrics
            Layout.fillWidth: true
            columns: root.width < 560 ? 2 : 3
            columnSpacing: Theme.spacingMedium
            rowSpacing: Theme.spacingSmall
            StatusItem {
                Layout.fillWidth: true
                Layout.minimumWidth: 0
                Layout.preferredWidth: (root.width - 2 * Theme.spacingMedium - (metrics.columns - 1) * Theme.spacingMedium) / metrics.columns
                label: AppLanguage.tr("Download / upload")
                value: root.workspace.traffic.state === "ready" ? root.size(root.workspace.traffic.download_bps) + "/s ↓" : AppLanguage.tr(root.workspace.traffic.state === "sampling" ? "Sampling…" : "Unavailable")
                detail: root.workspace.traffic.state === "ready" ? root.size(root.workspace.traffic.upload_bps) + "/s ↑" : ""
            }
            StatusItem {
                Layout.fillWidth: true
                Layout.minimumWidth: 0
                Layout.preferredWidth: (root.width - 2 * Theme.spacingMedium - (metrics.columns - 1) * Theme.spacingMedium) / metrics.columns
                label: AppLanguage.tr("Packets received / sent")
                value: root.workspace.traffic.received_packets || "—"
                detail: root.workspace.traffic.sent_packets || "—"
            }
            StatusItem {
                Layout.fillWidth: true
                Layout.minimumWidth: 0
                Layout.preferredWidth: (root.width - 2 * Theme.spacingMedium - (metrics.columns - 1) * Theme.spacingMedium) / metrics.columns
                label: AppLanguage.tr("Data received / sent")
                value: root.workspace.traffic.received_bytes !== undefined ? root.size(root.workspace.traffic.received_bytes) : "—"
                detail: root.workspace.traffic.sent_bytes !== undefined ? root.size(root.workspace.traffic.sent_bytes) : "—"
            }
        }
        ToggleOption {
                Layout.fillWidth: true
                label: AppLanguage.tr("Automatic IP checks")
                tooltip: AppLanguage.tr("Active-mode HTTPS IP checks: check.torproject.org, SOCKS every 1 s; system routing every 15 s. No checks while stopped and no direct proxy fallback.")
                checked: root.workspace.autoIP
                enabled: root.workspace.ready && !root.workspace.busy
                onToggled: {
                    root.workspace.setAutoIP(checked);
                    checked = Qt.binding(() => root.workspace.autoIP);
                }
        }
        Text {
            Layout.fillWidth: true
            text: AppLanguage.tr("Interface traffic, not a speed test. Updated every second.")
            color: Theme.textSecondary
            font.pixelSize: Theme.detailSize
            wrapMode: Text.WordWrap
        }
        Text {
            Layout.fillWidth: true
            text: AppLanguage.tr("IP checks: check.torproject.org · active modes only · SOCKS 1 s / system 15 s")
            color: Theme.textSecondary
            font.pixelSize: Theme.detailSize
            wrapMode: Text.WordWrap
        }
    }
}
