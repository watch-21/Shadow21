#include "anonsurf/anonsurfbackend.h"
#include "application/workspace.h"
#include <QCoreApplication>
#include <unistd.h>

namespace {
constexpr auto IdentityUnit = "shadow21-identity.service";
} // namespace

AnonSurfBackend::AnonSurfBackend(QObject *parent) : QObject(parent) {
  connect(&m_commands, &CommandClient::statusReady, this,
          &AnonSurfBackend::handleStatus);
  connect(&m_commands, &CommandClient::statusFailed, this,
          [this](const QString &message) {
            m_statusAvailable = false;
            m_protectionDetail = message;
            m_dnsMode = "Unavailable";
            m_dnsServers = "No data";
            invalidateIPCheck();
            emit changed();
          });
  connect(&m_commands, &CommandClient::monitorFailed, this,
          [this](const QString &message) {
            setError(message);
            emit operationFinished("monitor", false, message);
          });
  connect(&m_commands, &CommandClient::safeKillFinished, this,
          [this](bool success, const QString &message) {
            handleAction("safekill", success, message);
          });

  connect(&m_systemd, &SystemdClient::actionFinished, this,
          &AnonSurfBackend::handleAction);
  connect(&m_systemd, &SystemdClient::bootStateReady, this,
          [this](bool enabled) {
            m_bootEnabled = enabled;
            emit changed();
          });
  connect(&m_systemd, &SystemdClient::statusTextReady, this,
          [this](const QString &status) {
            if (m_busy && !status.isEmpty() && status != m_progress) {
              m_progress = status;
              emit changed();
            }
          });

  m_progressTimer.setInterval(750);
  connect(&m_progressTimer, &QTimer::timeout, &m_systemd,
          &SystemdClient::requestStatusText);
}

bool AnonSurfBackend::statusAvailable() const { return m_statusAvailable; }
bool AnonSurfBackend::anonsurfActive() const { return m_anonsurfActive; }
bool AnonSurfBackend::torActive() const { return m_torActive; }
bool AnonSurfBackend::bootEnabled() const { return m_bootEnabled; }
bool AnonSurfBackend::busy() const { return m_busy || !m_pendingMode.isEmpty() || (m_workspace && m_workspace->busy()); }
QString AnonSurfBackend::protectionState() const { return m_protectionState; }
QString AnonSurfBackend::protectionDetail() const { return m_protectionDetail; }
QString AnonSurfBackend::dnsMode() const { return m_dnsMode; }
QString AnonSurfBackend::dnsServers() const { return m_dnsServers; }
QString AnonSurfBackend::publicIP() const { return m_publicIP; }
QString AnonSurfBackend::torCheck() const { return m_torCheck; }
QString AnonSurfBackend::progress() const { return m_progress; }
QString AnonSurfBackend::errorMessage() const { return m_errorMessage; }

void AnonSurfBackend::refresh() {
 if(m_workspace)m_workspace->refreshModeStatus();
  m_commands.requestStatus();
  m_systemd.requestAnonSurfEnabled();
}

void AnonSurfBackend::selectMode(const QString &target){
 if(!m_workspace||busy()||(target!="system"&&target!="proxy"))return;
 if(!m_statusAvailable){setError("Network state is unknown. Refresh status before switching modes.");return;}
 if(m_anonsurfActive){m_pendingMode=target;beginLifecycle("stop",false);}
 else m_workspace->switchMode(target);
}
void AnonSurfBackend::start(bool safeKill) {
 if(geteuid()==0 && QCoreApplication::arguments().contains("--inspect-ui"))return;
  beginLifecycle("start", safeKill);
}

void AnonSurfBackend::stop(bool safeKill) {
 if(geteuid()==0 && QCoreApplication::arguments().contains("--inspect-ui"))return;
 beginLifecycle("stop", safeKill);
}

void AnonSurfBackend::restart(bool safeKill) {
 if(geteuid()==0 && QCoreApplication::arguments().contains("--inspect-ui"))return;
  beginLifecycle("restart", safeKill);
}

void AnonSurfBackend::changeIdentity() {
 if(geteuid()==0 && QCoreApplication::arguments().contains("--inspect-ui"))return;
  if (m_busy)
    return;
  invalidateIPCheck();
  beginAction("identity", "Requesting a new Tor identity");
  m_systemd.runUnit("identity", "StartUnit", IdentityUnit);
}

void AnonSurfBackend::checkIP() {
  if (m_busy)
    return;
  beginIPCheck(false);
}

void AnonSurfBackend::beginIPCheck(bool automatic) {
 if(automatic&&m_workspace&&(!m_workspace->autoIP()||m_workspace->systemIPBusy()))return;
 if(!m_workspace||!m_workspace->connectionActive())return;
 if(m_workspace)m_workspace->refreshIP(!m_anonsurfActive&&(m_workspace->mode()=="proxy"||m_workspace->modeStatus().value("mode").toString()=="proxy"));
}

void AnonSurfBackend::setBootEnabled(bool enabled) {
 if(geteuid()==0 && QCoreApplication::arguments().contains("--inspect-ui"))return;
  if (m_busy || enabled == m_bootEnabled)
    return;
  const QString action = enabled ? "boot-enable" : "boot-disable";
  beginAction(action, enabled ? "Enabling shadow21 at startup"
                              : "Disabling shadow21 at startup");
  m_systemd.setAnonSurfEnabled(enabled);
}

void AnonSurfBackend::openMonitor() { m_commands.openMonitor(); }

void AnonSurfBackend::dismissError() { setError({}); }

void AnonSurfBackend::setWorkspace(Workspace *workspace){
 m_workspace=workspace;
 connect(workspace,&Workspace::terminalLaunchFailed,this,[this](const QString &message){setError(message);emit operationFinished("terminal",false,message);});
 connect(this,&AnonSurfBackend::changed,workspace,[this](){m_workspace->setRoutingState(m_statusAvailable,m_anonsurfActive,m_busy);});
 connect(workspace,&Workspace::changed,this,[this](){
  const auto result=m_workspace->systemCheck();
  m_publicIP=result.value("ip").toString();if(m_publicIP.isEmpty())m_publicIP="Not checked";
  m_torCheck=m_workspace->systemIPState()=="ready"?(result.value("is_tor").toBool()?"TOR":"CLEARNET"):"UNKNOWN";
  updateProtection();emit changed();
 });
}
