#pragma once

#include <QMenu>

struct TrayState {
  bool available = false;
  bool active = false;
  bool torActive = false;
  bool busy = false;
  QString label = "UNAVAILABLE";
  QString detail;
};

class TrayMenu final : public QMenu {
  Q_OBJECT

public:
  explicit TrayMenu(QWidget *parent = nullptr);
  void updateState(const TrayState &state);

signals:
  void showRequested();
  void lifecycleRequested(const QString &action);
  void identityRequested();
  void ipRequested();
  void quitRequested();

private:
  QAction *m_show;
 QAction *m_status;
  QAction *m_toggle;
  QAction *m_restart;
  QAction *m_identity;
  QAction *m_ip;
  QAction *m_quit;
  bool m_active = false;
};
