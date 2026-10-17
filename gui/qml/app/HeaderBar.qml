import QtQuick
import Shadow21.Desktop
import QtQuick.Layouts
import QtQuick.Controls

Item {
    id: root

    required property var workspace
    required property var backend
    property string selectedView: "system"
    signal aboutRequested
    implicitHeight: 44

    RowLayout {
        anchors.fill: parent
        spacing: Theme.spacingSmall

        Item {
            Layout.preferredWidth: 38
            Layout.preferredHeight: 38
            Layout.alignment: Qt.AlignVCenter

            Image {
                anchors.centerIn: parent
                width: 36
                height: 36
                source: "qrc:/icons/shadow21.svg"
                fillMode: Image.PreserveAspectFit
            }
        }

        ColumnLayout {
            Layout.fillWidth: true
            Layout.alignment: Qt.AlignVCenter
            spacing: Theme.spacingTiny

            Text {
                text: AppLanguage.tr("Shadow21") + " · v1"
                color: Theme.textPrimary
                font.family: Theme.uiFont
                font.pixelSize: Theme.headingSize
                font.bold: true
                font.letterSpacing: 1
            }

            Text {
                Layout.fillWidth: true
                text: !root.backend.anonsurfActive && (root.selectedView === "proxy" || root.workspace.modeStatus.mode === "proxy") ? AppLanguage.tr("Optional root tmux sessions through SOCKS") : AppLanguage.tr("Route system traffic through Tor")
                color: Theme.textSecondary
                font.family: Theme.uiFont
                font.pixelSize: Theme.detailSize
                elide: Text.ElideRight
            }
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

            Layout.preferredWidth: 100
            Layout.alignment: Qt.AlignVCenter
            model: ["English", "Deutsch"]
            currentIndex: root.workspace.language === "de" ? 1 : 0
            enabled: !root.workspace.busy
            Accessible.name: AppLanguage.tr("Language")
            onActivated: root.workspace.save(currentIndex === 1 ? "de" : "en", root.workspace.mode, root.workspace.endpoint, root.workspace.tmuxEnabled)
        }

        IconButton {
            Layout.alignment: Qt.AlignVCenter
            iconName: "info"
            tooltip: AppLanguage.tr("About Shadow21")
            onClicked: root.aboutRequested()
        }
    }
}
