#include "tray/traymenu.h"

#include <QAction>
#include "application/workspace.h"

TrayMenu::TrayMenu(QWidget *parent) : QMenu(parent) {
  setToolTipsVisible(true);
  m_status = addAction("shadow21: UNAVAILABLE");
  m_status->setEnabled(false);
  m_show=addAction("Show shadow21", this, &TrayMenu::showRequested);
  addSeparator();
  m_toggle = addAction("Start shadow21", this, [this]() {
    emit lifecycleRequested(m_active ? "stop" : "start");
  });
  m_restart = addAction("Restart", this,
                        [this]() { emit lifecycleRequested("restart"); });
  m_identity = addAction("New Identity", this, &TrayMenu::identityRequested);
  m_ip = addAction("Verify IP", this, &TrayMenu::ipRequested);
  addSeparator();
  m_quit = addAction("Quit", this, &TrayMenu::quitRequested);
  m_quit->setToolTip("Quit the interface; the active mode keeps running.");
  m_toggle->setToolTip("Uses the SafeKill option selected in the main window.");
  m_restart->setToolTip(
      "Uses the SafeKill option selected in the main window.");
  m_identity->setToolTip(
      "Request new circuits; a different exit IP is not guaranteed.");
  updateState({});
}

void TrayMenu::updateState(const TrayState &state) {
  m_show->setText(WText("Show shadow21"));
 m_restart->setText(WText("Restart"));m_identity->setText(WText("New Identity"));m_ip->setText(WText("Verify IP"));m_quit->setText(WText("Quit"));
 m_ip->setToolTip(WText("Checks one request using the selected route at check.torproject.org."));
 m_quit->setToolTip(WText("Quit the interface; the active mode keeps running."));
 m_identity->setToolTip(WText("Request new circuits; a different exit IP is not guaranteed."));
 m_toggle->setToolTip(WText("Uses the SafeKill option selected in the main window."));m_restart->setToolTip(m_toggle->toolTip());
 m_active = state.active;
  m_status->setText("shadow21: " + WText(state.label));
  m_status->setToolTip(WText(state.detail));
  m_toggle->setText(WText(state.active ? "Stop shadow21" : "Start shadow21"));
  const bool ready = state.available && !state.busy;
  m_toggle->setEnabled(ready);
  m_restart->setEnabled(ready && state.active);
  m_identity->setEnabled(ready && state.active && state.torActive);
  m_ip->setEnabled(ready && state.active);
  m_quit->setEnabled(!state.busy);
}
