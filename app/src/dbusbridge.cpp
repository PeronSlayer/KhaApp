#include "dbusbridge.h"
#include "trayicon.h"

#include <KNotification>
#include <KNotificationReplyAction>
#include <KIO/OpenUrlJob>

#include <QDBusConnection>
#include <QDBusMessage>
#include <QDBusReply>
#include <QTimer>
#include <QUrl>
#include <QVariantMap>

namespace {
constexpr auto SERVICE_NAME = "org.khaapp.Daemon";
constexpr auto OBJECT_PATH = "/org/khaapp/Daemon";
constexpr auto INTERFACE_NAME = "org.khaapp.IMessenger";
}

DBusBridge::DBusBridge(QObject *parent)
    : QObject(parent)
    , m_interface(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, QDBusConnection::sessionBus())
    , m_connectionStatus(QStringLiteral("disconnected"))
{
    connectSignals();

    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return;
    }

    QTimer::singleShot(0, this, [this]() {
        fetchStatus();
    });
}

QString DBusBridge::connectionStatus() const
{
    return m_connectionStatus;
}

void DBusBridge::requestLogin()
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return;
    }

    m_interface.call(QStringLiteral("RequestLogin"));
}

void DBusBridge::logout()
{
    if (!m_interface.isValid()) {
        return;
    }

    m_interface.call(QStringLiteral("Logout"));
}

QString DBusBridge::sendTextMessage(const QString &jid, const QString &text)
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return {};
    }

    QDBusReply<QString> reply = m_interface.call(QStringLiteral("SendTextMessage"), jid, text);
    if (!reply.isValid()) {
        return {};
    }

    return reply.value();
}

QString DBusBridge::getStatus()
{
    return fetchStatus();
}

QString DBusBridge::fetchStatus()
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return m_connectionStatus;
    }

    QDBusReply<QString> reply = m_interface.call(QStringLiteral("GetStatus"));
    if (reply.isValid()) {
        setConnectionStatus(reply.value());
    }

    return m_connectionStatus;
}

QVariantList DBusBridge::getChats()
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return {};
    }

    QDBusReply<QList<QVariantMap>> reply = m_interface.call(QStringLiteral("GetChats"));
    if (!reply.isValid()) {
        return {};
    }

    QVariantList chats;
    for (const QVariantMap &chat : reply.value()) {
        chats.append(chat);
    }

    return chats;
}

QVariantList DBusBridge::getMessages(const QString &jid, int limit, int offset)
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return {};
    }

    QDBusReply<QList<QVariantMap>> reply = m_interface.call(QStringLiteral("GetMessages"), jid, limit, offset);
    if (!reply.isValid()) {
        return {};
    }

    QVariantList messages;
    for (const QVariantMap &message : reply.value()) {
        messages.append(message);
    }

    return messages;
}

QVariantMap DBusBridge::getGroupInfo(const QString &jid)
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return {};
    }

    QDBusReply<QVariantMap> reply = m_interface.call(QStringLiteral("GetGroupInfo"), jid);
    if (!reply.isValid()) {
        return {};
    }

    return reply.value();
}

QVariantList DBusBridge::searchMessages(const QString &jid, const QString &query)
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return {};
    }

    QDBusReply<QList<QVariantMap>> reply = m_interface.call(QStringLiteral("SearchMessages"), jid, query);
    if (!reply.isValid()) {
        return {};
    }

    QVariantList messages;
    for (const QVariantMap &message : reply.value()) {
        messages.append(message);
    }

    return messages;
}

QString DBusBridge::downloadMedia(const QString &messageId, const QString &jid)
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return {};
    }

    QDBusReply<QString> reply = m_interface.call(QStringLiteral("DownloadMedia"), messageId, jid);
    if (!reply.isValid()) {
        return {};
    }

    return reply.value();
}

void DBusBridge::markAsRead(const QString &jid)
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return;
    }

    m_interface.call(QStringLiteral("MarkAsRead"), jid);
}

QVariantMap DBusBridge::fetchLinkPreview(const QString &url)
{
    if (!m_interface.isValid()) {
        emit daemonNotAvailable();
        return {};
    }

    QDBusReply<QVariantMap> reply = m_interface.call(QStringLiteral("FetchLinkPreview"), url);
    if (!reply.isValid()) {
        return {};
    }

    return reply.value();
}

void DBusBridge::openLocalFile(const QString &localPath)
{
    if (localPath.isEmpty()) {
        return;
    }

    auto *job = new KIO::OpenUrlJob(QUrl::fromLocalFile(localPath), this);
    job->start();
}

void DBusBridge::updateTrayUnread()
{
    if (m_trayIcon) {
        m_trayIcon->setUnreadCount(getTotalUnread());
    }
}

