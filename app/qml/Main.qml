import QtQuick
import QtQuick.Controls
import org.kde.kirigami as Kirigami
import KhaApp 1.0
import "components"

Kirigami.ApplicationWindow {
    id: root

    width: 900
    height: 640
    visible: true
    title: qsTr("KhaApp")

    property bool daemonAvailable: true
    property string inlineMessageText: ""
    property string previousStatus: "disconnected"

    function registerActivity() {
        inactivityTimer.restart()
    }

    DBusBridge {
        id: bridge

        onLoginSuccessful: {
            root.showChatList()
        }

        onDaemonNotAvailable: {
            root.daemonAvailable = false
        }

        onOpenChatRequested: function(jid) {
            root.openChat(jid, root.findContactName(jid))
        }
    }

    Connections {
        target: bridge

        function onConnectionStatusChanged() {
            if (root.previousStatus === "connected" && bridge.connectionStatus === "disconnected") {
                root.inlineMessageText = qsTr("You were disconnected.")
                root.showLogin()
            }
            root.previousStatus = bridge.connectionStatus
        }
    }

    function showInitialPage() {
        if (appLock.lockEnabled && appLock.isLocked) {
            root.pageStack.clear()
            root.pageStack.push("qrc:/qml/LockPage.qml")
            return
        }

        const status = bridge.getStatus()
        root.previousStatus = status

        if (status === "connected") {
            showChatList()
            return
        }

        showLogin()
    }

    function showLogin() {
        root.pageStack.clear()
        root.pageStack.push("qrc:/qml/LoginPage.qml", {
            bridge: bridge
        })
        bridge.updateTrayUnread()
    }

    function showChatList() {
        root.inlineMessageText = ""
        root.pageStack.clear()
        root.pageStack.push("qrc:/qml/ChatListPage.qml", {
            bridge: bridge
        })
        bridge.updateTrayUnread()
    }

    function openChat(jid, contactName, avatarPath) {
        root.pageStack.push("qrc:/qml/ChatPage.qml", {
            bridge: bridge,
            jid: jid,
            contactName: contactName,
            avatarPath: avatarPath || ""
        })
    }

    function findContactName(jid) {
        const chats = bridge.getChats()
        for (let index = 0; index < chats.length; ++index) {
            if (chats[index].jid === jid) {
                return chats[index].name && chats[index].name.length > 0 ? chats[index].name : jid
            }
        }
        return jid
    }

    globalDrawer: null

    Component.onCompleted: Qt.callLater(showInitialPage)

    Connections {
        target: appLock

        function onIsLockedChanged() {
            if (appLock.isLocked) {
                root.pageStack.clear()
                root.pageStack.push("qrc:/qml/LockPage.qml")
            } else if (root.pageStack.currentItem && root.pageStack.currentItem.title === "Unlock") {
                root.showInitialPage()
            }
        }
    }

    Shortcut {
        sequence: "Ctrl+,"
        onActivated: root.pageStack.push("qrc:/qml/SettingsPage.qml")
    }

    Timer {
        id: inactivityTimer

        interval: 300000
        repeat: false
        onTriggered: {
            if (appLock.lockEnabled) {
                appLock.lock()
            }
        }
    }

    TapHandler {
        acceptedButtons: Qt.AllButtons
        onTapped: root.registerActivity()
    }

    ErrorBanner {
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        errorText: !root.daemonAvailable ? qsTr("KhaApp daemon is not running. Start khaapp-daemon first.") : root.inlineMessageText
        z: 100
    }
}
