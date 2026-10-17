import QtQuick
import Shadow21.Desktop
import QtQuick.Controls

CheckBox {
    id: root

    property alias label: textLabel.text
    property string tooltip: AppLanguage.tr("")
property color accentColor: Theme.accentBright

    spacing: Theme.spacingSmall
    hoverEnabled: true
    opacity: enabled ? 1 : 0.55
    Accessible.name: label
    Accessible.description: tooltip

    indicator: Rectangle {
        x: 0
        y: parent.height / 2 - height / 2
        width: 20
        height: 20
        color: root.checked ? root.accentColor : Theme.bgPanelAlt
        border.width: root.activeFocus ? 2 : 1
        border.color: root.activeFocus || root.checked ? root.accentColor : Theme.textSecondary

        Icon {
            anchors.centerIn: parent
            width: 12
            height: 12
            icon: "check"
            color: Theme.bgWindow
            visible: root.checked
        }
    }

    contentItem: Text {
        id: textLabel

        leftPadding: 28
        verticalAlignment: Text.AlignVCenter
        color: root.checked ? root.accentColor : Theme.textPrimary
        font.family: Theme.uiFont
        font.pixelSize: Theme.detailSize
    }

    ThemedToolTip {
        parent: root
        x: 0
        y: root.height + Theme.spacingTiny
        visible: root.tooltip !== "" && root.hovered
        text: root.tooltip
    }
}
