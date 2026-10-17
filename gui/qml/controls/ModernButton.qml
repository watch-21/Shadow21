import QtQuick
import QtQuick.Controls

Button {
    id: button
    property bool primary: false
    property bool danger: false
    implicitHeight: 44
    implicitWidth: Math.max(110, contentItem.implicitWidth + 32)
    font.pixelSize: 13
    font.bold: true
    Accessible.name: text
    contentItem: Text {
        text: button.text
        font: button.font
        color: !button.enabled ? "#768498" : button.primary ? "#0d2221" : "#e6edf6"
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
    }
    background: Rectangle {
        radius: 9
        color: !button.enabled ? "#233042" : button.down ? "#36566a" : button.primary ? "#64d7bf" : button.danger ? "#643140" : button.hovered ? "#304358" : "#233449"
        border.width: button.activeFocus ? 2 : 1
        border.color: button.activeFocus ? "#a7f4e6" : button.primary ? "#64d7bf" : "#3b5068"
    }
}
