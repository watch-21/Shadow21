#include "systemd/systemdclient.h"
#include "application/workspace.h"
#include <QDBusPendingCallWatcher>
#include <QDBusVariant>
#include <QDBusMessage>
#include <QDBusArgument>

void SystemdClient::handleJobRemoved(uint, const QDBusObjectPath &job,
                                     const QString &unit,
                                     const QString &result) {
  const QString jobPath = job.path();
  if (m_pendingJobs.contains(jobPath)) {
    finishJob(m_pendingJobs.take(jobPath), result);
    return;
  }
  if (m_pendingRequests.contains(unit))
    m_completedJobs.insert(jobPath, {unit, result});
}

void SystemdClient::finishJob(const QString &action, const QString &result) {
  if(result=="done"){emit actionFinished(action,true,{});return;}
  const auto path=action=="identity"?QString("/org/freedesktop/systemd1/unit/shadow21_2didentity_2eservice"):QString("/org/freedesktop/systemd1/unit/shadow21d_2eservice");
  auto query=QDBusMessage::createMethodCall("org.freedesktop.systemd1",path,"org.freedesktop.DBus.Properties","GetAll");
  query.setArguments({"org.freedesktop.systemd1.Service"});
  auto *watcher=new QDBusPendingCallWatcher(m_bus.asyncCall(query,5000),this);
  connect(watcher,&QDBusPendingCallWatcher::finished,this,[this,action,result](QDBusPendingCallWatcher *call){
   const auto reply=call->reply();call->deleteLater();
   QVariantMap properties;
   if(reply.type()!=QDBusMessage::ErrorMessage&&!reply.arguments().isEmpty())properties=qdbus_cast<QVariantMap>(reply.arguments().first());
   const auto text=properties.value("StatusText").toString().trimmed();
   const auto message=failureMessage(action,result,text);
   emit actionFinished(action,false,message);
  });
}

void SystemdClient::clearCompletedJobs(const QString &unit) {
  for (auto job = m_completedJobs.begin(); job != m_completedJobs.end();) {
    if (job->unit == unit)
      job = m_completedJobs.erase(job);
    else
      ++job;
  }
}

QString SystemdClient::jobError(const QString &action, const QString &result) {
  if (action == "identity") {
    return WText("Tor could not change identity (systemd result: %1).")
        .arg(result);
  }
  return WText("shadow21 could not complete the %1 operation (systemd result: %2).")
      .arg(action, result);
}

QString SystemdClient::failureMessage(const QString &action,const QString &result,const QString &status){
 const auto unit=action=="identity"?QString("shadow21-identity.service"):QString("shadow21d.service");
 QString message=jobError(action,result);
 if(!status.isEmpty())message+="\n"+WText("Service failure details")+": "+status.left(1500);
 message+="\n"+WText("Check the service log")+":\njournalctl -b -u "+unit+" -u shadow21-recovery.service --no-pager -n 60";
 return message;
}
