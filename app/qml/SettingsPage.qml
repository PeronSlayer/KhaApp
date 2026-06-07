import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Qt.labs.settings
import org.kde.kirigami as Kirigami

Kirigami.ScrollablePage {
    id: root

    title: qsTr("Settings")

    Settings {
        id: localSettings
        property bool desktopNotifications: true
    }

    Kirigami.FormLayout {
        width: parent.width

        Kirigami.Separator {
            Kirigami.FormData.isSection: true
            Kirigami.FormData.label: qsTr("Security")
        }

        CheckBox {
            id: appLockCheckBox
            Kirigami.FormData.label: qsTr("App lock")
            checked: appLock.lockEnabled
            onToggled: {
                if (checked) {
                    pinSetupDialog.open()
                } else {
                    appLock.disableLock()
                }
            }
            activeFocusOnTab: true
            Accessible.role: Accessible.CheckBox
            Accessible.name: Kirigami.FormData.label
            Accessible.checked: checked
        }

        Kirigami.Separator {
            Kirigami.FormData.isSection: true
            Kirigami.FormData.label: qsTr("Notifications")
        }

        CheckBox {
            id: notifToggle
            Kirigami.FormData.label: qsTr("Desktop notifications")
            checked: localSettings.desktopNotifications
            onToggled: localSettings.desktopNotifications = checked
            activeFocusOnTab: true
            Accessible.role: Accessible.CheckBox
            Accessible.name: Kirigami.FormData.label
            Accessible.checked: checked
        }

        Kirigami.Separator {
            Kirigami.FormData.isSection: true
            Kirigami.FormData.label: qsTr("KDE Connect")
        }

        Label {
            Kirigami.FormData.label: qsTr("Status")
            text: kdeConnect.available
                  ? (kdeConnect.connectedDevices.length > 0
                     ? qsTr("Phone connected: %1").arg(kdeConnect.connectedDevices.join(", "))
                     : qsTr("KDE Connect running, no phone connected"))
                  : qsTr("KDE Connect not running")
            wrapMode: Text.WordWrap
        }

        Button {
            id: refreshButton
            text: qsTr("Refresh")
            onClicked: kdeConnect.refresh()
            activeFocusOnTab: true
            Accessible.role: Accessible.Button
            Accessible.name: text
        }

        Kirigami.Separator {
            Kirigami.FormData.isSection: true
            Kirigami.FormData.label: qsTr("About")
        }

        Label {
            Kirigami.FormData.label: qsTr("Version")
            text: appVersion
        }

        Label {
            Kirigami.FormData.label: qsTr("License")
            text: qsTr("GPL-3.0")
        }

        Label {
            Kirigami.FormData.label: qsTr("Disclaimer")
            text: qsTr("Unofficial client. Not affiliated with Meta.")
            wrapMode: Text.WordWrap
            font.pixelSize: 11
            color: Kirigami.Theme.disabledTextColor
        }
    }

    Kirigami.Dialog {
        id: pinSetupDialog

        title: qsTr("Set app lock PIN")
        standardButtons: Kirigami.Dialog.Ok | Kirigami.Dialog.Cancel
        onAccepted: {
            if (pinField.text.length >= 4) {
                appLock.enableLock(pinField.text)
            } else {
                open()
            }
        }

        contentItem: TextField {
            id: pinField

            echoMode: TextInput.Password
            placeholderText: qsTr("Enter PIN (min 4 digits)")
            inputMethodHints: Qt.ImhDigitsOnly
            activeFocusOnTab: true
            Accessible.role: Accessible.EditableText
            Accessible.name: placeholderText
        }
    }

    Component.onCompleted: appLockCheckBox.forceActiveFocus(Qt.TabFocusReason)
}
