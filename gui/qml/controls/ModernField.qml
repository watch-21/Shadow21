import QtQuick
import QtQuick.Controls

TextField {
    id: field
    implicitHeight: 44
    font.pixelSize: 13
    color: "#e4edf7"
    placeholderTextColor: "#93a9c3"
    selectionColor: "#64d7bf"
    selectedTextColor: "#102729"
    leftPadding: 12
    rightPadding: 12
    background: Rectangle {
        color: "#172333"
        radius: 8
        border.width: field.activeFocus ? 2 : 1
        border.color: field.activeFocus ? "#64d7bf" : "#3a4c63"
    }
}
