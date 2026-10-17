import QtQuick
import Shadow21.Desktop
import QtQuick.Controls

Button {
    id: root

    property string iconName: ""
    property string tooltip: AppLanguage.tr("")
property bool primary: false
    property bool iconOnly: false

    implicitHeight: root.iconOnly ? Theme.buttonHeight : Math.max(Theme.buttonHeight, label.implicitHeight + 12)
    font.pixelSize: Theme.buttonSize
    font.family: Theme.uiFont
    font.bold: true
    padding: 0
    hoverEnabled: true
    focusPolicy: Qt.StrongFocus
    opacity: enabled ? 1 : 0.5
    Accessible.name: text

    contentItem: Item {
        transform: Translate {
            x: root.down ? Theme.buttonShadowSize : 0
            y: root.down ? Theme.buttonShadowSize : 0
        }

        Row {
            anchors.centerIn: parent
            spacing: Theme.spacingSmall

            Icon {
                icon: root.iconName
                color: label.color
                anchors.verticalCenter: parent.verticalCenter
                width: root.iconName !== "" ? 18 : 0
                visible: root.iconName !== ""
                height: 18
            }

            Text {
                id: label

                width: Math.max(30, root.availableWidth - (root.iconName !== "" ? 38 : 20))
                wrapMode: Text.Wrap
                horizontalAlignment: Text.AlignHCenter
                text: String(root.text).toUpperCase()
                visible: !root.iconOnly
                color: root.primary ? Theme.textOnAccent
                                    : (root.hovered ? Theme.accentBright : Theme.buttonText)
                font.family: Theme.uiFont
                font.pixelSize: Theme.buttonSize
                font.bold: true
                anchors.verticalCenter: parent.verticalCenter
                Behavior on color {
                    ColorAnimation { duration: Theme.durationFast }
                }
            }
        }
    }

    background: Item {
        Rectangle {
            x: root.down ? 0 : Theme.buttonShadowSize
            y: root.down ? 0 : Theme.buttonShadowSize
            width: parent.width
            height: parent.height
            color: root.primary ? Theme.buttonShadow : Theme.buttonSecondaryShadow
        }

        Rectangle {
            x: root.down ? Theme.buttonShadowSize : 0
            y: root.down ? Theme.buttonShadowSize : 0
            width: parent.width
            height: parent.height
            color: root.primary
                   ? (root.hovered ? Theme.buttonPrimaryHover : Theme.buttonPrimary)
                   : (root.hovered ? Theme.buttonSecondaryHover : Theme.buttonSecondary)
            border.width: Theme.frameBorder
            border.color: root.activeFocus ? Theme.accentBright : root.primary ? Theme.neonCyan : Theme.buttonBorder
            Behavior on color {
                ColorAnimation { duration: Theme.durationFast }
            }

            Rectangle {
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                height: 1
                color: Theme.buttonHighlight
                opacity: root.down ? 0 : 1
            }
        }
    }

    ThemedToolTip {
        parent: root
        x: Math.round((root.width - width) / 2)
        y: root.height + Theme.spacingTiny
        visible: root.tooltip !== "" && root.hovered
        text: root.tooltip
    }
}
