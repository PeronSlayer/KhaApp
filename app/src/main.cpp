#include "dbusbridge.h"
#include "trayicon.h"
#include "applock.h"
#include "kdeconnectbridge.h"

#include <QApplication>
#include <QDebug>
#include <QElapsedTimer>
#include <QEvent>
#include <QKeySequence>
#include <QLocale>
#include <QTranslator>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQuickWindow>
#include <QSettings>
#include <QShortcut>

namespace {
class WindowEventFilter : public QObject
{
public:
    explicit WindowEventFilter(QObject *parent = nullptr)
        : QObject(parent)
    {
    }

    void setHideToTray(bool enabled)
    {
        m_hideToTray = enabled;
    }

protected:
    bool eventFilter(QObject *watched, QEvent *event) override
    {
        if (m_hideToTray && event->type() == QEvent::Close) {
            if (auto *window = qobject_cast<QWindow *>(watched)) {
                window->hide();
                event->ignore();
                return true;
            }
        }

        return QObject::eventFilter(watched, event);
    }

private:
    bool m_hideToTray = true;
};
}

int main(int argc, char *argv[])
{
    QApplication app(argc, argv);
    QApplication::setApplicationName(QStringLiteral("khaapp"));
    QApplication::setApplicationDisplayName(QStringLiteral("KhaApp"));

    QElapsedTimer startupTimer;
#ifdef QT_DEBUG
    startupTimer.start();
#endif

    QTranslator translator;
    const QString locale = QLocale::system().name();
    if (!translator.load(QStringLiteral("khaapp_") + locale, QStringLiteral(":/translations"))
        && !translator.load(QStringLiteral("khaapp_") + locale.section(QLatin1Char('_'), 0, 0), QStringLiteral(":/translations"))) {
        (void)translator.load(QStringLiteral("khaapp_en"), QStringLiteral(":/translations"));
    }
    app.installTranslator(&translator);

    qmlRegisterType<DBusBridge>("KhaApp", 1, 0, "DBusBridge");

    TrayIcon tray;
    AppLock appLock;
    KDEConnectBridge kdeConnect;

    QQmlApplicationEngine engine;
    const QUrl url(QStringLiteral("qrc:/qml/Main.qml"));
    engine.rootContext()->setContextProperty(QStringLiteral("appLock"), &appLock);
    engine.rootContext()->setContextProperty(QStringLiteral("kdeConnect"), &kdeConnect);
    engine.rootContext()->setContextProperty(QStringLiteral("appVersion"), QStringLiteral(KHAAPP_VERSION));

    QObject::connect(
        &engine,
        &QQmlApplicationEngine::objectCreationFailed,
        &app,
        []() { QCoreApplication::exit(EXIT_FAILURE); },
        Qt::QueuedConnection);
    QObject::connect(
        &engine,
        &QQmlApplicationEngine::objectCreated,
        &app,
        [&startupTimer](QObject *object, const QUrl &) {
#ifdef QT_DEBUG
            if (object) {
                qDebug() << "Startup time:" << startupTimer.elapsed() << "ms";
            }
#else
            Q_UNUSED(startupTimer);
            Q_UNUSED(object);
#endif
        },
        Qt::SingleShotConnection);

    engine.load(url);
    if (engine.rootObjects().isEmpty()) {
        return EXIT_FAILURE;
    }

    auto *window = qobject_cast<QQuickWindow *>(engine.rootObjects().first());
    if (!window) {
        return EXIT_FAILURE;
    }

    auto *bridge = window->findChild<DBusBridge *>(QString(), Qt::FindChildrenRecursively);
    if (bridge) {
        bridge->setTrayIcon(&tray);
    }

    QSettings settings(QStringLiteral("khaapp"), QStringLiteral("khaapp"));
    if (settings.contains(QStringLiteral("windowGeometry"))) {
        window->setGeometry(settings.value(QStringLiteral("windowGeometry")).toRect());
    }
    if (settings.value(QStringLiteral("windowMaximized"), false).toBool()) {
        window->showMaximized();
    }

    auto *windowEventFilter = new WindowEventFilter(window);
    window->installEventFilter(windowEventFilter);

    auto *quitShortcut = new QShortcut(QKeySequence::Quit, window);
    QObject::connect(quitShortcut, &QShortcut::activated, &app, &QCoreApplication::quit);

    QObject::connect(&tray, &TrayIcon::showWindowRequested, window, [window]() {
        window->show();
        window->raise();
        window->requestActivate();
    });
    QObject::connect(&tray, &TrayIcon::quitRequested, &app, [&app, windowEventFilter]() {
        windowEventFilter->setHideToTray(false);
        app.quit();
    });

    QObject::connect(&app, &QGuiApplication::aboutToQuit, window, [window]() {
        QSettings settings(QStringLiteral("khaapp"), QStringLiteral("khaapp"));
        settings.setValue(QStringLiteral("windowGeometry"), window->geometry());
        settings.setValue(QStringLiteral("windowMaximized"), window->visibility() == QWindow::Maximized);
    });

    return app.exec();
}
