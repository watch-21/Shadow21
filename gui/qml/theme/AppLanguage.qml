pragma Singleton
import QtQuick
QtObject {
    property var workspace: null
    function tr(source) {
        if (workspace && workspace.language === "de")
            return workspace.translate(source);
        return source;
    }
}
