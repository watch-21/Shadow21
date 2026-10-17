#pragma once

#include <QDBusConnection>
#include <QObject>

class DesktopInstance final : public QObject {
  Q_OBJECT
  Q_CLASSINFO("D-Bus Interface", "org.shadow21.Desktop")

public:
  enum class Startup { Primary, Activated, Unavailable, Failed };

  explicit DesktopInstance(
      const QDBusConnection &bus = QDBusConnection::sessionBus(),
      QObject *parent = nullptr);
  ~DesktopInstance() override;
  Startup start();

public slots:
  Q_SCRIPTABLE void Activate();
  Q_SCRIPTABLE QString BuildId() const;

signals:
  void activationRequested();

private:
  QDBusConnection m_bus;
  bool m_registered = false;
  bool m_exported = false;
};
