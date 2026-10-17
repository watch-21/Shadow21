import QtQuick
import QtQuick.Controls

ComboBox {
    id: select
    implicitHeight: 42
    font.pixelSize: 13
    background: Rectangle {
        color: select.hovered ? "#293e54" : "#1d2d41"
        radius: 8
        border.width: select.activeFocus ? 2 : 1
        border.color: select.activeFocus ? "#64d7bf" : "#3a4c63"
    }
    contentItem: Text {
        text: select.displayText
        font: select.font
        color: "#e4edf7"
        leftPadding: 10
        rightPadding: 28
        verticalAlignment: Text.AlignVCenter
        elide: Text.ElideRight
    }
}
