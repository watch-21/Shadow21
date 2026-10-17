import QtQuick
import Shadow21.Desktop
import QtQuick.Controls

Button {
    id: root

    property string iconName: ""
    property string tooltip: AppLanguage.tr("")
implicitWidth: Theme.iconButtonSize
    implicitHeight: Theme.iconButtonSize
    padding: 0
    hoverEnabled: true
    opacity: enabled ? 1 : 0.5
    Accessible.name: root.tooltip !== "" ? root.tooltip : root.text

    contentItem: Item {
        transform: Translate { y: root.down ? 1 : 0 }

        Icon {
            anchors.centerIn: parent
            width: 20
            height: 20
            icon: root.iconName
            color: root.hovered || root.visualFocus ? Theme.accentBright
                                                    : Theme.textSecondary
        }
    }

    background: Rectangle {
        color: root.down ? Theme.bgPanelAlt
                         : root.hovered ? Theme.buttonSecondaryHover
                                        : Theme.buttonSecondary
        border.width: 1
        border.color: root.visualFocus ? Theme.neonCyan
                                       : root.hovered ? Theme.buttonBorder
                                                      : Theme.borderLight

        Behavior on color {
            ColorAnimation { duration: Theme.durationFast }
        }

        Behavior on border.color {
            ColorAnimation { duration: Theme.durationFast }
        }
    }

    MouseArea {
        anchors.fill: parent
        acceptedButtons: Qt.NoButton
        cursorShape: root.enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
    }

    ThemedToolTip {
        parent: root
        x: Math.round((root.width - width) / 2)
        y: root.height + Theme.spacingTiny
        visible: root.tooltip !== "" && (root.hovered || root.visualFocus)
        text: root.tooltip
    }
}
