import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.kirigami as Kirigami

Item {
    id: root

    required property var bridge
    property string url: ""
    property var previewData: ({})
    property bool loaded: false

    width: parent ? parent.width : implicitWidth
    implicitHeight: loaded ? previewColumn.implicitHeight + Kirigami.Units.largeSpacing : 0
    visible: loaded

    Component.onCompleted: {
        const data = bridge.fetchLinkPreview(url)
        if (data && ((data.title || "") !== "" || (data.image_url || "") !== "")) {
            previewData = data
            loaded = true
        }
    }

    Rectangle {
        anchors.fill: parent
        color: Kirigami.Theme.alternateBackgroundColor
        radius: 6
        border.color: Kirigami.Theme.disabledTextColor
        border.width: 1
    }

    Column {
        id: previewColumn

        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.margins: Kirigami.Units.smallSpacing
        spacing: Kirigami.Units.smallSpacing

        Image {
            width: parent.width
            height: 80
            fillMode: Image.PreserveAspectCrop
            source: root.previewData.image_url || ""
            visible: source !== ""
            Accessible.role: Accessible.Graphic
            Accessible.name: qsTr("Link preview image")
        }

        Text {
            text: root.previewData.title || ""
            font.pixelSize: 12
            font.bold: true
            color: Kirigami.Theme.textColor
            wrapMode: Text.WordWrap
            width: parent.width
            visible: text !== ""
        }

        Text {
            text: root.previewData.description || ""
            font.pixelSize: 11
            color: Kirigami.Theme.disabledTextColor
            wrapMode: Text.WordWrap
            width: parent.width
            maximumLineCount: 2
            visible: text !== ""
        }
    }

    MouseArea {
        anchors.fill: parent
        onClicked: Qt.openUrlExternally(root.url)
        Accessible.role: Accessible.Button
        Accessible.name: qsTr("Open link preview")
    }
}
