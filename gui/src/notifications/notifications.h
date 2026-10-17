#pragma once

#include <QDBusConnection>
#include <QObject>
#include <QSet>

class DesktopNotifications final : public QObject {
  Q_OBJECT

public:
  explicit DesktopNotifications(
      const QDBusConnection &bus = QDBusConnection::sessionBus(),
      QObject *parent = nullptr);
  void send(const QString &title, const QString &message, bool error = false);

signals:
  void activated();

private slots:
  void actionInvoked(uint id, const QString &action);
  void notificationClosed(uint id, uint reason);

private:
  void reportFailure(const QString &message);
  QDBusConnection m_bus;
  QSet<uint> m_pending;
  bool m_failed = false;
};
