cmake_minimum_required(VERSION 3.20)
project(khaapp VERSION 0.1.0)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(QT_DEFAULT_MAJOR_VERSION 6)

find_package(ECM REQUIRED NO_MODULE)
set(CMAKE_MODULE_PATH ${ECM_MODULE_PATH})

include(KDEInstallDirs)
include(KDECMakeSettings)
include(KDECompilerSettings NO_POLICY_SCOPE)

find_package(Qt6 REQUIRED COMPONENTS Core Gui Qml Quick DBus Multimedia Widgets LinguistTools)
find_package(KF6Kirigami REQUIRED)
find_package(KF6Notifications REQUIRED)
find_package(KF6KIO REQUIRED)
find_package(KF6StatusNotifierItem REQUIRED)
find_package(KF6Wallet REQUIRED)

set(TS_FILES
    translations/khaapp_en.ts
    translations/khaapp_it.ts
)

qt6_create_translation(QM_FILES ${CMAKE_CURRENT_SOURCE_DIR}/qml ${TS_FILES})
foreach(qm_file IN LISTS QM_FILES)
    get_filename_component(qm_name "${qm_file}" NAME)
    set_source_files_properties("${qm_file}" PROPERTIES QT_RESOURCE_ALIAS "${qm_name}")
endforeach()

add_executable(khaapp
    src/main.cpp
    src/dbusbridge.cpp
    src/trayicon.cpp
    src/applock.cpp
    src/kdeconnectbridge.cpp
)

set_target_properties(khaapp PROPERTIES OUTPUT_NAME "khaapp-bin")
add_compile_definitions(KHAAPP_VERSION="0.1.0-beta.1")

target_include_directories(khaapp PRIVATE src)

target_link_libraries(khaapp PRIVATE
    Qt6::Core Qt6::Gui Qt6::Qml Qt6::Quick Qt6::DBus Qt6::Multimedia Qt6::Widgets
    KF6::Kirigami KF6::Notifications KF6::KIOGui KF6::StatusNotifierItem KF6::Wallet
)

qt6_add_resources(khaapp "resources"
    PREFIX "/"
    FILES
        qml/Main.qml
        qml/LoginPage.qml
        qml/ChatListPage.qml
        qml/ChatPage.qml
        qml/LinkPreviewCard.qml
        qml/SettingsPage.qml
        qml/LockPage.qml
        qml/SearchPage.qml
        qml/components/ErrorBanner.qml
)

qt6_add_resources(khaapp "translations"
    PREFIX "/translations"
    FILES
        ${QM_FILES}
)

add_custom_target(translations ALL DEPENDS ${QM_FILES})

install(TARGETS khaapp DESTINATION ${KDE_INSTALL_BINDIR})
install(FILES khaapp.notifyrc DESTINATION ${KDE_INSTALL_KNOTIFY5RCDIR})
install(FILES icons/khaapp.svg
    DESTINATION ${KDE_INSTALL_ICONDIR}/hicolor/scalable/apps)
install(FILES org.khaapp.KhaApp.desktop
    DESTINATION ${KDE_INSTALL_APPDIR})
install(FILES org.khaapp.KhaApp.appdata.xml
    DESTINATION ${KDE_INSTALL_METAINFODIR})
install(FILES ${QM_FILES} DESTINATION ${KDE_INSTALL_LOCALEDIR}/khaapp)
install(PROGRAMS ../packaging/aur/khaapp-native-launcher.sh
    DESTINATION ${KDE_INSTALL_BINDIR}
    RENAME khaapp)
