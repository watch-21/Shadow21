import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Rectangle {
    id: card
    property string caption: ""
    property string value: ""
    property string detail: ""
    property color accent: "#64d7bf"
    color: "#172333"
    radius: 16
    border.color: "#2a3a4d"
    implicitHeight: content.implicitHeight + 36
    implicitWidth: 200
    ColumnLayout {
        id: content
        anchors.fill: parent
        anchors.margins: 18
        spacing: 9
        Label {
            text: card.caption
            color: "#a8b9ce"
            font.pixelSize: 12
            font.bold: true
            Layout.fillWidth: true
            wrapMode: Text.Wrap
        }
        Label {
            text: card.value
            color: card.accent
            font.pixelSize: 23
            font.bold: true
            Layout.fillWidth: true
            wrapMode: Text.Wrap
            Accessible.name: card.caption + ": " + card.value
        }
        Label {
            text: card.detail
            color: "#b5c4d6"
            font.pixelSize: 12
            Layout.fillWidth: true
            wrapMode: Text.Wrap
        }
    }
}
