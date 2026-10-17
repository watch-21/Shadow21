import QtQuick
import Shadow21.Desktop
import QtQuick.Layouts

Rectangle {
    id: root

    required property var backend
    required property var workspace
    property bool safeKill: false
    signal safeKillToggled(bool enabled)
    readonly property bool compact: width < 620
    readonly property bool veryCompact: width < 500
    readonly property color stateColor: root.backend.protectionState === "ROUTED" ? Theme.neonCyan : root.backend.protectionState === "DEGRADED" || root.backend.protectionState === "UNVERIFIED" ? Theme.warning : root.backend.protectionState === "INACTIVE" ? Theme.textSecondary : Theme.danger

    implicitHeight: content.implicitHeight + Theme.spacingLarge * 2
    color: Theme.bgPanel
    border.width: 1
    border.color: Theme.borderLight

    ColumnLayout {
        id: content

        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: Theme.spacingLarge
        spacing: Theme.spacingMedium

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingLarge

            Icon {
                Layout.preferredWidth: visible ? 48 : 0
                Layout.preferredHeight: visible ? 48 : 0
                Layout.alignment: Qt.AlignVCenter
                visible: !root.veryCompact
                icon: root.backend.protectionState === "DEGRADED" || root.backend.protectionState === "UNVERIFIED" ? "warning" : "shield"
                color: root.stateColor

                SequentialAnimation on opacity  {
                    running: root.backend.busy
                    loops: Animation.Infinite
                    NumberAnimation {
                        to: 0.45
                        duration: 500
                    }
                    NumberAnimation {
                        to: 1
                        duration: 500
                    }
                }
            }

            ColumnLayout {
                Layout.fillWidth: true
                Layout.alignment: Qt.AlignVCenter
                spacing: Theme.spacingSmall

                Text {
                    text: AppLanguage.tr("Routing status")
                    color: Theme.textSecondary
                    font.family: Theme.uiFont
                    font.pixelSize: Theme.detailSize
                }

                Text {
                    Layout.fillWidth: true
                    wrapMode: Text.WordWrap
                    text: AppLanguage.tr(root.backend.protectionState)
                    color: root.stateColor
                    font.family: Theme.uiFont
                    font.pixelSize: root.veryCompact ? Theme.bodySize + 2 : root.compact ? Theme.headingSize + 2 : Theme.titleSize
                    font.bold: true
                    font.letterSpacing: root.compact ? 1 : 2
                }

                // Keep lifecycle progress in the status panel.
                Text {
                    Layout.fillWidth: true
                    text: AppLanguage.tr(root.backend.busy ? root.backend.progress : root.backend.protectionDetail)
                    color: root.backend.busy ? Theme.neonCyan : Theme.textPrimary
                    font.family: Theme.uiFont
                    font.pixelSize: Theme.bodySize
                    wrapMode: Text.WordWrap
                    visible: !root.compact || root.backend.busy
                }
            }

            ActionButton {
                Layout.preferredWidth: root.compact ? 150 : 190
                Layout.fillWidth: false
                Layout.alignment: Qt.AlignVCenter
                text: AppLanguage.tr(root.backend.anonsurfActive ? "Stop shadow21" : "Start shadow21")
                iconName: root.backend.anonsurfActive ? "stop" : "start"
                tooltip: AppLanguage.tr(root.backend.anonsurfActive ? "Removes shadow21 routing and restores the previous network settings." : "Redirects supported TCP and DNS traffic through Tor. Private networks and loopback remain excluded; other UDP traffic is not tunneled.")
                primary: true
                enabled: (root.workspace.mode === "system" || root.backend.anonsurfActive) && root.workspace.modeStatus.phase === "inactive" && root.backend.statusAvailable && !root.backend.busy && !root.workspace.busy
                onClicked: root.backend.anonsurfActive ? root.backend.stop(root.safeKill) : root.backend.start(root.safeKill)
            }
        }

        ActionButton {
            Layout.fillWidth: true
            visible: root.backend.safeKillRunning
            text: AppLanguage.tr("Cancel cleanup")
            iconName: "close"
            onClicked: root.backend.cancelSafeKill()
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 1
            color: Theme.borderLight
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingLarge

            ToggleOption {
                Layout.alignment: Qt.AlignTop
                label: AppLanguage.tr("Use SafeKill")
                tooltip: safeKillDescription.text + AppLanguage.tr(" Uses BleachBit. Unsaved work may be lost and cleared data cannot be recovered.")
                accentColor: Theme.warning
                checked: root.safeKill
                enabled: !root.backend.busy
                // A checkbox click overwrites the binding, so it is restored to
                // keep the box following the state the window actually holds.
                onToggled: {
                    root.safeKillToggled(checked);
                    checked = Qt.binding(() => root.safeKill);
                }
            }

            Text {
                id: safeKillDescription

                Layout.fillWidth: true
                Layout.alignment: Qt.AlignVCenter
                text: AppLanguage.tr("Before start, stop or restart: closes selected apps " + "and clears caches, history and sessions.")
                color: Theme.textSecondary
                font.family: Theme.uiFont
                font.pixelSize: Theme.detailSize
                wrapMode: Text.WordWrap
            }
        }
    }

    Rectangle {
        anchors.left: parent.left
        anchors.bottom: parent.bottom
        width: root.backend.busy ? parent.width : 0
        height: 2
        color: Theme.neonCyan
        Behavior on width  {
            NumberAnimation {
                duration: Theme.durationMedium
            }
        }
    }
}
