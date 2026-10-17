#include "tray/traycontroller.h"

#include "anonsurf/anonsurfbackend.h"

#include <QApplication>
#include "application/workspace.h"
#include <QEvent>
#include <QPainter>

TrayController::TrayController(AnonSurfBackend &backend, QObject *parent)
    : QObject(parent), m_backend(backend) {
  m_tray.setContextMenu(&m_menu);
  connect(&m_menu, &TrayMenu::showRequested, this, &TrayController::showWindow);
  connect(&m_menu, &TrayMenu::quitRequested, this, &TrayController::quit);
  connect(&m_menu, &TrayMenu::lifecycleRequested, this,
          &TrayController::lifecycleRequested);
  connect(&m_menu, &TrayMenu::identityRequested, &backend,
          &AnonSurfBackend::changeIdentity);
  connect(&m_menu, &TrayMenu::ipRequested, &backend, &AnonSurfBackend::checkIP);
  connect(&m_notifications, &DesktopNotifications::activated, this,
          &TrayController::showWindow);
  connect(&m_tray, &QSystemTrayIcon::activated, this,
          [this](QSystemTrayIcon::ActivationReason reason) {
            if (reason == QSystemTrayIcon::Trigger ||
                reason == QSystemTrayIcon::DoubleClick)
              showWindow();
          });
  connect(&backend, &AnonSurfBackend::changed, this,
          &TrayController::updateState);
  connect(&backend, &AnonSurfBackend::operationFinished, this,
          [this](const QString &, bool success, const QString &message) {
            m_notifications.send(success ? "shadow21"
                                         : "shadow21 action failed",
                                 success?WText(message):WText("Operation failed"), !success);
          });
  m_availabilityTimer.setInterval(3000);
  connect(&m_availabilityTimer, &QTimer::timeout, this,
          &TrayController::updateAvailability);
  qApp->installEventFilter(this);
  updateState();
  m_tray.show();
  updateAvailability();
  m_availabilityTimer.start();
}

void TrayController::attachWindow(QWindow *window) { m_window = window; }

bool TrayController::trayAvailable() const { return m_available; }

bool TrayController::hideToTray() {
  updateAvailability();
  if (m_quitting || !m_available || !m_window)
    return false;
  m_window->hide();
  if (!m_hideNoticeShown) {
    m_hideNoticeShown = true;
    m_notifications.send(
        "shadow21 is running in the tray",
        "Use the tray menu to reopen the window or quit the interface. "
        "Closing the window does not stop routing.");
  }
  return true;
}

void TrayController::showWindow() {
  if (!m_window)
    return;
  m_window->showNormal();
  m_window->raise();
  m_window->requestActivate();
  m_backend.refresh();
}

void TrayController::quit() {
  if (m_backend.busy())
    return;
  m_quitting = true;
  QCoreApplication::quit();
}

bool TrayController::eventFilter(QObject *object, QEvent *event) {
  if (object == qApp && event->type() == QEvent::Quit)
    m_quitting = true;
  return QObject::eventFilter(object, event);
}

void TrayController::updateAvailability() {
  const bool available = QSystemTrayIcon::isSystemTrayAvailable();
  if (m_available == available)
    return;
  m_available = available;
  if (!available && m_window && !m_window->isVisible() && !m_quitting)
    showWindow();
  qApp->setQuitOnLastWindowClosed(!available);
  emit availabilityChanged();
}

void TrayController::updateState() {
  const QString detail =
      m_backend.busy() ? m_backend.progress() : m_backend.protectionDetail();
  m_menu.updateState({m_backend.statusAvailable(), m_backend.anonsurfActive(),
                      m_backend.torActive(), m_backend.busy(),
                      m_backend.protectionState(), detail});
  m_tray.setToolTip("shadow21: " + WText(m_backend.protectionState()) + "\n" + WText(detail));
  m_tray.setIcon(statusIcon());
}

QIcon TrayController::statusIcon() const {
  QPixmap image = qApp->windowIcon().pixmap(32, 32);
  if (image.isNull())
    return qApp->windowIcon();
  const QString state = m_backend.protectionState();
  const QColor color = state == "ROUTED" ? QColor("#38D9FF")
                       : state == "UNVERIFIED" || state == "DEGRADED"
                           ? QColor("#F4C95D")
                       : state == "INACTIVE" ? QColor("#8CB2C1")
                                             : QColor("#FF453A");
  QPainter painter(&image);
  painter.setRenderHint(QPainter::Antialiasing);
  const qreal size = image.width() / image.devicePixelRatio();
  painter.setPen(QPen(QColor("#050D12"), 1));
  painter.setBrush(color);
  painter.drawEllipse(
      QRectF(size * 0.58, size * 0.58, size * 0.38, size * 0.38));
  painter.end();
  return QIcon(image);
}
