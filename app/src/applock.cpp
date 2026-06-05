#include "applock.h"

#include <KWallet>
#include <QTimer>

AppLock::AppLock(QObject *parent)
    : QObject(parent)
{
    QTimer::singleShot(0, this, &AppLock::initializeState);
}

bool AppLock::lockEnabled() const
{
    return m_lockEnabled;
}

bool AppLock::isLocked() const
{
    return m_locked;
}

void AppLock::enableLock(const QString &pin)
{
    if (pin.length() < 4) {
        return;
    }

    auto *wallet = openWallet();
    if (!wallet) {
        return;
    }

    wallet->writePassword(walletKey(), pin);
    m_lockEnabled = true;
    m_locked = true;
    emit lockEnabledChanged();
    emit isLockedChanged();
}

void AppLock::disableLock()
{
    auto *wallet = openWallet();
    if (!wallet) {
        return;
    }

    wallet->removeEntry(walletKey());
    m_lockEnabled = false;
    m_locked = false;
    emit lockEnabledChanged();
    emit isLockedChanged();
}

bool AppLock::unlock(const QString &pin)
{
    auto *wallet = openWallet();
    if (!wallet) {
        return false;
    }

    QString storedPin;
    if (wallet->readPassword(walletKey(), storedPin) != 0 || storedPin != pin) {
        return false;
    }

    if (m_locked) {
        m_locked = false;
        emit isLockedChanged();
    }
    return true;
}

void AppLock::lock()
{
    if (!m_lockEnabled || m_locked) {
        return;
    }

    m_locked = true;
    emit isLockedChanged();
}

void AppLock::initializeState()
{
    if (auto *wallet = openWallet()) {
        const bool enabled = wallet->hasEntry(walletKey());
        if (m_lockEnabled != enabled) {
            m_lockEnabled = enabled;
            emit lockEnabledChanged();
        }
        if (m_locked != enabled) {
            m_locked = enabled;
            emit isLockedChanged();
        }
    }
}

QString AppLock::walletFolder() const
{
    return QStringLiteral("khaapp");
}

QString AppLock::walletKey() const
{
    return QStringLiteral("applock_pin");
}

KWallet::Wallet *AppLock::openWallet()
{
    if (m_wallet && m_wallet->isOpen()) {
        return m_wallet;
    }

    m_wallet = KWallet::Wallet::openWallet(KWallet::Wallet::NetworkWallet(), 0, KWallet::Wallet::Synchronous);
    if (!m_wallet) {
        return nullptr;
    }

    if (!m_wallet->hasFolder(walletFolder())) {
        m_wallet->createFolder(walletFolder());
    }
    m_wallet->setFolder(walletFolder());
    return m_wallet;
}
