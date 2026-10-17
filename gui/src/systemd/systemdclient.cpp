#include "systemd/systemdclient.h"

#include <QDBusMessage>
#include <QDBusPendingCallWatcher>
#include <QDBusVariant>

#include <limits>

namespace {
constexpr auto Service = "org.freedesktop.systemd1";
constexpr auto ManagerPath = "/org/freedesktop/systemd1";
constexpr auto ManagerInterface = "org.freedesktop.systemd1.Manager";
constexpr auto PropertiesInterface = "org.freedesktop.DBus.Properties";
constexpr auto AnonSurfUnit = "shadow21d.service";
constexpr auto AnonSurfPath =
    "/org/freedesktop/systemd1/unit/shadow21d_2eservice";
constexpr auto ServiceInterface = "org.freedesktop.systemd1.Service";
constexpr int AuthorizationTimeout = std::numeric_limits<int>::max();

QDBusMessage managerCall(const QString &method, const QVariantList &arguments) {
  QDBusMessage message = QDBusMessage::createMethodCall(
      Service, ManagerPath, ManagerInterface, method);
  message.setArguments(arguments);
  message.setInteractiveAuthorizationAllowed(true);
  return message;
}
} // namespace

SystemdClient::SystemdClient(QObject *parent)
    : QObject(parent), m_bus(QDBusConnection::systemBus()) {
  const bool signalConnected = m_bus.connect(
      Service, ManagerPath, ManagerInterface, "JobRemoved", this,
      SLOT(handleJobRemoved(uint, QDBusObjectPath, QString, QString)));
  const QDBusMessage reply =
      m_bus.call(managerCall("Subscribe", {}), QDBus::Block, 1000);
  m_subscribed = signalConnected && reply.type() != QDBusMessage::ErrorMessage;
}

void SystemdClient::runUnit(const QString &action, const QString &method,
                            const QString &unit) {
  if (!m_bus.isConnected() || !m_subscribed) {
    emit actionFinished(action, false,
                        "The system service manager is unavailable.");
    return;
  }

  if (m_pendingRequests.contains(unit)) {
    emit actionFinished(action, false,
                        "Another request for this service is still pending.");
    return;
  }
  m_pendingRequests.insert(unit, action);
  auto *watcher = new QDBusPendingCallWatcher(
      m_bus.asyncCall(managerCall(method, {unit, "replace"}),
                      AuthorizationTimeout),
      this);
  connect(
      watcher, &QDBusPendingCallWatcher::finished, this,
      [this, action, unit](QDBusPendingCallWatcher *call) {
        const QDBusMessage reply = call->reply();
        call->deleteLater();
        m_pendingRequests.remove(unit);
        if (reply.type() == QDBusMessage::ErrorMessage) {
          clearCompletedJobs(unit);
          emit actionFinished(
              action, false,
              dbusError(reply.errorName(), reply.errorMessage()));
          return;
        }
        if (reply.arguments().isEmpty()) {
          clearCompletedJobs(unit);
          emit actionFinished(action, false,
                              "Systemd returned no job for this request.");
          return;
        }

        const QString jobPath =
            qvariant_cast<QDBusObjectPath>(reply.arguments().first()).path();
        if (jobPath.isEmpty()) {
          clearCompletedJobs(unit);
          emit actionFinished(action, false,
                              "Systemd returned an invalid job.");
          return;
        }
        if (m_completedJobs.contains(jobPath)) {
          const QString result = m_completedJobs.take(jobPath).result;
          clearCompletedJobs(unit);
          finishJob(action, result);
          return;
        }
        clearCompletedJobs(unit);
        m_pendingJobs.insert(jobPath, action);
      });
}

void SystemdClient::setAnonSurfEnabled(bool enabled) {
  callUnitFileMethod(enabled ? "boot-enable" : "boot-disable",
                     enabled ? "EnableUnitFiles" : "DisableUnitFiles");
}

void SystemdClient::callUnitFileMethod(const QString &action,
                                       const QString &method) {
  QVariantList arguments{QStringList{AnonSurfUnit}, false};
  if (method == "EnableUnitFiles")
    arguments.append(false);

  auto *watcher = new QDBusPendingCallWatcher(
      m_bus.asyncCall(managerCall(method, arguments), AuthorizationTimeout),
      this);
  connect(watcher, &QDBusPendingCallWatcher::finished, this,
          [this, action](QDBusPendingCallWatcher *call) {
            const QDBusMessage reply = call->reply();
            call->deleteLater();
            if (reply.type() == QDBusMessage::ErrorMessage) {
              emit actionFinished(
                  action, false,
                  dbusError(reply.errorName(), reply.errorMessage()));
              return;
            }
            requestAnonSurfEnabled();
            emit actionFinished(action, true, {});
          });
}

void SystemdClient::requestAnonSurfEnabled() {
  auto *watcher = new QDBusPendingCallWatcher(
      m_bus.asyncCall(managerCall("GetUnitFileState", {AnonSurfUnit})), this);
  connect(watcher, &QDBusPendingCallWatcher::finished, this,
          [this](QDBusPendingCallWatcher *call) {
            const QDBusMessage reply = call->reply();
            call->deleteLater();
            if (reply.type() == QDBusMessage::ErrorMessage ||
                reply.arguments().isEmpty())
              return;
            const QString state = reply.arguments().first().toString();
            emit bootStateReady(state == "enabled" ||
                                state == "enabled-runtime");
          });
}

void SystemdClient::requestStatusText() {
  QDBusMessage message = QDBusMessage::createMethodCall(
      Service, AnonSurfPath, PropertiesInterface, "Get");
  message.setArguments({ServiceInterface, "StatusText"});
  auto *watcher = new QDBusPendingCallWatcher(m_bus.asyncCall(message), this);
  connect(watcher, &QDBusPendingCallWatcher::finished, this,
          [this](QDBusPendingCallWatcher *call) {
            const QDBusMessage reply = call->reply();
            call->deleteLater();
            if (reply.type() == QDBusMessage::ErrorMessage ||
                reply.arguments().isEmpty())
              return;
            const QDBusVariant value =
                qvariant_cast<QDBusVariant>(reply.arguments().first());
            emit statusTextReady(value.variant().toString().trimmed());
          });
}

QString SystemdClient::dbusError(const QString &name, const QString &message) {
  if (name.contains("AccessDenied") ||
      name.contains("InteractiveAuthorizationRequired") ||
      name.contains("PolicyKit")) {
    return "Authorization was denied or no Polkit agent is available.";
  }
  return message.isEmpty() ? "The system service request failed." : message;
}
