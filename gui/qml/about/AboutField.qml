import QtQuick
import Shadow21.Desktop
import QtQuick.Layouts

ColumnLayout {
    id: root

    property string label: AppLanguage.tr("")
    property string value: ""
    property string link: ""

    spacing: 2

    Text {
        Layout.fillWidth: true
        text: root.label
        color: Theme.textSecondary
        font.family: Theme.uiFont
        font.pixelSize: Theme.badgeSize
        font.weight: Font.Medium
        elide: Text.ElideRight
    }

    Text {
        id: valueText

        Layout.fillWidth: true
        text: root.value
        color: root.link === "" ? Theme.textPrimary : linkArea.containsMouse ? Theme.accentBright : Theme.neonCyan
        font.family: Theme.uiFont
        font.pixelSize: Theme.bodySize
        font.bold: true
        font.underline: root.link !== ""
        elide: Text.ElideRight
        Accessible.role: root.link !== "" ? Accessible.Link : Accessible.StaticText
        Accessible.name: root.value

        Behavior on color  {
            ColorAnimation {
                duration: Theme.durationFast
            }
        }

        // The text stretches to fill the cell, so the hit area is clamped to
        // the glyphs to avoid a link reacting on empty space.
        MouseArea {
            id: linkArea

            width: Math.min(valueText.contentWidth, valueText.width)
            height: valueText.height
            enabled: root.link !== ""
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            onClicked: Qt.openUrlExternally(root.link)
        }

        ThemedToolTip {
            parent: valueText
            x: 0
            y: valueText.height + Theme.spacingTiny
            visible: linkArea.containsMouse
            text: root.link.replace("mailto:", "")
        }
    }
}
