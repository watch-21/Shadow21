#include "application/instance.h"

#include <QDBusConnectionInterface>
#include <QDBusMessage>
#include <QDBusReply>
#include <QDebug>
#include <QMessageBox>
#include "application/workspace.h"

namespace {
constexpr auto Service = "org.shadow21.Desktop";
constexpr auto Path = "/org/shadow21/Desktop";
} // namespace

DesktopInstance::DesktopInstance(const QDBusConnection &bus, QObject *parent)
    : QObject(parent), m_bus(bus) {}

DesktopInstance::~DesktopInstance() {
  if (m_registered)
    m_bus.unregisterService(Service);
  if (m_exported)
    m_bus.unregisterObject(Path);
}

DesktopInstance::Startup DesktopInstance::start() {
  if (!m_bus.isConnected()) {
    qWarning() << "The session bus is unavailable; single-instance activation "
                  "is disabled.";
    return Startup::Unavailable;
  }
  m_exported =
      m_bus.registerObject(Path, this, QDBusConnection::ExportScriptableSlots);
  if (!m_exported) {
    qCritical() << "Cannot export the shadow21 activation endpoint:"
                << m_bus.lastError().message();
    return Startup::Failed;
  }

  const auto reply = m_bus.interface()->registerService(
      Service, QDBusConnectionInterface::DontQueueService);
  if (!reply.isValid()) {
    qCritical() << "Cannot register the shadow21 desktop instance:"
                << reply.error().message();
    return Startup::Failed;
  }
  if (reply.value() == QDBusConnectionInterface::ServiceRegistered) {
    m_registered = true;
    return Startup::Primary;
  }

  const auto build=m_bus.call(QDBusMessage::createMethodCall(Service,Path,Service,"BuildId"),QDBus::Block,3000);
  if(build.type()==QDBusMessage::ErrorMessage||build.arguments().value(0).toString()!=BuildId()){
    QMessageBox::warning(nullptr,"Shadow21",WText("A different Shadow21 build is already running. Quit it from the tray, then start the installed version again."));
    return Startup::Failed;
  }
  const auto activation = m_bus.call(
      QDBusMessage::createMethodCall(Service, Path, Service, "Activate"),
      QDBus::Block, 3000);
  if (activation.type() == QDBusMessage::ErrorMessage) {
    qCritical() << "Cannot activate the existing shadow21 window:"
                << activation.errorMessage();
    return Startup::Failed;
  }
  return Startup::Activated;
}

void DesktopInstance::Activate() { emit activationRequested(); }

QString DesktopInstance::BuildId() const { return "20261017-07"; }
