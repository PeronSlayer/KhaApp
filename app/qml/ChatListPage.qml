import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.kirigami as Kirigami
import "components"

Kirigami.Page {
    id: root

    required property var bridge

    title: qsTr("Chats")
    property bool loading: false

    ListModel {
        id: chatModel
    }

    function formatTimestamp(timestamp) {
        const value = Number(timestamp) * 1000
        const date = new Date(value)
        const now = new Date()
        if (date.getFullYear() === now.getFullYear()
                && date.getMonth() === now.getMonth()
                && date.getDate() === now.getDate()) {
            return Qt.formatTime(date, "HH:mm")
        }
        return Qt.formatDate(date, "dd/MM")
    }

    function loadChats() {
        loading = true
        const chats = bridge.getChats()
        chatModel.clear()
        if (chats.length === 0 && bridge.connectionStatus === "connected") {
            listError.errorText = qsTr("Could not load chats.")
        } else {
            listError.errorText = ""
        }
        for (let index = 0; index < chats.length; ++index) {
            chatModel.append(chats[index])
        }
        loading = false
    }

    Connections {
        target: bridge

        function onConnectionStatusChanged() {
            if (bridge.connectionStatus === "connected") {
                root.loadChats()
            }
        }

        function onMessageReceived(jid, senderName, text, timestampUnix) {
            root.loadChats()
        }

        function onAvatarUpdated(jid, localPath) {
            for (let index = 0; index < chatModel.count; ++index) {
                if (chatModel.get(index).jid === jid) {
                    chatModel.setProperty(index, "avatar_path", localPath)
                    break
                }
            }
        }
    }

    Component.onCompleted: loadChats()
    Component.onCompleted: {
        loadChats()
        chatListView.currentIndex = 0
        chatListView.forceActiveFocus(Qt.TabFocusReason)
    }

    Shortcut {
        sequence: "Ctrl+R"
        onActivated: root.loadChats()
    }

    Shortcut {
        sequence: StandardKey.Refresh
        onActivated: root.loadChats()
    }

    Connections {
        target: applicationWindow().pageStack

        function onCurrentItemChanged() {
            if (applicationWindow().pageStack.currentItem === root) {
                root.loadChats()
            }
        }
    }

    actions: [
        Kirigami.Action {
            icon.name: "view-refresh"
            text: qsTr("Refresh")
            tooltip: qsTr("Refresh")
            onTriggered: root.loadChats()
        },
        Kirigami.Action {
            icon.name: kdeConnect.available && kdeConnect.connectedDevices.length > 0 ? "smartphone" : "smartphone-disconnected"
            text: kdeConnect.available && kdeConnect.connectedDevices.length > 0 ? qsTr("Phone connected") : qsTr("Phone offline")
            enabled: false
        },
        Kirigami.Action {
            icon.name: "settings-configure"
            text: qsTr("Settings")
            onTriggered: applicationWindow().pageStack.push("qrc:/qml/SettingsPage.qml")
        },
        Kirigami.Action {
            icon.name: "system-log-out"
            text: qsTr("Logout")
            onTriggered: {
                root.bridge.logout()
                applicationWindow().showLogin()
            }
        }
    ]

    ListView {
        id: chatListView

        anchors.fill: parent
        clip: true
        model: chatModel
        keyNavigationEnabled: true
        keyNavigationWraps: false
        focus: true

        delegate: ItemDelegate {
            required property string jid
            required property string name
            required property string last_message
            required property var timestamp
            required property int unread
            required property bool is_group
            required property string avatar_path

            width: chatListView.width
            onClicked: {
                applicationWindow().openChat(jid, name.length > 0 ? name : jid, avatar_path)
            }
            Keys.onReturnPressed: clicked()
            Keys.onEnterPressed: clicked()
            activeFocusOnTab: true
            Accessible.role: Accessible.ListItem
            Accessible.name: (name.length > 0 ? name : jid) + ", " + last_message
            Accessible.description: qsTr("%1 unread messages").arg(unread)

            contentItem: RowLayout {
                spacing: Kirigami.Units.largeSpacing

                Rectangle {
                    Layout.preferredWidth: 44
                    Layout.preferredHeight: 44
                    radius: 22
                    color: Kirigami.Theme.highlightColor
                    clip: true

                    Image {
                        anchors.fill: parent
                        source: avatar_path !== "" ? ("file://" + avatar_path) : ""
                        visible: avatar_path !== ""
                        fillMode: Image.PreserveAspectCrop
                        Accessible.role: Accessible.Graphic
                        Accessible.name: qsTr("Chat avatar")
                    }

                    Text {
                        anchors.centerIn: parent
                        visible: avatar_path === ""
                        text: (name || jid || "?").charAt(0).toUpperCase()
                        color: Kirigami.Theme.highlightedTextColor
                        font.pixelSize: 18
                        font.bold: true
                    }
                }

                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: 2

                    Label {
                        text: name.length > 0 ? name : jid
                        font.bold: true
                        font.pointSize: 14
                        elide: Text.ElideRight
                    }

                    Label {
                        text: last_message
                        color: Kirigami.Theme.disabledTextColor
                        font.pointSize: 12
                        elide: Text.ElideRight
                    }
                }

                Label {
                    text: root.formatTimestamp(timestamp)
                    color: Kirigami.Theme.disabledTextColor
                    font.pointSize: 11
                    Layout.alignment: Qt.AlignTop
                }

                Rectangle {
                    visible: unread > 0
                    width: unreadLabel.implicitWidth + 10
                    height: 20
                    radius: 10
                    color: Kirigami.Theme.highlightColor
                    Layout.alignment: Qt.AlignVCenter

                    Text {
                        id: unreadLabel

                        anchors.centerIn: parent
                        text: unread > 99 ? "99+" : unread.toString()
                        font.pixelSize: 11
                        color: Kirigami.Theme.highlightedTextColor
                    }
                }
            }
        }
    }

    ErrorBanner {
        id: listError
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        retryVisible: true
        onRetryRequested: root.loadChats()
    }

    BusyIndicator {
        anchors.centerIn: parent
        visible: loading && chatModel.count === 0
        running: visible
    }

    Kirigami.PlaceholderMessage {
        anchors.centerIn: parent
        visible: chatModel.count === 0 && !loading
        icon.name: "mail-message"
        text: qsTr("No chats yet")
        explanation: qsTr("Log in and exchange messages to populate the chat list.")
    }
}
