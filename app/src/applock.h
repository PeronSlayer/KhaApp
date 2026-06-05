#pragma once

#include <QObject>

namespace KWallet
{
class Wallet;
}

class AppLock : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool lockEnabled READ lockEnabled NOTIFY lockEnabledChanged)
    Q_PROPERTY(bool isLocked READ isLocked NOTIFY isLockedChanged)

public:
    explicit AppLock(QObject *parent = nullptr);

    bool lockEnabled() const;
    bool isLocked() const;

    Q_INVOKABLE void enableLock(const QString &pin);
    Q_INVOKABLE void disableLock();
    Q_INVOKABLE bool unlock(const QString &pin);
    Q_INVOKABLE void lock();

signals:
    void lockEnabledChanged();
    void isLockedChanged();

private:
    void initializeState();
    QString walletFolder() const;
    QString walletKey() const;
    KWallet::Wallet *openWallet();

    bool m_lockEnabled = false;
    bool m_locked = false;
    KWallet::Wallet *m_wallet = nullptr;
};
