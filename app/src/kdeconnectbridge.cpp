#include "kdeconnectbridge.h"

#include <QDBusInterface>
#include <QDBusReply>
#include <QDBusServiceWatcher>
#include <QUrl>
#include <QVariantMap>

namespace {
constexpr auto KDECONNECT_SERVICE = "org.kde.kdeconnect.daemon";
constexpr auto KDECONNECT_DAEMON_PATH = "/modules/kdeconnect";
constexpr auto KDECONNECT_DAEMON_INTERFACE = "org.kde.kdeconnect.daemon";
constexpr auto KDECONNECT_DEVICE_INTERFACE = "org.kde.kdeconnect.device";
constexpr auto KDECONNECT_SHARE_INTERFACE = "org.kde.kdeconnect.device.share";
}

KDEConnectBridge::KDEConnectBridge(QObject *parent)
    : QObject(parent)
    , m_watcher(new QDBusServiceWatcher(QString::fromLatin1(KDECONNECT_SERVICE),
                                        QDBusConnection::sessionBus(),
                                        QDBusServiceWatcher::WatchForRegistration | QDBusServiceWatcher::WatchForUnregistration,
                                        this))
{
    connect(m_watcher, &QDBusServiceWatcher::serviceRegistered, this, &KDEConnectBridge::refresh);
    connect(m_watcher, &QDBusServiceWatcher::serviceUnregistered, this, &KDEConnectBridge::refresh);
    refresh();
}

bool KDEConnectBridge::available() const
{
    return m_available;
}

QStringList KDEConnectBridge::connectedDevices() const
{
    QStringList names;
    for (const QVariant &device : m_devices) {
        const QVariantMap deviceMap = device.toMap();
        names << deviceMap.value(QStringLiteral("name")).toString();
    }
    return names;
}

bool KDEConnectBridge::sendFile(const QString &deviceId, const QString &filePath)
{
    if (!m_available || deviceId.isEmpty() || filePath.isEmpty()) {
        return false;
    }

    QDBusInterface shareInterface(QString::fromLatin1(KDECONNECT_SERVICE),
                                  QStringLiteral("/modules/kdeconnect/devices/%1/share").arg(deviceId),
                                  QString::fromLatin1(KDECONNECT_SHARE_INTERFACE),
                                  QDBusConnection::sessionBus());
    if (!shareInterface.isValid()) {
        return false;
    }

    QDBusReply<void> reply = shareInterface.call(QStringLiteral("shareUrl"), QUrl::fromLocalFile(filePath).toString());
    return reply.isValid();
}

QVariantList KDEConnectBridge::devices() const
{
    return m_devices;
}

void KDEConnectBridge::refresh()
{
    const bool previousAvailability = m_available;
    const QVariantList previousDevices = m_devices;

    checkAvailability();
    m_devices.clear();

    if (m_available) {
        QDBusInterface daemonInterface(QString::fromLatin1(KDECONNECT_SERVICE),
                                       QString::fromLatin1(KDECONNECT_DAEMON_PATH),
                                       QString::fromLatin1(KDECONNECT_DAEMON_INTERFACE),
                                       QDBusConnection::sessionBus());
        QDBusReply<QStringList> reply = daemonInterface.call(QStringLiteral("devices"), true, true);
        if (reply.isValid()) {
            for (const QString &deviceId : reply.value()) {
                m_devices.append(deviceInfo(deviceId));
            }
        }
    }

    if (previousAvailability != m_available) {
        emit availableChanged();
    }
    if (previousDevices != m_devices) {
        emit devicesChanged();
    }
}

void KDEConnectBridge::checkAvailability()
{
    QDBusInterface daemonInterface(QString::fromLatin1(KDECONNECT_SERVICE),
                                   QString::fromLatin1(KDECONNECT_DAEMON_PATH),
                                   QString::fromLatin1(KDECONNECT_DAEMON_INTERFACE),
                                   QDBusConnection::sessionBus());
    m_available = daemonInterface.isValid();
}

QVariantMap KDEConnectBridge::deviceInfo(const QString &deviceId) const
{
    QDBusInterface deviceInterface(QString::fromLatin1(KDECONNECT_SERVICE),
                                   QStringLiteral("/modules/kdeconnect/devices/%1").arg(deviceId),
                                   QString::fromLatin1(KDECONNECT_DEVICE_INTERFACE),
                                   QDBusConnection::sessionBus());

    QVariantMap device;
    device.insert(QStringLiteral("id"), deviceId);
    device.insert(QStringLiteral("name"), deviceInterface.property("name"));
    device.insert(QStringLiteral("reachable"), deviceInterface.property("isReachable"));
    device.insert(QStringLiteral("trusted"), deviceInterface.property("isTrusted"));
    return device;
}
