#pragma once

#include <QObject>

class KStatusNotifierItem;

class TrayIcon : public QObject
{
    Q_OBJECT

public:
    explicit TrayIcon(QObject *parent = nullptr);

    void setUnreadCount(int count);
    void setStatus(const QString &status);

signals:
    void showWindowRequested();
    void quitRequested();

private:
    KStatusNotifierItem *m_item;
};
