import QtQuick
import org.kde.kirigami as Kirigami

Kirigami.InlineMessage {
    id: banner

    property string errorText: ""
    property bool retryVisible: false

    signal retryRequested()

    text: errorText
    type: Kirigami.MessageType.Error
    visible: errorText !== ""
    showCloseButton: true
    onVisibleChanged: if (!visible) errorText = ""

    actions: [
        Kirigami.Action {
            visible: banner.retryVisible
            text: qsTr("Retry")
            onTriggered: banner.retryRequested()
        }
    ]
}
