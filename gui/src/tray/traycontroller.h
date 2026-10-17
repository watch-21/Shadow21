#pragma once

#include "notifications/notifications.h"
#include "tray/traymenu.h"

#include <QPointer>
#include <QSystemTrayIcon>
#include <QTimer>
#include <QWindow>

class AnonSurfBackend;

class TrayController final : public QObject {
  Q_OBJECT
  Q_PROPERTY(bool trayAvailable READ trayAvailable NOTIFY availabilityChanged)

public:
  explicit TrayController(AnonSurfBackend &backend, QObject *parent = nullptr);
  void attachWindow(QWindow *window);
  bool trayAvailable() const;
  Q_INVOKABLE bool hideToTray();
  void showWindow();

signals:
  void availabilityChanged();
  void lifecycleRequested(const QString &action);

protected:
  bool eventFilter(QObject *object, QEvent *event) override;

private:
  void updateAvailability();
  void updateState();
  void quit();
  QIcon statusIcon() const;
  AnonSurfBackend &m_backend;
  TrayMenu m_menu;
  QSystemTrayIcon m_tray;
  DesktopNotifications m_notifications;
  QTimer m_availabilityTimer;
  QPointer<QWindow> m_window;
  bool m_available = false;
  bool m_quitting = false;
  bool m_hideNoticeShown = false;
};
