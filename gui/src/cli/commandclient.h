#pragma once

#include <QObject>
#include <QProcess>
#include <QStringList>

class CommandClient final : public QObject {
  Q_OBJECT

public:
  explicit CommandClient(QObject *parent = nullptr);
  ~CommandClient() override;

  void requestStatus();
  void runSafeKill();
  void cancelSafeKill();
  void openMonitor();

signals:
  void statusReady(bool anonsurfActive, bool torActive, const QString &dnsState,
                   const QStringList &dnsAddresses);
  void statusFailed(const QString &message);
  void safeKillFinished(bool success, const QString &message);
  void monitorFailed(const QString &message);

private:
 friend class ModeTest;
  void handleStatusFinished(int exitCode, QProcess::ExitStatus exitStatus);
  void handleSafeKillFinished(int exitCode, QProcess::ExitStatus exitStatus);
  static QString processError(QProcess &process, const QString &fallback);

  QProcess m_statusProcess;
  QProcess m_safeKillProcess;
 quint64 m_safeKillGeneration=0;
 bool m_safeKillCanceled=false,m_safeKillTimedOut=false;
};
