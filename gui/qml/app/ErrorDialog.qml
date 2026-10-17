import QtQuick
import Shadow21.Desktop
import QtQuick.Controls
import QtQuick.Layouts

Dialog {
    id: root

    required property var backend
    readonly property int maxWidth: 520

    parent: Overlay.overlay
    anchors.centerIn: parent
    width: parent ? Math.min(root.maxWidth, parent.width - Theme.contentMargin * 2) : root.maxWidth
    height: parent ? Math.min(380, parent.height - Theme.contentMargin * 2) : 380
    modal: true
    focus: true
    padding: Theme.spacingLarge
    standardButtons: Dialog.NoButton
    title: AppLanguage.tr("Action failed")
    header: null
    onClosed: root.backend.dismissError()

    contentItem: ColumnLayout {
        spacing: Theme.spacingMedium

        StatusBadge {
            Layout.alignment: Qt.AlignLeft | Qt.AlignVCenter
            text: AppLanguage.tr("SYSTEM ERROR")
            accentColor: Theme.danger
        }

        ScrollView {
            id: errorScroll

            Layout.fillWidth: true
            Layout.fillHeight: true
            contentWidth: availableWidth
            clip: true
            ScrollBar.horizontal.policy: ScrollBar.AlwaysOff

            TextEdit {
                readOnly: true
                selectByMouse: true
                textFormat: TextEdit.PlainText
                width: errorScroll.availableWidth
                text: root.backend.errorMessage
                color: Theme.textSecondary
                font.family: Theme.uiFont
                font.pixelSize: Theme.bodySize
                wrapMode: TextEdit.WordWrap
            }
        }

        ActionButton {
            Layout.alignment: Qt.AlignRight
            Layout.preferredWidth: 130
            text: AppLanguage.tr("Dismiss")
            iconName: "close"
            onClicked: root.close()
        }
    }

    background: Rectangle {
        color: Theme.bgPanel
        border.width: Theme.frameBorder
        border.color: Theme.danger
    }
}
