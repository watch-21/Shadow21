import QtQuick
import Shadow21.Desktop
import QtQuick.Controls

ToolTip {
    id: root

    property int maxWidth: 300

    delay: 400
    // Keeps the tip inside the window instead of letting it overflow the edge.
    margins: Theme.spacingSmall
    width: Math.min(root.implicitWidth, root.maxWidth)

    contentItem: Text {
        text: root.text
        color: Theme.textPrimary
        font.family: Theme.uiFont
        font.pixelSize: Theme.badgeSize
        wrapMode: Text.WordWrap
    }

    background: Rectangle {
        color: Theme.bgBottom
        border.width: 1
        border.color: Theme.buttonBorder
    }
}
