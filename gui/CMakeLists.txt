cmake_minimum_required(VERSION 3.22)
project(shadow21-gui VERSION 1 LANGUAGES CXX)

# Deployment uses distribution libraries, never a developer SDK search path.
set(CMAKE_SKIP_RPATH TRUE)

set(ANONSURF_VERSION "v1" CACHE STRING
    "Version displayed by the shadow21 GUI"
)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)


find_package(Qt6 6.4 REQUIRED COMPONENTS Core DBus Gui Qml Quick QuickControls2 Widgets Network)
if(Qt6_VERSION VERSION_GREATER_EQUAL "6.8")
 qt_policy(SET QTP0004 NEW)
endif()

qt_add_executable(shadow21-gui
    src/application/workspace.cpp
    src/application/workspace.h
    src/application/externalterminal.cpp
    src/application/externalterminal.h
    src/application/main.cpp
    src/application/instance.cpp
    src/application/instance.h
    src/anonsurf/anonsurfactions.cpp
    src/anonsurf/anonsurfbackend.cpp
    src/anonsurf/anonsurfbackend.h
    src/anonsurf/anonsurfstate.cpp
    src/cli/commandclient.cpp
    src/cli/commandclient.h
    src/notifications/notifications.cpp
    src/notifications/notifications.h
    src/systemd/systemdclient.cpp
    src/systemd/systemdclient.h
    src/systemd/systemdjobs.cpp
    src/tray/traycontroller.cpp
    src/tray/traycontroller.h
    src/tray/traymenu.cpp
    src/tray/traymenu.h
)

target_include_directories(shadow21-gui PRIVATE src)

set_source_files_properties(qml/theme/Theme.qml qml/theme/IconPaths.qml qml/theme/AppLanguage.qml PROPERTIES QT_QML_SINGLETON_TYPE TRUE)
qt_add_qml_module(shadow21-gui
 URI Shadow21.Desktop VERSION 1.0 RESOURCE_PREFIX /qt/qml
 QML_FILES
  qml/about/AboutField.qml
  qml/about/AboutWindow.qml
  qml/app/ConfirmDialog.qml
  qml/app/CrtOverlay.qml
  qml/app/ErrorDialog.qml
  qml/app/HeaderBar.qml
  qml/app/Main.qml
  qml/app/TmuxPanel.qml
  qml/dashboard/MacPanel.qml
  qml/controls/ActionButton.qml
  qml/controls/Icon.qml
  qml/controls/IconButton.qml
  qml/controls/StatusBadge.qml
  qml/controls/StatusItem.qml
  qml/controls/ThemedToolTip.qml
  qml/controls/ToggleOption.qml
  qml/dashboard/ActionPanel.qml
  qml/dashboard/ConnectionOverview.qml
  qml/dashboard/NetworkOverview.qml
  qml/dashboard/ProtectionPanel.qml
  qml/theme/IconPaths.qml
  qml/theme/AppLanguage.qml
  qml/theme/Theme.qml
)

set(ANONSURF_ICON "${CMAKE_CURRENT_SOURCE_DIR}/../icons/shadow21.svg")
set_source_files_properties("${ANONSURF_ICON}" PROPERTIES
    QT_RESOURCE_ALIAS icons/shadow21.svg
)
qt_add_resources(shadow21-gui "anonsurf-assets"
    PREFIX /
    FILES "${ANONSURF_ICON}"
)

target_compile_options(shadow21-gui PRIVATE
    "$<$<COMPILE_LANG_AND_ID:CXX,GNU,Clang,AppleClang>:-Wall;-Wextra;-Wpedantic>"
)
target_compile_definitions(shadow21-gui PRIVATE
    ANONSURF_VERSION="${ANONSURF_VERSION}"
)
target_link_libraries(shadow21-gui PRIVATE
    Qt6::Core
    Qt6::DBus
    Qt6::Gui
    Qt6::Qml
    Qt6::Quick
    Qt6::QuickControls2
    Qt6::Widgets
    Qt6::Network
)

install(TARGETS shadow21-gui RUNTIME DESTINATION bin)
install(FILES org.shadow21.Desktop.desktop DESTINATION share/applications)
install(FILES "${ANONSURF_ICON}"
    DESTINATION share/icons/hicolor/512x512/apps
    RENAME shadow21.svg
)

qt_add_resources(shadow21-gui "translations" PREFIX "/" FILES i18n/de.json)

add_executable(shadow21-workspace-test EXCLUDE_FROM_ALL
 src/application/externalterminal.cpp tests/workspace_test.cpp src/application/workspace.cpp src/application/workspace.h)
target_include_directories(shadow21-workspace-test PRIVATE src)
target_link_libraries(shadow21-workspace-test PRIVATE Qt6::Core Qt6::Gui Qt6::Network)

add_executable(shadow21-telemetry-test EXCLUDE_FROM_ALL
 src/application/externalterminal.cpp tests/telemetry_test.cpp src/application/workspace.cpp src/application/workspace.h)
target_include_directories(shadow21-telemetry-test PRIVATE src)
target_link_libraries(shadow21-telemetry-test PRIVATE Qt6::Core Qt6::Gui Qt6::Network)


add_executable(shadow21-mode-test EXCLUDE_FROM_ALL
 src/application/externalterminal.cpp tests/mode_test.cpp src/application/workspace.cpp src/application/workspace.h
 src/anonsurf/anonsurfbackend.cpp src/anonsurf/anonsurfbackend.h
 src/anonsurf/anonsurfactions.cpp src/anonsurf/anonsurfstate.cpp
 src/cli/commandclient.cpp src/cli/commandclient.h
 src/systemd/systemdclient.cpp src/systemd/systemdclient.h src/systemd/systemdjobs.cpp)
target_include_directories(shadow21-mode-test PRIVATE src)
target_link_libraries(shadow21-mode-test PRIVATE Qt6::Core Qt6::Gui Qt6::Network Qt6::DBus)

add_executable(shadow21-externalterminal-test EXCLUDE_FROM_ALL
 tests/externalterminal_test.cpp src/application/externalterminal.cpp)
target_include_directories(shadow21-externalterminal-test PRIVATE src)
target_link_libraries(shadow21-externalterminal-test PRIVATE Qt6::Core)

add_executable(shadow21-clipboard-test EXCLUDE_FROM_ALL
 src/application/externalterminal.cpp tests/clipboard_test.cpp src/application/workspace.cpp src/application/workspace.h)
target_include_directories(shadow21-clipboard-test PRIVATE src)
target_link_libraries(shadow21-clipboard-test PRIVATE Qt6::Core Qt6::Gui Qt6::Network)
