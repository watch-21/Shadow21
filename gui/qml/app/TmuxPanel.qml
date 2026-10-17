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
    function openInspectionDialog(kind) {
        if (kind === "start") startDialog.open();
        
        else if (kind === "stop") stopDialog.open();
    }
    ColumnLayout {
        id: content
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: Theme.spacingMedium
        spacing: Theme.spacingMedium
        Text { Layout.fillWidth: true; text: AppLanguage.tr("Optional tmux sessions"); color: Theme.textPrimary; font.pixelSize: Theme.headingSize; font.bold: true; wrapMode: Text.WordWrap }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("Root session") + " · " + AppLanguage.tr(root.workspace.modeStatus.phase === "active" ? "Active" : root.workspace.modeStatus.phase === "inactive" ? "Inactive" : root.workspace.modeStatus.phase === "starting" ? "Starting…" : root.workspace.modeStatus.phase === "stopping" ? "Stopping…" : root.workspace.modeStatus.phase === "failed" ? "Failed — recovery required" : "Unknown"); color: Theme.textSecondary; wrapMode: Text.WordWrap }
        ToggleOption {
            label: AppLanguage.tr("Enable tmux")
            checked: root.workspace.tmuxEnabled
            enabled: !root.workspace.busy && root.workspace.modeStatus.phase === "inactive"
            onToggled: { root.workspace.save(root.workspace.language, root.workspace.mode, root.workspace.endpoint, checked); checked = Qt.binding(() => root.workspace.tmuxEnabled); }
        }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("Proxy IP") + " · SOCKS " + root.workspace.endpoint; color: Theme.textSecondary }
        Text { Layout.fillWidth: true; text: root.workspace.connectionActive && root.workspace.proxyIPState === "ready" ? AppLanguage.tr(root.workspace.proxyCheck.comparison === "first" ? "First successful check — no previous comparison" : root.workspace.proxyCheck.comparison === "changed" ? "IP changed since the previous successful check" : "IP unchanged since the previous successful check") : AppLanguage.tr("This check describes one request only."); color: Theme.textSecondary; wrapMode: Text.WordWrap }
        ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("Check proxy IP"); iconName: "globe"; enabled: root.workspace.connectionActive && !root.workspace.busy && root.workspace.modeStatus.phase === "active"; onClicked: root.workspace.testProxy() }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("A separate terminal opens with its default title. Commands run as root after authorization. No additional shell commands are inserted."); color: Theme.textSecondary; wrapMode: Text.WordWrap }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("System terminal") + ": " + (root.workspace.terminalProgram || AppLanguage.tr("Unavailable")); color: Theme.textSecondary; wrapMode: Text.WrapAnywhere }
        Text { Layout.fillWidth: true; visible: root.workspace.terminalRunning && root.workspace.modeStatus.phase !== "active"; text: AppLanguage.tr("Terminal open — authorization or Tor preparation may still be pending."); color: Theme.warning; wrapMode: Text.WordWrap }
        GridLayout {
            Layout.fillWidth: true
            columns: root.width < 650 ? 1 : 2
            columnSpacing: Theme.spacingMedium
            rowSpacing: Theme.spacingSmall
            ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("Start session"); iconName: "start"; primary: true; enabled: root.workspace.sessionStartAvailable; onClicked: startDialog.open() }
            ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("Attach session"); iconName: "monitor"; enabled: root.workspace.sessions.length > 0 && !root.workspace.busy && !root.workspace.terminalRunning; onClicked: root.workspace.sessionAction("open", root.workspace.sessions[0]) }
            ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("Detach session"); iconName: "stop"; enabled: root.workspace.sessions.length > 0 && !root.workspace.busy; onClicked: root.workspace.sessionAction("detach", root.workspace.sessions[0]) }
            ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("Stop session"); iconName: "close"; enabled: root.workspace.sessions.length > 0 && !root.workspace.busy; onClicked: stopDialog.open() }
            ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("List sessions"); iconName: "refresh"; enabled: !root.workspace.busy; onClicked: root.workspace.refreshSessions() }
            ActionButton { Layout.fillWidth: true; text: AppLanguage.tr("Recover proxy mode"); iconName: "restart"; enabled: !root.workspace.busy && root.workspace.modeStatus.phase !== "inactive"; onClicked: recoverDialog.open() }
        }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("Commands — copy only"); color: Theme.textPrimary; font.bold: true }
        Repeater {
            model: 3
            delegate: RowLayout {
                id: commandRow
                required property int index
                property bool copied: false
                property bool copyFailed: false
                Layout.fillWidth: true
                spacing: Theme.spacingSmall
                TextArea {
                    Layout.fillWidth: true
                    text: root.workspace.presetCommand(commandRow.index)
                    readOnly: true
                    selectByMouse: true
                    wrapMode: TextEdit.WrapAnywhere
                    color: Theme.textPrimary
                    selectionColor: Theme.accent
                    selectedTextColor: Theme.textOnAccent
                    font.family: "monospace"
                    font.pixelSize: Theme.detailSize
                    Accessible.name: AppLanguage.tr("Command") + " " + (commandRow.index + 1)
                    background: Rectangle { color: Theme.bgPanelAlt; border.color: parent.activeFocus ? Theme.accentBright : Theme.borderLight }
                }
                ActionButton {
                    Layout.preferredWidth: 125
                    text: AppLanguage.tr(commandRow.copied ? "Copied" : commandRow.copyFailed ? "Copy failed" : "Copy")
                    Accessible.name: AppLanguage.tr("Copy command") + " " + (commandRow.index + 1)
                    onClicked: {
                        commandRow.copied = root.workspace.copyPresetCommand(commandRow.index);
                        commandRow.copyFailed = !commandRow.copied;
                        copyReset.restart();
                    }
                }
                Timer { id: copyReset; interval: 1800; onTriggered: { commandRow.copied = false; commandRow.copyFailed = false; } }
            }
        }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("Copying does not execute commands. HUP reloads Tor configuration; it does not guarantee a different IP."); color: Theme.textSecondary; font.pixelSize: Theme.detailSize; wrapMode: Text.WordWrap }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("Owned sessions") + ": " + root.workspace.sessions.length; color: Theme.textSecondary }
        ActionButton { Layout.fillWidth: true; visible: !root.workspace.terminalAvailable; text: AppLanguage.tr("Install terminal"); iconName: "start"; enabled: !root.workspace.busy; onClicked: terminalDialog.open() }
        ActionButton { Layout.fillWidth: true; visible: !!root.workspace.health.dependencies && root.workspace.health.dependencies.tmux === "missing"; text: AppLanguage.tr("Install tmux"); iconName: "start"; enabled: !root.workspace.busy; onClicked: installDialog.open() }
        Text { Layout.fillWidth: true; text: root.workspace.terminalMessage || root.workspace.message; color: Theme.textSecondary; wrapMode: Text.WordWrap }
        Text { Layout.fillWidth: true; text: AppLanguage.tr("Detach preserves the session. Stop terminates the owned session and closes its terminal. Each application needs its own proxy configuration; tmux and root privileges do not force traffic through Tor."); color: Theme.textSecondary; wrapMode: Text.WordWrap }
    }
    ConfirmDialog { id: startDialog; badge: "ROOT"; confirmText: AppLanguage.tr("Start session"); message: AppLanguage.tr("Verify Tor and open an external root terminal? An existing proxy is checked through check.torproject.org. No automatic commands are run after tmux starts."); onAccepted: root.workspace.sessionAction("start", "0") }
    ConfirmDialog { id: stopDialog; badge: "TMUX"; confirmText: AppLanguage.tr("Stop session"); message: AppLanguage.tr("Stop this session and its running commands?"); onAccepted: root.workspace.sessionAction("stop", root.workspace.sessions[0]) }
    ConfirmDialog { id: recoverDialog; badge: "TMUX"; confirmText: AppLanguage.tr("Recover proxy mode"); message: AppLanguage.tr("Terminate the owned root session and managed Tor instance, then restore the saved configuration? External edits are preserved and may require manual recovery."); onAccepted: root.workspace.recoverProxy() }
    ConfirmDialog { id: installDialog; badge: "TMUX"; confirmText: AppLanguage.tr("Install tmux"); message: AppLanguage.tr("Install the optional tmux package using the system package manager?"); onAccepted: root.workspace.installTmux() }
    ConfirmDialog { id: terminalDialog; badge: "TERM"; confirmText: AppLanguage.tr("Install terminal"); message: AppLanguage.tr("Install xterm using the system package manager?"); onAccepted: root.workspace.installTerminal() }
}