void DBusBridge::handleQRCodeUpdated(const QString &qrData)
{
    emit qrCodeUpdated(qrData);
}

void DBusBridge::handleLoginSuccessful(const QString &phoneNumber)
{
    setConnectionStatus(QStringLiteral("connected"));
    updateTrayUnread();
    emit loginSuccessful(phoneNumber);
}

void DBusBridge::handleMessageReceived(const QString &jid, const QString &senderName, const QString &text, qint64 timestampUnix)
{
    auto *notification = new KNotification(QStringLiteral("messageReceived"), KNotification::CloseOnTimeout, this);
    notification->setComponentName(QStringLiteral("khaapp"));
    notification->setTitle(senderName);
    notification->setText(text);
    notification->setIconName(QStringLiteral("dialog-messages"));

    auto *replyAction = notification->addAction(QStringLiteral("Reply"));
    connect(replyAction, &KNotificationAction::activated, this, [this, jid]() {
        emit openChatRequested(jid);
    });

    auto inlineReply = std::make_unique<KNotificationReplyAction>(QStringLiteral("Reply"));
    inlineReply->setPlaceholderText(QStringLiteral("Reply…"));
    inlineReply->setSubmitButtonText(QStringLiteral("Send"));
    inlineReply->setSubmitButtonIconName(QStringLiteral("mail-send"));
    inlineReply->setFallbackBehavior(KNotificationReplyAction::FallbackBehavior::UseRegularAction);
    connect(inlineReply.get(), &KNotificationReplyAction::replied, this, [this, jid](const QString &replyText) {
        if (!replyText.trimmed().isEmpty()) {
            sendTextMessage(jid, replyText);
        }
    });
    connect(inlineReply.get(), &KNotificationReplyAction::activated, this, [this, jid]() {
        emit openChatRequested(jid);
    });
    notification->setReplyAction(std::move(inlineReply));
    notification->sendEvent();

    updateTrayUnread();
    emit messageReceived(jid, senderName, text, timestampUnix);
}

void DBusBridge::handleTypingStarted(const QString &jid, const QString &senderJid)
{
    emit typingStarted(jid, senderJid);
}

void DBusBridge::handleTypingStopped(const QString &jid, const QString &senderJid)
{
    emit typingStopped(jid, senderJid);
}

void DBusBridge::handleMessageAcknowledged(const QString &messageId, int receiptType)
{
    emit messageAcknowledged(messageId, receiptType);
}

void DBusBridge::handleAvatarUpdated(const QString &jid, const QString &localPath)
{
    emit avatarUpdated(jid, localPath);
}

void DBusBridge::handleConnectionStatusChanged(const QString &status)
{
    setConnectionStatus(status);
    if (m_trayIcon) {
        m_trayIcon->setStatus(status);
    }
    updateTrayUnread();
}

void DBusBridge::connectSignals()
{
    QDBusConnection sessionBus = QDBusConnection::sessionBus();

    sessionBus.connect(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, "QRCodeUpdated", this, SLOT(handleQRCodeUpdated(QString)));
    sessionBus.connect(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, "LoginSuccessful", this, SLOT(handleLoginSuccessful(QString)));
    sessionBus.connect(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, "MessageReceived", this, SLOT(handleMessageReceived(QString,QString,QString,qint64)));
    sessionBus.connect(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, "TypingStarted", this, SLOT(handleTypingStarted(QString,QString)));
    sessionBus.connect(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, "TypingStopped", this, SLOT(handleTypingStopped(QString,QString)));
    sessionBus.connect(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, "MessageAcknowledged", this, SLOT(handleMessageAcknowledged(QString,int)));
    sessionBus.connect(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, "AvatarUpdated", this, SLOT(handleAvatarUpdated(QString,QString)));
    sessionBus.connect(SERVICE_NAME, OBJECT_PATH, INTERFACE_NAME, "ConnectionStatusChanged", this, SLOT(handleConnectionStatusChanged(QString)));
}

void DBusBridge::setConnectionStatus(const QString &status)
{
    if (m_connectionStatus == status) {
        return;
    }

    m_connectionStatus = status;
    emit connectionStatusChanged();
}

int DBusBridge::getTotalUnread()
{
    if (!m_interface.isValid()) {
        return 0;
    }

    QDBusReply<QList<QVariantMap>> reply = m_interface.call(QStringLiteral("GetChats"));
    if (!reply.isValid()) {
        return 0;
    }

    int total = 0;
    for (const QVariantMap &chat : reply.value()) {
        total += chat.value(QStringLiteral("unread")).toInt();
    }
    return total;
}

void DBusBridge::setTrayIcon(TrayIcon *trayIcon)
{
    m_trayIcon = trayIcon;
    if (m_trayIcon) {
        m_trayIcon->setStatus(m_connectionStatus);
        updateTrayUnread();
    }
}
