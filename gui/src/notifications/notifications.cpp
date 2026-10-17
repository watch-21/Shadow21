#include "notifications/notifications.h"

#include <QDBusMessage>
#include "application/workspace.h"
#include <QDBusPendingCallWatcher>
#include <QDBusPendingReply>
#include <QDebug>
#include <QVariantMap>

namespace {
constexpr auto Service = "org.freedesktop.Notifications";
constexpr auto Path = "/org/freedesktop/Notifications";
} // namespace

DesktopNotifications::DesktopNotifications(const QDBusConnection &bus,
                                           QObject *parent)
    : QObject(parent), m_bus(bus) {
  const bool actions = m_bus.connect(Service, Path, Service, "ActionInvoked",
                                     this, SLOT(actionInvoked(uint, QString)));
  const bool closed =
      m_bus.connect(Service, Path, Service, "NotificationClosed", this,
                    SLOT(notificationClosed(uint, uint)));
  if (!actions || !closed)
    reportFailure("Cannot subscribe to desktop notification events.");
}

void DesktopNotifications::send(const QString &title, const QString &message,
                                bool error) {
  if (!m_bus.isConnected()) {
    reportFailure("The session bus is unavailable.");
    return;
  }
  auto request =
      QDBusMessage::createMethodCall(Service, Path, Service, "Notify");
  const QVariantMap hints{
      {"desktop-entry", "org.shadow21.Desktop"},
      {"urgency", QVariant::fromValue(static_cast<uchar>(error ? 2 : 1))}};
  request.setArguments({QString("shadow21"), uint(0), QString("shadow21"),
                        WText(title), WText(message).toHtmlEscaped(),
                        QStringList{"default", WText("Open shadow21")}, hints, -1});
  auto *watcher =
      new QDBusPendingCallWatcher(m_bus.asyncCall(request, 5000), this);
  connect(watcher, &QDBusPendingCallWatcher::finished, this,
          [this](QDBusPendingCallWatcher *call) {
            const QDBusPendingReply<uint> reply = *call;
            call->deleteLater();
            if (reply.isError()) {
              reportFailure(reply.error().message());
              return;
            }
            m_failed = false;
            m_pending.insert(reply.value());
          });
}

void DesktopNotifications::actionInvoked(uint id, const QString &action) {
  if (m_pending.contains(id) && action == "default")
    emit activated();
}

void DesktopNotifications::notificationClosed(uint id, uint) {
  m_pending.remove(id);
}

void DesktopNotifications::reportFailure(const QString &message) {
  if (!m_failed)
    qWarning().noquote() << "Desktop notifications are unavailable:" << message;
  m_failed = true;
}
