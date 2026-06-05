#pragma once

#include <QVariantList>
#include <QVariantMap>
#include <QObject>
#include <QDBusInterface>

class DBusBridge : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString connectionStatus READ connectionStatus NOTIFY connectionStatusChanged)

public:
    explicit DBusBridge(QObject *parent = nullptr);

    QString connectionStatus() const;

    Q_INVOKABLE void requestLogin();
    Q_INVOKABLE void logout();
    Q_INVOKABLE QString sendTextMessage(const QString &jid, const QString &text);
    Q_INVOKABLE QString getStatus();
    Q_INVOKABLE QString fetchStatus();
    Q_INVOKABLE QVariantList getChats();
    Q_INVOKABLE QVariantList getMessages(const QString &jid, int limit, int offset);
    Q_INVOKABLE QVariantMap getGroupInfo(const QString &jid);
    Q_INVOKABLE QVariantList searchMessages(const QString &jid, const QString &query);
    Q_INVOKABLE QString downloadMedia(const QString &messageId, const QString &jid);
    Q_INVOKABLE void markAsRead(const QString &jid);
    Q_INVOKABLE QVariantMap fetchLinkPreview(const QString &url);
    Q_INVOKABLE void openLocalFile(const QString &localPath);
    Q_INVOKABLE void updateTrayUnread();

signals:
    void qrCodeUpdated(const QString &qrData);
    void loginSuccessful(const QString &phoneNumber);
    void messageReceived(const QString &jid, const QString &senderName, const QString &text, qint64 timestampUnix);
    void typingStarted(const QString &jid, const QString &senderJid);
    void typingStopped(const QString &jid, const QString &senderJid);
    void messageAcknowledged(const QString &messageId, int receiptType);
    void avatarUpdated(const QString &jid, const QString &localPath);
    void connectionStatusChanged();
    void daemonNotAvailable();
    void openChatRequested(const QString &jid);

private slots:
    void handleQRCodeUpdated(const QString &qrData);
    void handleLoginSuccessful(const QString &phoneNumber);
    void handleMessageReceived(const QString &jid, const QString &senderName, const QString &text, qint64 timestampUnix);
    void handleTypingStarted(const QString &jid, const QString &senderJid);
    void handleTypingStopped(const QString &jid, const QString &senderJid);
    void handleMessageAcknowledged(const QString &messageId, int receiptType);
    void handleAvatarUpdated(const QString &jid, const QString &localPath);
    void handleConnectionStatusChanged(const QString &status);

private:
    void connectSignals();
    void setConnectionStatus(const QString &status);
    int getTotalUnread();

    QDBusInterface m_interface;
    QString m_connectionStatus;
    class TrayIcon *m_trayIcon = nullptr;

public:
    void setTrayIcon(class TrayIcon *trayIcon);
};
