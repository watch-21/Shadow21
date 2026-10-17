pragma Singleton
import QtQuick

QtObject {
    // Icon geometry taken from Lucide (https://lucide.dev): ISC licensed, with
    // the icons inherited from Feather under MIT. Every source icon is drawn on
    // a 24x24 grid and its sub-paths are merged into a single path string.
    readonly property var paths: ({
        "start": "M5 5a2 2 0 0 1 3.008-1.728l11.997 6.998a2 2 0 0 1 .003 3.458l-12 7A2 2 0 0 1 5 19z",
        "stop": "M18.36 6.64A9 9 0 0 1 20.77 15 M6.16 6.16a9 9 0 1 0 12.68 12.68 M12 2v4 M2 2l20 20",
        "restart": "M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8 M3 3v5h5",
        "refresh": "M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8 M21 3v5h-5 M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16 M8 16H3v5",
        "identity": "M18 11c-1.5 0-2.5.5-3 2 M4 6a2 2 0 0 0-2 2v4a5 5 0 0 0 5 5 8 8 0 0 1 5 2 8 8 0 0 1 5-2 5 5 0 0 0 5-5V8a2 2 0 0 0-2-2h-3a8 8 0 0 0-5 2 8 8 0 0 0-5-2z M6 11c1.5 0 2.5.5 3 2",
        "monitor": "M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2",
        "globe": "M2 12a10 10 0 1 0 20 0a10 10 0 1 0 -20 0Z M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20 M2 12h20",
        "shield": "M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z",
        "warning": "M21.73 18l-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3 M12 9v4 M12 17h.01",
        "check": "M20 6 9 17l-5-5",
        "info": "M2 12a10 10 0 1 0 20 0a10 10 0 1 0 -20 0Z M12 16v-4 M12 8h.01",
        "external": "M15 3h6v6 M10 14 21 3 M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6",
        "close": "M18 6 6 18 M6 6l12 12",
        "bug": "M12 20v-9 M14 7a4 4 0 0 1 4 4v3a6 6 0 0 1-12 0v-3a4 4 0 0 1 4-4z M14.12 3.88 16 2 M21 21a4 4 0 0 0-3.81-4 M21 5a4 4 0 0 1-3.55 3.97 M22 13h-4 M3 21a4 4 0 0 1 3.81-4 M3 5a4 4 0 0 0 3.55 3.97 M6 13H2 M8 2l1.88 1.88 M9 7.13V6a3 3 0 1 1 6 0v1.13",
    })
}
