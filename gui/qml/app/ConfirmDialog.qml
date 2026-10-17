import QtQuick
import Shadow21.Desktop
import QtQuick.Controls
import QtQuick.Layouts

Dialog {
    id: root

    property string badge: ""
    property string message: AppLanguage.tr("")
    property string confirmText: AppLanguage.tr("Confirm")
    property color accentColor: Theme.warning
    readonly property int maxWidth: 560

    parent: Overlay.overlay
    anchors.centerIn: parent
    width: parent ? Math.min(root.maxWidth, parent.width - Theme.contentMargin * 2) : root.maxWidth
    height: parent ? Math.min(implicitHeight, parent.height - Theme.contentMargin * 2) : implicitHeight
    modal: true
    focus: true
    padding: Theme.spacingLarge
    standardButtons: Dialog.NoButton
    header: null

    contentItem: ColumnLayout {
        spacing: Theme.spacingMedium

        StatusBadge {
            Layout.alignment: Qt.AlignLeft | Qt.AlignVCenter
            text: root.badge
            accentColor: root.accentColor
        }

        ScrollView {
            id: messageScroll

            Layout.fillWidth: true
            Layout.fillHeight: true
            implicitHeight: messageText.implicitHeight
            contentWidth: availableWidth
            clip: true
            ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
            ScrollBar.vertical.policy: ScrollBar.AsNeeded

            Text {
                id: messageText

                width: messageScroll.availableWidth
                text: root.message
                color: Theme.textPrimary
                font.family: Theme.uiFont
                font.pixelSize: Theme.bodySize
                wrapMode: Text.WordWrap
            }
        }

        GridLayout {
            columns: root.width < 500 ? 1 : 2
            Layout.fillWidth: true
            Layout.topMargin: Theme.spacingSmall
            columnSpacing: Theme.spacingSmall
            rowSpacing: Theme.spacingSmall

            ActionButton {
                Layout.fillWidth: true
                text: AppLanguage.tr("Cancel")
                iconName: "close"
                onClicked: root.reject()
            }

            ActionButton {
                Layout.fillWidth: true
                text: root.confirmText
                iconName: "check"
                primary: true
                onClicked: root.accept()
            }
        }
    }

    background: Rectangle {
        color: Theme.bgPanel
        border.width: Theme.frameBorder
        border.color: root.accentColor
    }
}
