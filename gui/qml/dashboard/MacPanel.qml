import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Shadow21.Desktop
Rectangle {
    id: root
    required property var workspace
    color: Theme.bgPanel
    border.color: Theme.borderLight
    implicitHeight: content.implicitHeight + 2 * Theme.spacingMedium
    function openConfirmation(restore) { macDialog.restore = !!restore; macDialog.open(); }
    ColumnLayout {
        id: content
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: Theme.spacingMedium
        spacing: Theme.spacingSmall
        Text { Layout.fillWidth: true; text: AppLanguage.tr("MAC address"); color: Theme.textPrimary; font.bold: true }
        ComboBox {
            palette.base: Theme.bgPanelAlt
            palette.button: Theme.bgPanelAlt
            palette.window: Theme.bgPanel
            palette.text: Theme.textPrimary
            palette.buttonText: Theme.textPrimary
            palette.windowText: Theme.textPrimary
            palette.highlight: Theme.accent
            palette.highlightedText: Theme.textOnAccent
 Layout.fillWidth: true; model: root.workspace.interfaces; currentIndex: root.workspace.interfaces.indexOf(root.workspace.networkInterface); Accessible.name: AppLanguage.tr("Network interface"); enabled: !root.workspace.busy; onActivated: root.workspace.selectInterface(currentText) }
        Text {
            Layout.fillWidth: true
            text: AppLanguage.tr("Current MAC") + ": " + (root.workspace.currentMAC || AppLanguage.tr("Unavailable"))
            color: Theme.textPrimary
            wrapMode: Text.WrapAnywhere
        }
        GridLayout {
            Layout.fillWidth: true
            columns: root.width < 600 ? 1 : 2
            rowSpacing: Theme.spacingSmall
            columnSpacing: Theme.spacingMedium
            ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("Randomize MAC"); iconName: "refresh"; enabled: !root.workspace.busy && !root.workspace.systemRoutingActive && root.workspace.modeStatus.phase === "inactive" && !root.workspace.terminalRunning && !!root.workspace.currentMAC; onClicked: { macDialog.restore = false; macDialog.open(); } }
            ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("Restore previous MAC"); iconName: "restart"; enabled: !root.workspace.busy && !root.workspace.systemRoutingActive && root.workspace.modeStatus.phase === "inactive" && !root.workspace.terminalRunning && !!root.workspace.currentMAC; onClicked: { macDialog.restore = true; macDialog.open(); } }
        }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("Stop both modes first. Changing the MAC briefly reconnects the selected NetworkManager connection. This affects the local network address, not the public IP by itself."); color: Theme.textSecondary; wrapMode: Text.WordWrap }
        Text { Layout.fillWidth: true; text: root.workspace.macMessage; color: Theme.textSecondary; wrapMode: Text.WordWrap }
    }
    ConfirmDialog {
        id: macDialog
        property bool restore: false
        badge: "MAC"
        confirmText: AppLanguage.tr(restore ? "Restore previous MAC" : "Randomize MAC")
        message: AppLanguage.tr("Reconnect the selected network interface to change its MAC address? Connectivity can be interrupted. Only supported physical Ethernet/Wi-Fi connections managed by NetworkManager are accepted.")
        onAccepted: root.workspace.changeMAC(restore)
    }
}
