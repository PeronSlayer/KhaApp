import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.kirigami as Kirigami

Kirigami.Page {
    id: root

    required property var bridge

    title: qsTr("Sign in")

    property string qrData: ""

    Connections {
        target: root.bridge

        function onQrCodeUpdated(data) {
            qrImage.source = "data:image/png;base64," + data
            root.qrData = data
        }
    }

    ColumnLayout {
        anchors.centerIn: parent
        spacing: Kirigami.Units.largeSpacing

        Image {
            id: qrImage

            Layout.alignment: Qt.AlignHCenter
            width: 220
            height: 220
            fillMode: Image.PreserveAspectFit
            source: ""
            visible: root.qrData !== ""
            Accessible.role: Accessible.Graphic
            Accessible.name: qsTr("QR code for WhatsApp login")
        }

        BusyIndicator {
            Layout.alignment: Qt.AlignHCenter
            running: root.bridge.connectionStatus === "connecting"
            visible: running
        }

        Label {
            Layout.alignment: Qt.AlignHCenter
            text: {
                if (root.bridge.connectionStatus === "connecting") {
                    return qsTr("Connecting...")
                }
                if (root.bridge.connectionStatus === "qr_pending" && root.qrData.length === 0) {
                    return qsTr("Waiting for QR...")
                }
                if (root.qrData.length > 0) {
                    return qsTr("Scan with WhatsApp on your phone")
                }
                return qsTr("Press Request QR to start login")
            }
            horizontalAlignment: Text.AlignHCenter
        }

        Button {
            id: requestQRButton
            Layout.alignment: Qt.AlignHCenter
            text: qsTr("Request QR")
            onClicked: root.bridge.requestLogin()
            activeFocusOnTab: true
            Accessible.role: Accessible.Button
            Accessible.name: text
            Accessible.description: qsTr("Generate a QR code to link your WhatsApp account")
        }
    }

    Component.onCompleted: requestQRButton.forceActiveFocus()
}
