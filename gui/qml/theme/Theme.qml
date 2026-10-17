pragma Singleton
import QtQuick

QtObject {
    readonly property color accent: "#007CA6"
    readonly property color accentDim: "#005E85"
    readonly property color accentBright: "#20C7F4"
    readonly property color warning: "#F4C95D"
    readonly property color danger: "#FF453A"
    readonly property color info: "#007CA6"
    readonly property color neonCyan: "#38D9FF"
    readonly property color bgWindow: "#050D12"
    readonly property color bgPanel: "#081821"
    readonly property color bgPanelAlt: "#0C2633"
    readonly property color bgBottom: "#03090D"
    readonly property color textPrimary: "#F4FAFC"
    readonly property color textSecondary: "#A0B9C4"
    readonly property color textOnAccent: "#F7FCFF"
    readonly property color borderLight: Qt.rgba(0.22, 0.85, 1, 0.18)
    readonly property color buttonText: "#BDEEFF"
    readonly property color buttonPrimary: "#007CA6"
    readonly property color buttonPrimaryHover: "#006F96"
    readonly property color buttonSecondary: "#0A202A"
    readonly property color buttonSecondaryHover: "#0D3040"
    readonly property color buttonBorder: "#008DBD"
    readonly property color buttonShadow: "#00394D"
    readonly property color buttonSecondaryShadow: "#02080C"
    readonly property color buttonHighlight: Qt.rgba(0.75, 0.94, 1, 0.28)
    readonly property color scanline: Qt.rgba(0, 0, 0, 0.34)
    readonly property color vignette: Qt.rgba(0, 0, 0, 0.58)
    readonly property color grid: Qt.rgba(0, 0.49, 0.65, 0.09)

    readonly property string uiFont: "Inter"
    readonly property int titleSize: 26
    readonly property int headingSize: 18
    readonly property int bodySize: 15
    readonly property int buttonSize: 13
    readonly property int badgeSize: 13
    readonly property int detailSize: 14
    readonly property int spacingTiny: 3
    readonly property int spacingSmall: 6
    readonly property int spacingMedium: 7
    readonly property int spacingLarge: 10
    readonly property int contentMargin: 10
    readonly property int frameBorder: 2
    readonly property int buttonHeight: 34
    readonly property int buttonShadowSize: 3
    readonly property int iconButtonSize: 34
    readonly property int badgeHeight: 24
    readonly property int badgePaddingX: 9
    readonly property int durationFast: 150
    readonly property int durationMedium: 220
    readonly property int scanlineStep: 4
}
