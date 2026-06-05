#include "trayicon.h"

#include <KStatusNotifierItem>

#include <QAction>
#include <QIcon>
#include <QMenu>
#include <QPainter>
#include <QPixmap>

TrayIcon::TrayIcon(QObject *parent)
    : QObject(parent)
    , m_item(new KStatusNotifierItem(QStringLiteral("khaapp"), this))
{
    m_item->setCategory(KStatusNotifierItem::Communications);
    m_item->setIconByName(QStringLiteral("dialog-messages"));
    m_item->setToolTipTitle(QStringLiteral("KhaApp"));
    m_item->setToolTipSubTitle(QStringLiteral("Disconnected"));
    m_item->setStatus(KStatusNotifierItem::Passive);

    auto *menu = m_item->contextMenu();
    auto *showAction = menu->addAction(QStringLiteral("Show KhaApp"));
    connect(showAction, &QAction::triggered, this, &TrayIcon::showWindowRequested);

    menu->addSeparator();

    auto *quitAction = menu->addAction(QStringLiteral("Quit"));
    connect(quitAction, &QAction::triggered, this, &TrayIcon::quitRequested);

    connect(m_item, &KStatusNotifierItem::activateRequested, this, [this](bool, const QPoint &) {
        emit showWindowRequested();
    });
}

void TrayIcon::setUnreadCount(int count)
{
    if (count > 0) {
        QPixmap pixmap(32, 32);
        pixmap.fill(Qt::transparent);

        QPainter painter(&pixmap);
        painter.setRenderHint(QPainter::Antialiasing);
        painter.setBrush(QColor(QStringLiteral("#d32f2f")));
        painter.setPen(Qt::NoPen);
        painter.drawEllipse(0, 0, 32, 32);

        painter.setPen(Qt::white);
        QFont font = painter.font();
        font.setBold(true);
        font.setPixelSize(count > 99 ? 12 : 14);
        painter.setFont(font);
        painter.drawText(pixmap.rect(), Qt::AlignCenter, count > 99 ? QStringLiteral("99+") : QString::number(count));

        m_item->setOverlayIconByPixmap(QIcon(pixmap));
        m_item->setToolTipSubTitle(QStringLiteral("%1 unread messages").arg(count));
        m_item->setStatus(KStatusNotifierItem::Active);
        return;
    }

    m_item->setOverlayIconByPixmap(QIcon());
    m_item->setToolTipSubTitle(QStringLiteral("No unread messages"));
    m_item->setStatus(KStatusNotifierItem::Passive);
}

void TrayIcon::setStatus(const QString &status)
{
    if (status == QStringLiteral("connected")) {
        m_item->setIconByName(QStringLiteral("dialog-messages"));
        return;
    }

    m_item->setIconByName(QStringLiteral("network-offline"));
}
