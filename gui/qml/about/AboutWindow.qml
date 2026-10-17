import QtQuick
import Shadow21.Desktop
import QtQuick.Layouts
import QtQuick.Window

Window {
    id: root

    readonly property string projectUrl: "https://gitlab.com/parrotsec/packages/anonsurf"
    readonly property int preferredHeight: content.implicitHeight + Theme.spacingLarge * 2

    // Note:
    //   Centring is done by hand because a Window is placed by the window
    //   manager, unlike a popup that is laid out inside its parent.
    function open(): void {
        const owner = root.transientParent;
        if (owner) {
            root.x = owner.x + Math.round((owner.width - root.width) / 2);
            root.y = owner.y + Math.round((owner.height - root.height) / 2);
        }
        root.show();
        root.raise();
        root.requestActivate();
    }

    width: 460
    height: root.preferredHeight
    minimumWidth: root.width
    maximumWidth: root.width
    minimumHeight: root.preferredHeight
    maximumHeight: root.preferredHeight
    title: AppLanguage.tr("About Shadow21")
    color: Theme.bgPanel
    flags: Qt.Dialog

    Shortcut {
        sequence: "Escape"
        onActivated: root.close()
    }

    Rectangle {
        anchors.fill: parent
        color: "transparent"
        border.width: Theme.frameBorder
        border.color: Theme.borderLight
    }

    ColumnLayout {
        id: content

        anchors.fill: parent
        anchors.margins: Theme.spacingLarge
        spacing: Theme.spacingMedium

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingMedium

            Image {
                Layout.preferredWidth: 56
                Layout.preferredHeight: 56
                Layout.alignment: Qt.AlignVCenter
                source: "qrc:/icons/shadow21.svg"
                fillMode: Image.PreserveAspectFit
            }

            StatusBadge {
                Layout.alignment: Qt.AlignVCenter
                text: "VERSION " + Qt.application.version + " · 20261017-07"
                accentColor: Theme.neonCyan
            }

            Item {
                Layout.fillWidth: true
            }
        }

        Text {
            Layout.fillWidth: true
            text: AppLanguage.tr("Uses a transparent proxy for supported TCP traffic and DNS " + "lookups through Tor, with local-network exceptions. " + "Arbitrary UDP is not tunneled. IPv6 is disabled; " + "application cleanup is optional.")
            color: Theme.textSecondary
            font.family: Theme.uiFont
            font.pixelSize: Theme.bodySize
            wrapMode: Text.WordWrap
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.preferredHeight: 1
            color: Theme.borderLight
        }

        GridLayout {
            Layout.fillWidth: true
            Layout.topMargin: Theme.spacingSmall
            Layout.bottomMargin: Theme.spacingSmall
            columns: 2
            columnSpacing: Theme.spacingLarge
            rowSpacing: Theme.spacingSmall

            AboutField {
                Layout.fillWidth: true
                label: AppLanguage.tr("License")
                value: "GPL-3.0"
            }

            AboutField {
                Layout.fillWidth: true
                label: AppLanguage.tr("Maintainer")
                value: "shadow21 contributors"
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingSmall

            ActionButton {
                Layout.fillWidth: true
                Layout.preferredWidth: 130
                text: AppLanguage.tr("Project")
                iconName: "external"
                onClicked: Qt.openUrlExternally(root.projectUrl)
            }

            ActionButton {
                Layout.fillWidth: true
                Layout.preferredWidth: 150
                text: AppLanguage.tr("Report issue")
                iconName: "bug"
                onClicked: Qt.openUrlExternally(root.projectUrl + "/-/issues")
            }

            ActionButton {
                Layout.fillWidth: true
                Layout.preferredWidth: 110
                text: AppLanguage.tr("Close")
                iconName: "close"
                primary: true
                onClicked: root.close()
            }
        }
    }
}
