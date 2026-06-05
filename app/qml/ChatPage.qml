import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtMultimedia
import org.kde.kirigami as Kirigami
import "components"

Kirigami.Page {
    id: root

    required property var bridge
    property string jid: ""
    property string contactName: ""
    property string avatarPath: ""
    property bool isGroup: jid.endsWith("@g.us")
    property int groupMemberCount: 0
    property bool loadingMessages: false
    property int loadedCount: 50
    property bool loadingMore: false
    property string typingText: ""

    title: contactName

    actions: [
        Kirigami.Action {
            icon.name: "search"
            text: qsTr("Search")
            onTriggered: applicationWindow().pageStack.push("qrc:/qml/SearchPage.qml", {
                bridge: root.bridge,
                jid: root.jid
            })
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

    ListModel {
        id: messageModel
    }

    Timer {
        id: typingTimer
        interval: 5000
        onTriggered: root.typingText = ""
    }

    function sameDay(firstTimestamp, secondTimestamp) {
        const first = new Date(Number(firstTimestamp) * 1000)
        const second = new Date(Number(secondTimestamp) * 1000)
        return first.getFullYear() === second.getFullYear()
                && first.getMonth() === second.getMonth()
                && first.getDate() === second.getDate()
    }

    function shouldShowDateSeparator(index, timestamp) {
        if (index === 0) {
            return true
        }
        return !sameDay(timestamp, messageModel.get(index - 1).timestamp)
    }

    function formatDateLabel(timestamp) {
        return Qt.formatDate(new Date(Number(timestamp) * 1000), "dddd, dd MMMM")
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

    function reloadMessages() {
        root.loadingMessages = true
        root.loadedCount = 50
        const messages = bridge.getMessages(root.jid, root.loadedCount, 0)
        messageModel.clear()
        for (let index = messages.length - 1; index >= 0; --index) {
            messageModel.append(messages[index])
        }
        Qt.callLater(function() {
            if (messageModel.count > 0) {
                listView.positionViewAtEnd()
            }
            root.loadingMessages = false
            root.loadingMore = false
        })
    }

    function loadMoreMessages() {
        const older = bridge.getMessages(root.jid, 25, root.loadedCount)
        if (older.length === 0) {
            root.loadingMore = false
            return
        }

        for (let index = older.length - 1; index >= 0; --index) {
            messageModel.insert(0, older[index])
        }
        root.loadedCount += older.length
        root.loadingMore = false
    }

    function refreshGroupInfo() {
        if (!root.isGroup) {
            root.groupMemberCount = 0
            return
        }
        const info = bridge.getGroupInfo(root.jid)
        root.groupMemberCount = info.participant_count || 0
        if ((root.contactName === "" || root.contactName === root.jid) && info.name) {
            root.contactName = info.name
        }
    }

    function sendCurrentMessage() {
        const outgoingText = textField.text.trim()
        if (outgoingText.length === 0) {
            return
        }
        bridge.sendTextMessage(root.jid, outgoingText)
        bridge.markAsRead(root.jid)
        textField.clear()
        root.reloadMessages()
    }

    function formatDuration(ms) {
        const seconds = Math.floor(Number(ms) / 1000)
        return Math.floor(seconds / 60) + ":" + String(seconds % 60).padStart(2, "0")
    }

    Connections {
        target: bridge

        function onMessageReceived(incomingJid, senderName, text, timestampUnix) {
            if (incomingJid === root.jid) {
                bridge.markAsRead(root.jid)
                root.reloadMessages()
            }
        }

        function onMessageAcknowledged(messageId, receiptType) {
            for (let index = 0; index < messageModel.count; ++index) {
                if (messageModel.get(index).id === messageId) {
                    messageModel.setProperty(index, "receipt", receiptType)
                    break
                }
            }
        }

        function onTypingStarted(incomingJid, senderJid) {
            if (incomingJid !== root.jid) {
                return
            }
            root.typingText = root.isGroup ? (senderJid + " is typing...") : "typing..."
            typingTimer.restart()
        }

        function onTypingStopped(incomingJid, senderJid) {
            if (incomingJid === root.jid) {
                root.typingText = ""
            }
        }

        function onAvatarUpdated(updatedJid, localPath) {
            if (updatedJid === root.jid) {
                root.avatarPath = localPath
            }
        }
    }

    Component.onCompleted: {
        refreshGroupInfo()
        bridge.markAsRead(root.jid)
        reloadMessages()
    }

    onVisibleChanged: {
        if (visible) {
            bridge.markAsRead(root.jid)
            refreshGroupInfo()
            reloadMessages()
        }
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        Rectangle {
            Layout.fillWidth: true
            color: Kirigami.Theme.alternateBackgroundColor
            implicitHeight: headerRow.implicitHeight + Kirigami.Units.largeSpacing

            RowLayout {
                id: headerRow
                anchors.fill: parent
                anchors.margins: Kirigami.Units.largeSpacing
                spacing: Kirigami.Units.largeSpacing

                Rectangle {
                    Layout.preferredWidth: 32
                    Layout.preferredHeight: 32
                    radius: 16
                    color: Kirigami.Theme.highlightColor
                    clip: true

                    Image {
                        anchors.fill: parent
                        source: root.avatarPath !== "" ? ("file://" + root.avatarPath) : ""
                        visible: root.avatarPath !== ""
                        fillMode: Image.PreserveAspectCrop
                    }

                    Text {
                        anchors.centerIn: parent
                        visible: root.avatarPath === ""
                        text: (root.contactName || root.jid || "?").charAt(0).toUpperCase()
                        color: Kirigami.Theme.highlightedTextColor
                        font.pixelSize: 14
                        font.bold: true
                    }
                }

                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: 2

                    Label {
                        text: root.contactName
                        font.bold: true
                        font.pointSize: 14
                    }

                    Label {
                        visible: root.isGroup
                        text: root.groupMemberCount > 0 ? (root.groupMemberCount + " members") : "Group chat"
                        color: Kirigami.Theme.disabledTextColor
                        font.pointSize: 11
                    }
                }
            }
        }

        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            ListView {
                id: listView
                anchors.fill: parent
                clip: true
                boundsBehavior: Flickable.DragAndOvershootBounds
                spacing: Kirigami.Units.smallSpacing
                model: messageModel
                ScrollBar.vertical: ScrollBar {
                    policy: ScrollBar.AsNeeded
                }

                onAtYBeginningChanged: {
                    if (atYBeginning && !root.loadingMore && !root.loadingMessages) {
                        root.loadingMore = true
                        root.loadMoreMessages()
                    }
                }

                delegate: Item {
                    id: bubbleDelegate

                    required property string id
                    required property string text
                    required property bool from_me
                    required property var timestamp
                    required property string sender_name
                    required property string sender_jid
                    required property int receipt
                    required property bool has_media
                    required property string media_type
                    required property string media_mime
                    required property var media_size
                    required property string local_path
                    required property string url_preview
                    property string resolvedLocalPath: local_path

                    width: listView.width
                    height: bubbleColumn.implicitHeight + Kirigami.Units.smallSpacing

                    Column {
                        id: bubbleColumn
                        anchors.right: from_me ? parent.right : undefined
                        anchors.left: from_me ? undefined : parent.left
                        anchors.margins: Kirigami.Units.largeSpacing
                        spacing: 4
                        width: Math.min(parent.width * 0.78, bubbleText.implicitWidth + Kirigami.Units.largeSpacing * 2)

                        Label {
                            anchors.horizontalCenter: parent.horizontalCenter
                            visible: root.shouldShowDateSeparator(index, timestamp)
                            text: root.formatDateLabel(timestamp)
                            color: Kirigami.Theme.disabledTextColor
                            font.pixelSize: 11
                        }

                        Rectangle {
                            width: parent.width
                            radius: 12
                            color: from_me ? Kirigami.Theme.highlightColor : Kirigami.Theme.alternateBackgroundColor
                            implicitHeight: bubbleContent.implicitHeight + Kirigami.Units.largeSpacing

                            ColumnLayout {
                                id: bubbleContent
                                anchors.fill: parent
                                anchors.margins: Kirigami.Units.smallSpacing
                                spacing: Kirigami.Units.smallSpacing

                                Text {
                                    visible: root.isGroup && !from_me && (sender_name !== "" || sender_jid !== "")
                                    text: sender_name !== "" ? sender_name : sender_jid
                                    color: Kirigami.Theme.highlightColor
                                    font.pixelSize: 11
                                    font.bold: true
                                }

                                Text {
                                    id: bubbleText
                                    Layout.fillWidth: true
                                    color: from_me ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                                    wrapMode: Text.Wrap
                                    text: bubbleDelegate.text
                                    visible: text.length > 0
                                }

                                Loader {
                                    Layout.fillWidth: true
                                    active: bubbleDelegate.has_media && bubbleDelegate.media_type === "image"
                                    sourceComponent: Column {
                                        spacing: Kirigami.Units.smallSpacing

                                        Image {
                                            width: 200
                                            height: 150
                                            fillMode: Image.PreserveAspectCrop
                                            source: bubbleDelegate.resolvedLocalPath !== "" ? ("file://" + bubbleDelegate.resolvedLocalPath) : ""
                                            visible: bubbleDelegate.resolvedLocalPath !== ""

                                            MouseArea {
                                                anchors.fill: parent
                                                enabled: bubbleDelegate.resolvedLocalPath !== ""
                                                onClicked: bridge.openLocalFile(bubbleDelegate.resolvedLocalPath)
                                            }
                                        }

                                        Button {
                                            visible: bubbleDelegate.resolvedLocalPath === ""
                                            text: qsTr("Download image")
                                            onClicked: {
                                                const path = bridge.downloadMedia(bubbleDelegate.id, root.jid)
                                                if (path !== "") {
                                                    bubbleDelegate.resolvedLocalPath = path
                                                } else {
                                                    errorBanner.errorText = qsTr("Media download failed.")
                                                }
                                            }
                                        }
                                    }
                                }

                                Loader {
                                    Layout.fillWidth: true
                                    active: bubbleDelegate.has_media && bubbleDelegate.media_type === "audio"
                                    sourceComponent: RowLayout {
                                        MediaPlayer {
                                            id: player
                                            audioOutput: AudioOutput {}
                                            source: bubbleDelegate.resolvedLocalPath !== "" ? ("file://" + bubbleDelegate.resolvedLocalPath) : ""
                                        }

                                        Button {
                                            icon.name: player.playbackState === MediaPlayer.PlayingState ? "media-playback-pause" : "media-playback-start"
                                            onClicked: {
                                                if (bubbleDelegate.resolvedLocalPath === "") {
                                                    const path = bridge.downloadMedia(bubbleDelegate.id, root.jid)
                                                    if (path !== "") {
                                                        bubbleDelegate.resolvedLocalPath = path
                                                        player.source = "file://" + path
                                                    } else {
                                                        errorBanner.errorText = qsTr("Media download failed.")
                                                    }
                                                }
                                                if (player.playbackState === MediaPlayer.PlayingState) {
                                                    player.pause()
                                                } else {
                                                    player.play()
                                                }
                                            }
                                        }

                                        Slider {
                                            from: 0
                                            to: player.duration
                                            value: player.position
                                            onMoved: player.position = value
                                            Layout.fillWidth: true
                                        }

                                        Text {
                                            text: root.formatDuration(player.duration)
                                            color: Kirigami.Theme.disabledTextColor
                                            font.pixelSize: 11
                                        }
                                    }
                                }

                                Loader {
                                    Layout.fillWidth: true
                                    active: bubbleDelegate.has_media && bubbleDelegate.media_type === "document"
                                    sourceComponent: Item {
                                        implicitWidth: documentRow.implicitWidth
                                        implicitHeight: documentRow.implicitHeight

                                        RowLayout {
                                            id: documentRow
                                            spacing: Kirigami.Units.smallSpacing

                                            Kirigami.Icon {
                                                source: "document-open"
                                                width: 24
                                                height: 24
                                            }

                                            Column {
                                                spacing: 2

                                                Text {
                                                    text: bubbleDelegate.media_mime !== "" ? bubbleDelegate.media_mime : qsTr("Document")
                                                    font.pixelSize: 12
                                                    color: Kirigami.Theme.textColor
                                                }

                                                Text {
                                                    text: bubbleDelegate.resolvedLocalPath !== "" ? qsTr("Tap to open") : qsTr("Tap to download")
                                                    font.pixelSize: 11
                                                    color: Kirigami.Theme.disabledTextColor
                                                }
                                            }
                                        }

                                        MouseArea {
                                            anchors.fill: parent
                                            onClicked: {
                                                if (bubbleDelegate.resolvedLocalPath === "") {
                                                    const path = bridge.downloadMedia(bubbleDelegate.id, root.jid)
                                                    if (path !== "") {
                                                        bubbleDelegate.resolvedLocalPath = path
                                                    } else {
                                                        errorBanner.errorText = qsTr("Media download failed.")
                                                    }
                                                }
                                                if (bubbleDelegate.resolvedLocalPath !== "") {
                                                    bridge.openLocalFile(bubbleDelegate.resolvedLocalPath)
                                                }
                                            }
                                        }
                                    }
                                }

                                Loader {
                                    Layout.fillWidth: true
                                    active: bubbleDelegate.url_preview !== "" && !bubbleDelegate.has_media
                                    sourceComponent: LinkPreviewCard {
                                        bridge: root.bridge
                                        url: bubbleDelegate.url_preview
                                    }
                                }
                            }
                        }

                        RowLayout {
                            width: parent.width
                            layoutDirection: from_me ? Qt.RightToLeft : Qt.LeftToRight
                            spacing: 4

                            Label {
                                text: root.formatTimestamp(timestamp)
                                color: Kirigami.Theme.disabledTextColor
                            }

                            Kirigami.Icon {
                                visible: from_me
                                source: {
                                    if (receipt === 2) return "mail-read"
                                    if (receipt === 1) return "mail-mark-read"
                                    return "mail-sent"
                                }
                                width: 14
                                height: 14
                                color: receipt === 2 ? Kirigami.Theme.highlightColor : Kirigami.Theme.disabledTextColor
                            }
                        }
                    }
                }
            }

            ErrorBanner {
                id: errorBanner
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                retryVisible: true
                onRetryRequested: root.reloadMessages()
            }

            BusyIndicator {
                anchors.centerIn: parent
                running: root.loadingMessages
                visible: running
            }

            Kirigami.PlaceholderMessage {
                anchors.centerIn: parent
                visible: !root.loadingMessages && messageModel.count === 0
                icon.name: "mail-message"
                text: qsTr("No messages yet")
            }

            RoundButton {
                anchors.right: parent.right
                anchors.bottom: parent.bottom
                anchors.margins: Kirigami.Units.largeSpacing
                visible: !listView.atYEnd
                icon.name: "go-bottom"
                onClicked: listView.positionViewAtEnd()
            }
        }

    }

    footer: ToolBar {
        ColumnLayout {
            anchors.fill: parent
            anchors.margins: Kirigami.Units.smallSpacing
            spacing: Kirigami.Units.smallSpacing

            Label {
                Layout.fillWidth: true
                visible: root.typingText !== ""
                text: root.typingText
                color: Kirigami.Theme.disabledTextColor
                font.italic: true
            }

            RowLayout {
                Layout.fillWidth: true
                spacing: Kirigami.Units.smallSpacing

                TextArea {
                    id: textField
                    Layout.fillWidth: true
                    placeholderText: qsTr("Message...")
                    wrapMode: TextArea.Wrap
                    maximumLength: 4096
                    activeFocusOnTab: true
                    Accessible.role: Accessible.EditableText
                    Accessible.name: qsTr("Message input")
                    Keys.onReturnPressed: function(event) {
                        if (!(event.modifiers & Qt.ShiftModifier)) {
                            event.accepted = true
                            root.sendCurrentMessage()
                        }
                    }
                }

                ToolButton {
                    icon.name: "mail-send"
                    enabled: textField.text.trim().length > 0
                    onClicked: root.sendCurrentMessage()
                    activeFocusOnTab: true
                    Accessible.role: Accessible.Button
                    Accessible.name: qsTr("Send message")
                }
            }
        }
    }

    Shortcut {
        sequence: StandardKey.Back
        onActivated: applicationWindow().pageStack.pop()
    }

    Shortcut {
        sequence: "Escape"
        onActivated: applicationWindow().pageStack.pop()
    }

    Shortcut {
        sequence: StandardKey.MoveToEndOfDocument
        onActivated: listView.positionViewAtEnd()
    }
}
