#include "cli/commandclient.h"

#include <QDebug>
#include "application/workspace.h"
#include <QTimer>
#include <QStandardPaths>
#include <QFileInfo>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>

namespace {
constexpr auto AnonSurfPath = "/usr/bin/shadow21";
constexpr auto SafeKillPath = "/usr/lib/shadow21/safekill";
constexpr auto KonsolePath = "/usr/bin/konsole";
} // namespace

CommandClient::CommandClient(QObject *parent) : QObject(parent) {
 for(auto *process:{&m_statusProcess,&m_safeKillProcess}) {auto *timer=new QTimer(process);timer->setSingleShot(true);timer->setInterval(30000);connect(process,&QProcess::started,timer,[timer](){timer->start();});connect(process,&QProcess::finished,timer,[timer](int,QProcess::ExitStatus){timer->stop();});connect(timer,&QTimer::timeout,process,[this,process](){if(process==&m_safeKillProcess){m_safeKillTimedOut=true;cancelSafeKill();}else process->kill();});}
  connect(&m_statusProcess, &QProcess::finished, this,
          &CommandClient::handleStatusFinished);
  connect(&m_safeKillProcess, &QProcess::finished, this,
          &CommandClient::handleSafeKillFinished);

  connect(&m_statusProcess, &QProcess::errorOccurred, this,
          [this](QProcess::ProcessError error) {
            if (error == QProcess::FailedToStart)
              emit statusFailed("The shadow21 CLI could not be started.");
          });
  connect(&m_safeKillProcess, &QProcess::errorOccurred, this,
          [this](QProcess::ProcessError error) {
            if (error == QProcess::FailedToStart) {
              emit safeKillFinished(false, "SafeKill could not be started.");
            }
          });
}

CommandClient::~CommandClient() {
  // QProcess can emit completion signals while its owner is being destroyed.
  for (auto *process : {&m_statusProcess, &m_safeKillProcess}) {
    process->disconnect(this);
    if (process->state() != QProcess::NotRunning) {
      process->kill();
      if (!process->waitForFinished(1000))
        qWarning() << "Cannot finish GUI command shutdown:"
                   << process->errorString();
    }
  }
}

void CommandClient::requestStatus() {
  if (m_statusProcess.state() != QProcess::NotRunning)
    return;
  m_statusProcess.start(Workspace::cliPath(), {"status", "--json"});
}

void CommandClient::runSafeKill() {
  if (m_safeKillProcess.state() != QProcess::NotRunning)
    return;
  ++m_safeKillGeneration;
  m_safeKillCanceled=false;m_safeKillTimedOut=false;
  m_safeKillProcess.start(SafeKillPath);
}

void CommandClient::cancelSafeKill(){
 if(m_safeKillProcess.state()==QProcess::NotRunning)return;
 m_safeKillCanceled=true;m_safeKillProcess.terminate();
 const auto generation=m_safeKillGeneration;
 QTimer::singleShot(1500,this,[this,generation](){if(generation==m_safeKillGeneration&&m_safeKillProcess.state()!=QProcess::NotRunning)m_safeKillProcess.kill();});
}
void CommandClient::openMonitor() {
  for(const auto &candidate:QStringList{"x-terminal-emulator","konsole","gnome-terminal","xfce4-terminal","xterm"}){
    auto path=QStandardPaths::findExecutable(candidate);if(path.isEmpty())continue;
    auto flag=candidate=="gnome-terminal"?"--":candidate=="xfce4-terminal"?"-x":"-e";
    if(QProcess::startDetached(path,{flag,Workspace::cliPath(),"monitor"}))return;
  }
  emit monitorFailed("Use a supported terminal emulator to open the Tor monitor.");
}

void CommandClient::handleStatusFinished(int exitCode,
                                         QProcess::ExitStatus exitStatus) {
  if (exitStatus != QProcess::NormalExit || exitCode != 0) {
    emit statusFailed(
        processError(m_statusProcess, "shadow21 status is unavailable."));
    return;
  }

  QJsonParseError parseError;
  const QJsonDocument document = QJsonDocument::fromJson(
      m_statusProcess.readAllStandardOutput(), &parseError);
  if (parseError.error != QJsonParseError::NoError || !document.isObject()) {
    emit statusFailed("shadow21 returned an invalid status response.");
    return;
  }

  const QJsonObject object = document.object();
  if (!object.value("shadow21_active").isBool() ||
      !object.value("tor_active").isBool() ||
      !object.value("dns_state").isString() ||
      !object.value("dns_addresses").isArray()) {
    emit statusFailed("shadow21 returned an incomplete status response.");
    return;
  }

  QStringList addresses;
  for (const QJsonValue &value : object.value("dns_addresses").toArray()) {
    if (value.isString())
      addresses.append(value.toString());
  }
  emit statusReady(object.value("shadow21_active").toBool(),
                   object.value("tor_active").toBool(),
                   object.value("dns_state").toString(), addresses);
}

void CommandClient::handleSafeKillFinished(int exitCode,
                                           QProcess::ExitStatus exitStatus) {
  const bool success = exitStatus == QProcess::NormalExit && exitCode == 0;
  emit safeKillFinished(
      success,
      success
          ? QString{}
          : m_safeKillTimedOut ? WText("SafeKill timed out; cleanup was stopped.") : m_safeKillCanceled ? WText("SafeKill was canceled.") : processError(m_safeKillProcess,
                         "SafeKill failed. Ensure BleachBit is installed."));
}

QString CommandClient::processError(QProcess &process,
                                    const QString &fallback) {
  const QString stderrText =
      QString::fromUtf8(process.readAllStandardError()).trimmed();
  if (stderrText.isEmpty())
    return fallback;

  const QStringList lines = stderrText.split('\n', Qt::SkipEmptyParts);
  for (auto line = lines.crbegin(); line != lines.crend(); ++line) {
    if (line->startsWith("[FAIL] "))
      return line->mid(7).trimmed();
  }
  return stderrText;
}
