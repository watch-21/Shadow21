import QtQuick
import Shadow21.Desktop
import QtQuick.Shapes

Item {
    id: root

    property string icon: ""
    property color color: Theme.textPrimary
    property real lineWidth: 2

    implicitWidth: 16
    implicitHeight: 16

    Shape {
        width: 24
        height: 24
        // Call sites size icons freely, so the whole shape is scaled: the
        // stroke then keeps the same weight relative to the drawing.
        transform: Scale {
            xScale: root.width / 24
            yScale: root.height / 24
        }

        ShapePath {
            strokeColor: root.color
            strokeWidth: root.lineWidth
            fillColor: "transparent"
            capStyle: ShapePath.RoundCap
            joinStyle: ShapePath.RoundJoin

            PathSvg { path: IconPaths.paths[root.icon] ?? "" }
        }
    }
}
