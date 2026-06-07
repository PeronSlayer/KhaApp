import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.kirigami as Kirigami

Kirigami.Page {
    id: root

    required property var bridge
    property string jid: ""

    title: qsTr("Search")

    ListModel {
        id: resultsModel
    }

    function formatTimestamp(timestamp) {
        return Qt.formatDateTime(new Date(Number(timestamp) * 1000), "dd/MM HH:mm")
    }

    function performSearch(query) {
        resultsModel.clear()
        const results = bridge.searchMessages(root.jid, query)
        for (let index = 0; index < results.length; ++index) {
            resultsModel.append(results[index])
        }
    }

    ColumnLayout {
        anchors.fill: parent

        TextField {
            id: searchField
            Layout.fillWidth: true
            placeholderText: qsTr("Search messages...")
            onTextChanged: {
                if (text.length >= 2) {
                    root.performSearch(text)
                } else {
                    resultsModel.clear()
                }
            }
            activeFocusOnTab: true
            Accessible.role: Accessible.EditableText
            Accessible.name: qsTr("Search messages")
        }

        ListView {
            id: resultsList
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            model: resultsModel
            keyNavigationEnabled: true
            keyNavigationWraps: false
            focus: true

            delegate: ItemDelegate {
                required property string text
                required property var timestamp

                width: resultsList.width
                onClicked: applicationWindow().pageStack.pop()
                Keys.onReturnPressed: clicked()
                Keys.onEnterPressed: clicked()
                activeFocusOnTab: true
                Accessible.role: Accessible.ListItem
                Accessible.name: text

                contentItem: ColumnLayout {
                    spacing: 2

                    Label {
                        Layout.fillWidth: true
                        text: parent.parent.text
                        wrapMode: Text.Wrap
                    }

                    Label {
                        text: root.formatTimestamp(parent.parent.timestamp)
                        color: Kirigami.Theme.disabledTextColor
                        font.pointSize: 10
                    }
                }
            }
        }

        Kirigami.PlaceholderMessage {
            anchors.centerIn: parent
            visible: resultsModel.count === 0 && searchField.text.length >= 2
            text: qsTr("No results")
        }
    }

    Component.onCompleted: searchField.forceActiveFocus(Qt.TabFocusReason)
}
