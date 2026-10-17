import QtQuick
import Shadow21.Desktop
import QtQuick.Layouts

Item {
    id: root

    property string label: AppLanguage.tr("")
property string value: ""
    property string detail: ""
    property string timestamp: ""
    property string tooltip: AppLanguage.tr("")
property color accentColor: Theme.neonCyan

    implicitHeight: content.implicitHeight

    HoverHandler {
        id: hover

        enabled: root.tooltip !== ""
    }

    ThemedToolTip {
        parent: root
        x: 0
        y: root.height + Theme.spacingTiny
        visible: hover.hovered
        text: root.tooltip
    }

    ColumnLayout {
        id: content

        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        spacing: Theme.spacingTiny

        Text {
            Layout.fillWidth: true
            wrapMode: Text.WordWrap
            text: root.label
            color: Theme.textSecondary
            font.family: Theme.uiFont
            font.pixelSize: Theme.detailSize
            font.weight: Font.Medium
        }

        Text {
            Layout.fillWidth: true
            text: root.value
            color: root.accentColor
            font.family: Theme.uiFont
            font.pixelSize: Theme.bodySize
            font.bold: true
            wrapMode: Text.Wrap
        }

        Text {
            Layout.fillWidth: true
            text: root.detail
            color: Theme.textSecondary
            font.family: Theme.uiFont
            font.pixelSize: Theme.detailSize
            wrapMode: Text.WordWrap
        }
        Text {
            Layout.fillWidth: true
            visible: root.timestamp !== ""
            text: root.timestamp !== "" ? AppLanguage.tr("Last successful check: ") + Qt.formatDateTime(new Date(root.timestamp), "yyyy-MM-dd HH:mm:ss") : ""
            color: Theme.textSecondary
            font.pixelSize: Theme.detailSize
            wrapMode: Text.Wrap
        }
    }
}
