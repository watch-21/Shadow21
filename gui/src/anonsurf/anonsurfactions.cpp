#include "anonsurf/anonsurfbackend.h"
#include "application/workspace.h"

namespace {
constexpr auto AnonSurfUnit = "shadow21d.service";
constexpr int IPCheckDelayMs = 2000;
} // namespace

void AnonSurfBackend::beginLifecycle(const QString &action, bool safeKill) {
  if (m_busy)
    return;
  if(action!="stop"&&m_workspace&&(m_workspace->mode()!="system"||m_workspace->modeStatus().value("phase").toString()!="inactive")){
   setError("Stop or recover proxy mode and select system routing first.");return;
  }
  m_cleanupWarning.clear();
  if(action=="start"&&m_workspace)m_workspace->captureBaseline();
  invalidateIPCheck();
  if (safeKill) {
    m_pendingAction = action;
    beginAction("safekill",
                "Closing networked applications and clearing caches");
    m_commands.runSafeKill();
    return;
  }
  runLifecycle(action);
}

void AnonSurfBackend::runLifecycle(const QString &action) {
  beginAction(action, lifecycleProgress(action));
  m_progressTimer.start();
  m_systemd.runUnit(action, lifecycleMethod(action), AnonSurfUnit);
}

void AnonSurfBackend::beginAction(const QString &action,
                                  const QString &progress) {
  m_busy = true;
  m_activeAction = action;
  m_progress = progress;
  setError({});
  emit changed();
}

void AnonSurfBackend::finishAction(const QString &action, bool success,
                                   const QString &message) {
  const bool automatic = action == "ip" && m_autoCheck;
  m_autoCheck = false;
  m_busy = false;
  m_activeAction.clear();
  m_progressTimer.stop();
  updateProtection();
  if (success) {
    m_progress = successMessage(action);
    if(!m_cleanupWarning.isEmpty())m_progress += "\n"+WText("Cleanup failed; networking will still be stopped.")+"\n"+WText(m_cleanupWarning);
    if (action == "stop") {
      invalidateIPCheck();
    }
    if (action == "start" || action == "restart" || action == "identity")
      scheduleIPCheck();
  } else if (automatic) {
    // An unattended check must not raise a modal error the user did not ask
    // for; the unknown result is already visible in the overview.
    m_progress = "Public IP check unavailable";
  } else {
    m_progress = "Operation failed";
    setError(message);
  }
  emit changed();
  if (!automatic) {
    QString result = success ? WText(successMessage(action)) : WText(message);
    if (action == "ip" && success)
      result = QString("Public IP: %1\nTor check: %2\n"
                       "This result describes this request only.")
                   .arg(m_publicIP, m_torCheck);
    if(!m_cleanupWarning.isEmpty())result+="\n"+WText("Cleanup failed; networking will still be stopped.")+"\n"+WText(m_cleanupWarning);
    emit operationFinished(action, success, result);
  }
  m_cleanupWarning.clear();
  refresh();
  if(action=="stop"&&!m_pendingMode.isEmpty()){
   const auto target=m_pendingMode;m_pendingMode.clear();
   if(success&&m_workspace){
    // Completed StopUnit includes ExecStop restoration. The CLI also checks
    // remaining ownership markers before it accepts the new preference.
    m_anonsurfActive=false;m_statusAvailable=true;
    m_workspace->setRoutingState(true,false,false);m_workspace->switchMode(target);
   }
   emit changed();
  }
}

void AnonSurfBackend::scheduleIPCheck() {
  // Give Tor a short bootstrap grace period before the automatic check.
  QTimer::singleShot(IPCheckDelayMs, this, [this]() {
    if (m_busy || !m_statusAvailable || !m_anonsurfActive || (m_workspace && !m_workspace->autoIP()))
      return;
    beginIPCheck(true);
  });
}

void AnonSurfBackend::handleAction(const QString &action, bool success,
                                   const QString &message) {
  if (action == "safekill" && (success || m_pendingAction=="stop") && !m_pendingAction.isEmpty()) {
    if(!success)m_cleanupWarning=message;
    const QString nextAction = m_pendingAction;
    m_pendingAction.clear();
    runLifecycle(nextAction);
    return;
  }
  m_pendingAction.clear();
  finishAction(action, success, message);
}

void AnonSurfBackend::setError(const QString &message) {
  if (m_errorMessage == message)
    return;
  m_errorMessage = message;
  emit changed();
}
