import QtQuick
import Shadow21.Desktop

Rectangle {
    id: root

    property alias text: label.text
    property color accentColor: Theme.accent
    property bool filled: false

    implicitWidth: label.implicitWidth + Theme.badgePaddingX * 2
    implicitHeight: Theme.badgeHeight
    color: filled ? accentColor : Theme.bgPanel
    border.width: 1
    border.color: accentColor

    Text {
        id: label

        anchors.centerIn: parent
        color: root.filled ? Theme.textOnAccent : root.accentColor
        font.family: Theme.uiFont
        font.pixelSize: Theme.badgeSize
        font.bold: true
    }
}
