import QtQuick
import Shadow21.Desktop
import QtQuick.Controls
import QtQuick.Layouts
import QtQuick.Window

ApplicationWindow {
    id: window

    required property var backend
    required property var workspace
    required property var desktop
    property bool safeKill: false
    readonly property int selectedModeIndex: window.backend.anonsurfActive ? 0 : window.workspace.modeStatus.mode === "proxy" || window.workspace.mode === "proxy" ? 1 : 0
    // The tray needs fresh status even while the window is hidden.
    readonly property bool live: window.visible && window.visibility !== Window.Minimized
    property string viewMode: "system"
    readonly property int contentMargins: 2 * (window.width < 600 ? 8 : Theme.contentMargin)
    width: Math.min(1100, Screen.width - 32)
    height: Math.min(800, Screen.height - 48)
    minimumWidth: Math.min(720, Screen.width - 32)
    minimumHeight: Math.min(540, Screen.height - 48)
    visible: true
    title: AppLanguage.tr("Shadow21")
    color: Theme.bgWindow
    font.family: Theme.uiFont

    Component.onCompleted: {
        AppLanguage.workspace = window.workspace;
        window.backend.refresh();
        window.workspace.load();
        window.workspace.refreshModeStatus();
    }
    ConfirmDialog {
        id: modeSwitchDialog
        property string targetMode: "system"
        badge: "MODE"
        confirmText: AppLanguage.tr("Switch mode")
        message: AppLanguage.tr("Stop the current mode and its owned session before switching? Selecting tmux starts the Tor service; its session stays stopped until Start session.")
        onAccepted: window.backend.selectMode(targetMode)
    }
    function chooseMode(target) {
        if (window.workspace.busy || window.backend.busy) return;
        if (target === "system" && window.viewMode === target && window.workspace.mode === target) return;
        if (target === "proxy" && window.workspace.modeStatus.phase === "active" && window.workspace.modeStatus.mode === "proxy") { window.viewMode = target; return; }
        if (window.backend.anonsurfActive || window.workspace.modeStatus.phase !== "inactive") {
            modeSwitchDialog.targetMode = target;
            modeSwitchDialog.open();
        } else {
            window.backend.selectMode(target);
        }
    }
    function openTmux() {
        window.viewMode = "proxy"; window.workspace.refreshSessions();
    }
    function openInspectionDialog(kind) { if(kind === "mac" || kind === "mac-restore") macPanel.openConfirmation(kind === "mac-restore"); else tmuxPanel.openInspectionDialog(kind); }
    onClosing: close => {
        if (window.workspace.busy || window.backend.busy) { close.accepted = false; return; }
        aboutWindow.close();
        if (window.desktop.hideToTray())
            close.accepted = false;
    }

    onLiveChanged: if (window.live)
        window.backend.refresh()

    Timer {
        interval: 3000
        running: window.live || window.desktop.trayAvailable
        repeat: true
        onTriggered: window.backend.refresh()
    }

    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop {
                position: 0
                color: Theme.bgWindow
            }
            GradientStop {
                position: 0.62
                color: "#071923"
            }
            GradientStop {
                position: 1
                color: Theme.bgBottom
            }
        }
    }

    CrtOverlay {
        anchors.fill: parent
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: window.width < 600 ? 8 : Theme.contentMargin
        spacing: Theme.spacingMedium

        HeaderBar {
            id: headerBar

            Layout.fillWidth: true
            workspace: window.workspace
            backend: window.backend
            selectedView: window.viewMode
            onAboutRequested: aboutWindow.open()
        }

        ScrollView {
            id: contentScroll

            Layout.fillWidth: true
            Layout.fillHeight: true
            contentWidth: availableWidth
            clip: true

            ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
            ScrollBar.vertical.policy: ScrollBar.AsNeeded

            Column {
                id: contentColumn

                width: contentScroll.availableWidth
                spacing: Theme.spacingMedium

                RowLayout {
                    width: parent.width
                    spacing: Theme.spacingSmall
                    Text {
                        text: AppLanguage.tr("Connection mode")
                        color: Theme.textSecondary
                        font.pixelSize: Theme.detailSize
                    }
                    ActionButton {
                        Layout.fillWidth: true
                        text: "Shadow21"
                        iconName: "start"
                        primary: window.viewMode === "system"
                        enabled: window.workspace.ready && !window.workspace.busy && !window.backend.busy && !(window.workspace.terminalRunning && window.workspace.modeStatus.phase !== "active")
                        onClicked: window.chooseMode("system")
                    }
                    ActionButton {
                        Layout.fillWidth: true
                        text: "tmux"
                        iconName: "monitor"
                        primary: window.viewMode === "proxy"
                        enabled: window.workspace.ready && !window.workspace.busy && !window.backend.busy && !(window.workspace.terminalRunning && window.workspace.modeStatus.phase !== "active")
                        onClicked: window.chooseMode("proxy")
                    }
                }
                Text {
                    width: parent.width
                    text: window.workspace.torPreparing ? window.workspace.message : window.workspace.modeStatus.phase !== "inactive" ? AppLanguage.tr("Proxy mode is active or needs recovery. Stop it before switching.") : window.backend.anonsurfActive ? AppLanguage.tr("Stop system routing before switching modes.") : AppLanguage.tr("Select one mode. System routing and tmux cannot run together.")
                    color: Theme.textSecondary
                    font.pixelSize: Theme.detailSize
                    wrapMode: Text.WordWrap
                }
                ProtectionPanel {
                    visible: window.viewMode === "system"
                    width: parent.width
                    backend: window.backend
                    workspace: window.workspace
                    safeKill: window.safeKill
                    onSafeKillToggled: enabled => {
                        if (enabled)
                            safeKillDialog.open();
                        else
                            window.safeKill = false;
                    }
                }

                ConnectionOverview {
                    width: parent.width
                    backend: window.backend
                    workspace: window.workspace
                }

                NetworkOverview {
                    width: parent.width
                    workspace: window.workspace
                }

                ActionPanel {
                    visible: window.viewMode === "system"
                    width: parent.width
                    backend: window.backend
                    workspace: window.workspace
                    safeKill: window.safeKill
                }

                TmuxPanel {
                    id: tmuxPanel
                    objectName: "tmuxPanel"
                    width: parent.width
                    visible: window.viewMode === "proxy"
                    workspace: window.workspace
                }
                MacPanel {
                    id: macPanel
                    width: parent.width
                    workspace: window.workspace
                }
            }
        }
    }

    AboutWindow {
        id: aboutWindow
    }

    ConfirmDialog {
        id: safeKillDialog

        badge: "SAFEKILL"
        confirmText: AppLanguage.tr("Enable SafeKill")
        message: AppLanguage.tr("Enabling SafeKill does not run it now. Before each start, stop " + "or restart, it requests termination of selected applications, " + "then clears selected caches, history and sessions with BleachBit. " + "Unsaved work may be lost and cleared data cannot be recovered.")
        onAccepted: window.safeKill = true
    }

    ErrorDialog {
        id: errorDialog

        backend: window.backend
    }

    Connections {
        target: window.workspace
        function onModeReady(mode: string) { window.viewMode = Qt.application.arguments.indexOf("--inspect-ui") >= 0 && Qt.application.arguments.indexOf("--tmux") >= 0 ? "proxy" : mode; }
    }
    Connections {
        target: window.desktop

        function onLifecycleRequested(action: string) {
            if (action === "start")
                window.backend.start(window.safeKill);
            else if (action === "stop")
                window.backend.stop(window.safeKill);
            else if (action === "restart")
                window.backend.restart(window.safeKill);
        }
    }

    Connections {
        target: window.backend

        function onChanged() {
            if (window.backend.errorMessage !== "" && !errorDialog.opened)
                errorDialog.open();
        }
    }
}
