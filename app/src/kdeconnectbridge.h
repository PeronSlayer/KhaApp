#pragma once

#include <QObject>
#include <QStringList>
#include <QVariantList>

class QDBusServiceWatcher;

class KDEConnectBridge : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool available READ available NOTIFY availableChanged)
    Q_PROPERTY(QStringList connectedDevices READ connectedDevices NOTIFY devicesChanged)

public:
    explicit KDEConnectBridge(QObject *parent = nullptr);

    bool available() const;
    QStringList connectedDevices() const;

    Q_INVOKABLE bool sendFile(const QString &deviceId, const QString &filePath);
    Q_INVOKABLE QVariantList devices() const;

public slots:
    void refresh();

signals:
    void availableChanged();
    void devicesChanged();

private:
    void checkAvailability();
    QVariantMap deviceInfo(const QString &deviceId) const;

    bool m_available = false;
    QVariantList m_devices;
    QDBusServiceWatcher *m_watcher = nullptr;
};
