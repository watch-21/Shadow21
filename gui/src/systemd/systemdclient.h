#pragma once

#include <QDBusConnection>
#include <QDBusObjectPath>
#include <QHash>
#include <QObject>

class SystemdClient final : public QObject {
  Q_OBJECT

public:
  explicit SystemdClient(QObject *parent = nullptr);

  void runUnit(const QString &action, const QString &method,
               const QString &unit);
  void setAnonSurfEnabled(bool enabled);
  void requestAnonSurfEnabled();
  void requestStatusText();
  static QString failureMessage(const QString &action,const QString &result,const QString &status);

signals:
  void actionFinished(const QString &action, bool success,
                      const QString &message);
  void bootStateReady(bool enabled);
  void statusTextReady(const QString &status);

private slots:
  void handleJobRemoved(uint id, const QDBusObjectPath &job,
                        const QString &unit, const QString &result);

private:
  struct CompletedJob {
    QString unit;
    QString result;
  };

  void callUnitFileMethod(const QString &action, const QString &method);
  void finishJob(const QString &action, const QString &result);
  void clearCompletedJobs(const QString &unit);
  static QString dbusError(const QString &name, const QString &message);
  static QString jobError(const QString &action, const QString &result);

  QDBusConnection m_bus;
  QHash<QString, QString> m_pendingRequests;
  QHash<QString, QString> m_pendingJobs;
  QHash<QString, CompletedJob> m_completedJobs;
  bool m_subscribed = false;
};
