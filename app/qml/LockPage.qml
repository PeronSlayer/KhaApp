import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.kirigami as Kirigami

Kirigami.Page {
    id: root

    title: qsTr("Unlock")

    property string errorText: ""

    ColumnLayout {
        anchors.centerIn: parent
        spacing: Kirigami.Units.largeSpacing

        Kirigami.Icon {
            Layout.alignment: Qt.AlignHCenter
            source: "dialog-password"
            width: 64
            height: 64
        }

        Label {
            Layout.alignment: Qt.AlignHCenter
            text: qsTr("KhaApp is locked")
            font.pointSize: 16
        }

        TextField {
            id: pinField

            Layout.preferredWidth: 220
            echoMode: TextInput.Password
            placeholderText: qsTr("Enter PIN")
            inputMethodHints: Qt.ImhDigitsOnly
            onAccepted: unlockButton.clicked()
            activeFocusOnTab: true
            Accessible.role: Accessible.EditableText
            Accessible.name: qsTr("PIN input")
        }

        Label {
            Layout.alignment: Qt.AlignHCenter
            visible: root.errorText.length > 0
            text: root.errorText
            color: Kirigami.Theme.negativeTextColor
        }

        Button {
            id: unlockButton

            Layout.alignment: Qt.AlignHCenter
            text: qsTr("Unlock")
            onClicked: {
                if (appLock.unlock(pinField.text)) {
                    root.errorText = ""
                    applicationWindow().showInitialPage()
                } else {
                    root.errorText = qsTr("Wrong PIN")
                }
            }
            activeFocusOnTab: true
            Accessible.role: Accessible.Button
            Accessible.name: text
        }
    }

    Component.onCompleted: pinField.forceActiveFocus()
}
