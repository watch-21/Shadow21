#include <QHostAddress>
#include "application/externalterminal.h"
#include <QJsonArray>
#include "application/workspace.h"
#include <QCoreApplication>
#include <QFile>
#include <QFileInfo>
#include <QJsonDocument>
#include <QJsonArray>
#include <QSaveFile>
#include <QStandardPaths>
#include <QTimer>
#include <QDateTime>
#include <QDir>
#include <QSettings>
#include <QRegularExpression>
#include <unistd.h>
#include <QGuiApplication>
#include <QClipboard>

namespace { Workspace *activeWorkspace=nullptr; }
QString WText(const QString &source){return activeWorkspace?activeWorkspace->translate(source):source;}

QString Workspace::cliPath(){
 const QString adjacent=QCoreApplication::applicationDirPath()+"/../release/shadow21";
 if(QFileInfo(adjacent).isExecutable())return adjacent;
 return "/usr/bin/shadow21";
}
Workspace::Workspace(QObject *parent):QObject(parent){
 activeWorkspace=this;
 QSettings settings(QSettings::IniFormat,QSettings::UserScope,"shadow21","traffic");
 m_interface=settings.value("interface").toString();
 m_clock.start();m_pollTimer.setInterval(1000);
 connect(&m_pollTimer,&QTimer::timeout,this,[this](){readTraffic();automaticTick(m_clock.elapsed());});
 if(!QCoreApplication::arguments().contains("--inspect-ui"))m_pollTimer.start();
 readTraffic();
 QFile file(":/i18n/de.json");if(file.open(QIODevice::ReadOnly))m_translations=QJsonDocument::fromJson(file.readAll()).object();
}
Workspace::~Workspace(){
 if(activeWorkspace==this)activeWorkspace=nullptr;
 for(auto *process:findChildren<QProcess *>()){
  process->disconnect(this);
  if(process->state()!=QProcess::NotRunning){process->kill();process->waitForFinished(100);}
 }
}
QString Workspace::translate(const QString &source)const{
 if(language()!="de")return source;
 return m_translations.value(source).toString(source);
}
void Workspace::load(){run("preferences",{"preferences"});}
void Workspace::save(const QString &language,const QString &mode,const QString &endpoint,bool tmux){
 run("preferences",{"preferences","set","--language",language,"--mode",mode,"--endpoint",endpoint,"--tmux="+QString(tmux?"true":"false")});
}
void Workspace::switchMode(const QString &target){
 if(target!="system"&&target!="proxy")return;
 if(m_busy||m_routingBusy)return;
 if(!m_routingKnown||m_routingActive){m_message="Stop system routing before switching modes.";emit changed();return;}
 m_pendingMode=target;
 if(m_modeStatus.value("phase").toString()!="inactive")privileged("mode-cleanup","/usr/lib/shadow21/shadow21-helper",{"session-recover"});
 else completeModeSelection(target);
}
void Workspace::completeModeSelection(const QString &target){
 m_pendingMode=target;
 if(target=="proxy"){
  m_terminalMessage.clear();
  privileged("prepare-tor","/usr/lib/shadow21/shadow21-helper",{"prepare-tor"});
 }else save(language(),target,endpoint(),tmuxEnabled());
}
void Workspace::refreshHealth(){run("health",{"health"});}
void Workspace::refreshSessions(){refreshModeStatus();}
void Workspace::sessionAction(const QString &action,const QString &name){
 if(m_busy)return;
 if(action!="start"&&action!="stop"&&action!="detach"&&action!="open")return;
 if(action=="open"){emit terminalAttachmentRequested(name);openExternalTerminal(name);return;}
 if(action=="start"&&(!tmuxEnabled()||mode()!="proxy")){m_message="Select proxy mode and enable tmux first.";emit changed();return;}
 if(action=="start"&&!sessionStartAvailable()){terminalFailure("The session cannot start while routing is active, unknown, or another operation is pending.");return;}
 if(action=="start"&&!terminalAvailable()){terminalFailure("No supported terminal found. Install xterm first.");return;}
 invalidateProxyIP();
 if(action=="start"){captureBaseline();openExternalTerminal(name,true);return;}
 if(action=="stop")m_pendingSession="stop";
 privileged(action=="start"?"session-start":"session","/usr/lib/shadow21/shadow21-helper",{"session-"+action,name});
}
void Workspace::testProxy(){if(connectionActive()&&!m_routingActive)refreshIP(true);}
void Workspace::setAutoIP(bool enabled){run("preferences",{"preferences","set",QString("--auto-ip=")+(enabled?"true":"false")});}
QString Workspace::presetCommand(int index) const {
 switch(index){
 case 0:return QStringLiteral("while true; do pkill -HUP tor; sleep 1; done");
 case 1:return QStringLiteral("curl --socks5-hostname localhost:9050 https://api.ipify.org/");
 case 2:return QStringLiteral("socks5://127.0.0.1:9050");
 default:return {};
 }
}
bool Workspace::copyPresetCommand(int index) {
 const auto command=presetCommand(index);
 if(command.isEmpty())return false;
 auto *clipboard=QGuiApplication::clipboard();
 if(!clipboard)return false;
 clipboard->setText(command);
 return clipboard->text()==command;
}
void Workspace::copyIP(){const auto value=check().value("ip").toString();if(ipState()=="ready"&&!value.isEmpty())QGuiApplication::clipboard()->setText(value);}
void Workspace::invalidate(IPRequest &request,bool resetComparison){
 ++request.generation;
 if(request.process){request.process->kill();request.process=nullptr;}
 const bool hadResult=!request.result.isEmpty();
 request.busy=false;request.result={};request.state=hadResult?"stale":"idle";
 if(resetComparison){request.previousIP.clear();request.lastSuccessfulTimestamp.clear();}
}
void Workspace::invalidateIP(){invalidate(m_systemIP,true);invalidate(m_proxyIP,true);m_nextIP=0;emit changed();}
void Workspace::invalidateSystemIP(){invalidate(m_systemIP,true);emit changed();}
void Workspace::invalidateProxyIP(){invalidate(m_proxyIP,true);emit changed();}
Workspace::IPRequest &Workspace::ipRequest(int kind){return kind==2?m_baselineIP:kind==1?m_proxyIP:m_systemIP;}
void Workspace::refreshIP(bool useProxy){refreshRequest(useProxy?1:0);}
void Workspace::refreshBaseline(){
 if(m_routingKnown&&!m_routingActive&&!m_routingBusy&&m_modeStatus.value("phase").toString()=="inactive"){m_baselineFrozen=false;refreshRequest(2);}
}
void Workspace::captureBaseline(){
 m_baselineFrozen=true;
 if(m_baselineIP.busy){invalidate(m_baselineIP,true);m_baselineIP.state="idle";}
 // A failed or aged observation is never presented as a fresh pre-start address.
 const auto time=QDateTime::fromString(m_baselineIP.result.value("timestamp").toString(),Qt::ISODate);
 if(m_baselineIP.state!="ready"||!time.isValid()||time.secsTo(QDateTime::currentDateTimeUtc())>30){invalidate(m_baselineIP,true);}
 emit changed();
}
void Workspace::setRoutingState(bool known,bool active,bool busy){
 if(m_routingKnown==known&&m_routingActive==active&&m_routingBusy==busy)return;
 if(active!=m_routingActive||known!=m_routingKnown||busy!=m_routingBusy){invalidate(m_systemIP,true);if(!m_baselineFrozen)invalidate(m_baselineIP,false);}
 m_routingKnown=known;m_routingActive=active;m_routingBusy=busy;
 emit changed();automaticTick(m_clock.elapsed());
}
void Workspace::automaticTick(qint64 now){
 if(QCoreApplication::arguments().contains("--inspect-ui"))return;
 const auto phase=m_modeStatus.value("phase").toString();
 QString context;
 if(m_loaded&&autoIP()&&m_routingKnown&&!m_routingBusy&&!m_busy&&(!terminalRunning()||phase=="active")){
  if(!m_routingActive&&phase=="active")context="proxy";
  else if(m_routingActive&&phase=="inactive")context="system";

 }
 if(context!=m_autoContext){
  // Cancel requests whose connection context is no longer observed active.
  if(m_autoContext=="proxy")invalidate(m_proxyIP,false);
  if(m_autoContext=="system")invalidate(m_systemIP,false);
  if(m_autoContext=="baseline"&&m_baselineIP.busy)invalidate(m_baselineIP,false);
  if(context=="baseline")m_baselineFrozen=false;
  m_autoContext=context;m_nextIP=now;
  emit changed();
 }
 if(context.isEmpty()||now<m_nextIP)return;
 const int kind=context=="proxy"?1:context=="system"?0:2;
 if(ipRequest(kind).busy)return;
 m_nextIP=now+(kind==1?1000:15000);
 refreshRequest(kind);
}
void Workspace::refreshRequest(int kind){
 if(QCoreApplication::arguments().contains("--inspect-ui"))return;
 if(kind!=2)m_lastProxy=kind==1;
 auto &request=ipRequest(kind);
 ++request.generation;if(request.process)request.process->kill();
 const auto generation=request.generation;
 request.busy=true;request.state="checking";emit changed();
 auto *process=new QProcess(this);request.process=process;
 auto *timer=new QTimer(process);timer->setSingleShot(true);timer->setInterval(30000);
 connect(timer,&QTimer::timeout,process,[process](){process->kill();});
 connect(process,&QProcess::finished,this,[this,process,timer,generation,kind](int code,QProcess::ExitStatus status){
  timer->stop();const auto output=process->readAllStandardOutput();process->deleteLater();
  auto &request=ipRequest(kind);
  if(generation!=request.generation)return;
  request.process=nullptr;request.busy=false;
  QJsonParseError error;auto document=QJsonDocument::fromJson(output,&error);auto result=document.object();
  QHostAddress address(result.value("ip").toString());
  if(code!=0||status!=QProcess::NormalExit||error.error!=QJsonParseError::NoError||!document.isObject()||address.isNull()||!result.value("is_tor").isBool()){
   request.result={{"last_successful_timestamp",request.lastSuccessfulTimestamp}};request.state="failed";emit changed();return;
  }
  result.insert("ip",address.toString());
  if(!QDateTime::fromString(result.value("timestamp").toString(),Qt::ISODate).isValid())result.insert("timestamp",QDateTime::currentDateTimeUtc().toString(Qt::ISODate));
  result.insert("scope",kind==2?"baseline":kind==1?"proxy":"system");
 request.lastSuccessfulTimestamp=result.value("timestamp").toString();
  if(kind==1){result.insert("comparison",request.previousIP.isEmpty()?"first":request.previousIP==address.toString()?"unchanged":"changed");request.previousIP=address.toString();}
  request.result=result;request.state="ready";emit changed();
 });
 connect(process,&QProcess::errorOccurred,this,[this,process,generation,kind](QProcess::ProcessError error){
  if(error!=QProcess::FailedToStart)return;
  auto &request=ipRequest(kind);
  if(generation==request.generation){request.busy=false;request.process=nullptr;request.result={{"last_successful_timestamp",request.lastSuccessfulTimestamp}};request.state="failed";emit changed();}
  process->deleteLater();
 });
 process->start(cliPath(),kind==1?QStringList{"proxy","test"}:QStringList{"ip","--json"});timer->start();
}
void Workspace::refreshModeStatus(){
 if(m_modeProcess)return;
 auto *process=new QProcess(this);m_modeProcess=process;
 auto *timer=new QTimer(process);timer->setSingleShot(true);timer->setInterval(8000);
 connect(timer,&QTimer::timeout,process,[process](){process->kill();});
 connect(process,&QProcess::finished,this,[this,process,timer](int code,QProcess::ExitStatus exitStatus){
  timer->stop();
  if(m_modeProcess!=process){process->deleteLater();return;}
  m_modeProcess=nullptr;
  const auto doc=QJsonDocument::fromJson(process->readAllStandardOutput());process->deleteLater();
  QJsonObject status;
  if(code==0&&exitStatus==QProcess::NormalExit&&doc.isObject())status=doc.object();
  else status={{"phase","unknown"},{"mode","unknown"}};
  const auto oldPhase=m_modeStatus.value("phase");
  if(status!=m_modeStatus){m_modeStatus=status;if(oldPhase!=status.value("phase")){invalidateProxyIP();}emit changed();}
  QStringList names;
  if(status.value("owner").toInteger(-1)==qint64(getuid()))for(const auto &name:status.value("sessions").toArray())names.append(name.toString());
  if(names!=m_sessions){m_sessions=names;emit changed();}
 });
 connect(process,&QProcess::errorOccurred,this,[this,process](QProcess::ProcessError error){if(error==QProcess::FailedToStart){m_modeProcess=nullptr;m_modeStatus={{"phase","unknown"}};emit changed();process->deleteLater();}});
 process->start(cliPath(),{"tmux","status"});timer->start();
}
void Workspace::selectInterface(const QString &name){
 if(!m_interfaces.contains(name)||name==m_interface)return;
 m_interface=name;m_trafficTime=-1;m_traffic={{"state","sampling"}};
 QSettings settings(QSettings::IniFormat,QSettings::UserScope,"shadow21","traffic");
 settings.setValue("interface",name);emit changed();
}
void Workspace::readTraffic(){
 QFile file("/proc/net/dev");
 if(!file.open(QIODevice::ReadOnly)){m_traffic={{"state","unavailable"}};m_trafficTime=-1;emit changed();return;}
 sampleTraffic(file.readAll(),m_clock.elapsed());
}
void Workspace::sampleTraffic(const QByteArray &data,qint64 now){
 struct Counters {quint64 rx,tx,rp,tp;};
 QMap<QString,Counters> rows;
 for(const auto &line:QString::fromLatin1(data).split('\n')){
  const auto colon=line.lastIndexOf(':');if(colon<0)continue;
  const auto name=line.left(colon).trimmed();
  const auto fields=line.mid(colon+1).trimmed().split(QRegularExpression("\\s+"),Qt::SkipEmptyParts);
  if(name=="lo"||fields.size()!=16)continue;
  bool good=true;
  auto number=[&](int i){bool ok=false;auto n=fields[i].toULongLong(&ok);good=good&&ok;return n;};
  Counters value{number(0),number(8),number(1),number(9)};
  if(good)rows.insert(name,value);
 }
 m_interfaces=rows.keys();
 if(m_interface.isEmpty()&&!m_interfaces.isEmpty()){
  // Prefer an observed default route; users can choose another interface.
  QFile route("/proc/net/route");
  if(route.open(QIODevice::ReadOnly))for(const auto &line:QString::fromLatin1(route.readAll()).split('\n')){
   const auto fields=line.split(QRegularExpression("\\s+"),Qt::SkipEmptyParts);
   if(fields.size()>3&&fields[1]=="00000000"&&rows.contains(fields[0])){m_interface=fields[0];break;}
  }
  if(m_interface.isEmpty())m_interface=m_interfaces.first();
 }
 if(!rows.contains(m_interface)){
  m_traffic={{"state","unavailable"}};m_trafficTime=-1;emit changed();return;
 }
 const auto c=rows.value(m_interface);
 const bool reset=m_trafficTime<0||c.rx<m_rx||c.tx<m_tx||c.rp<m_rp||c.tp<m_tp;
 const double seconds=(now-m_trafficTime)/1000.0;
 if(reset){m_firstRx=c.rx;m_firstTx=c.tx;m_firstRp=c.rp;m_firstTp=c.tp;}
 m_traffic={{"state",reset?"sampling":"ready"},
 {"download_bps",reset||seconds<=0?0.0:(c.rx-m_rx)/seconds},
 {"upload_bps",reset||seconds<=0?0.0:(c.tx-m_tx)/seconds},
 {"received_bytes",QString::number(c.rx-m_firstRx)},
 {"sent_bytes",QString::number(c.tx-m_firstTx)},
 {"received_packets",QString::number(c.rp-m_firstRp)},
 {"sent_packets",QString::number(c.tp-m_firstTp)}};
 m_rx=c.rx;m_tx=c.tx;m_rp=c.rp;m_tp=c.tp;m_trafficTime=now;emit changed();
}
void Workspace::recoverProxy(){if(m_busy)return;m_pendingSession="stop";privileged("session","/usr/lib/shadow21/shadow21-helper",{"session-recover"});}
void Workspace::exportDiagnostics(){run("export",{"diagnostics"});}
void Workspace::loadNetworkSettings(){privileged("config-read","/usr/lib/shadow21/shadow21-helper",{"config-read"});}
void Workspace::installTmux(){privileged("install","/usr/lib/shadow21/shadow21-install",{"tmux"});}
void Workspace::applyNetworkSettings(const QString &bridge,bool autoBridge,bool safeSocks,bool killSwitch){
 QJsonObject settings{{"bridge",bridge},{"auto_bridge",autoBridge},{"safe_socks",safeSocks},{"kill_switch",killSwitch}};
 privileged("config","/usr/lib/shadow21/shadow21-helper",{"config"},QJsonDocument(settings).toJson(QJsonDocument::Compact));
}
void Workspace::privileged(const QString &action,const QString &program,const QStringList &args,const QByteArray &input){
 if(m_busy||geteuid()==0)return;
 m_busy=true;m_message=action=="prepare-tor"?"Starting Tor before opening tmux…":"Working…";emit changed();
 auto *process=new QProcess(this);auto *timer=new QTimer(process);timer->setSingleShot(true);timer->setInterval(180000);
 connect(timer,&QTimer::timeout,process,[process](){process->kill();});
 connect(process,&QProcess::finished,this,[this,process,timer,action](int code,QProcess::ExitStatus status){timer->stop();finish(action,status==QProcess::NormalExit?code:-1,process->readAllStandardOutput());process->deleteLater();});
 connect(process,&QProcess::errorOccurred,this,[this,process](QProcess::ProcessError error){if(error==QProcess::FailedToStart){m_busy=false;m_pendingMode.clear();m_pendingSession.clear();m_message="Authorization helper is unavailable.";emit changed();process->deleteLater();}});
 process->start("/usr/bin/pkexec",QStringList{program}+args);
 if(!input.isEmpty())process->write(input);
 process->closeWriteChannel();timer->start();
}
void Workspace::run(const QString &action,const QStringList &args){
 if(m_busy)return;
 m_busy=true;m_message="Working…";emit changed();
 auto *process=new QProcess(this);auto *timer=new QTimer(process);timer->setSingleShot(true);timer->setInterval(150000);
 connect(timer,&QTimer::timeout,process,[process](){process->kill();});
 connect(process,&QProcess::finished,this,[this,process,timer,action](int code,QProcess::ExitStatus status){timer->stop();finish(action,status==QProcess::NormalExit?code:-1,process->readAllStandardOutput());process->deleteLater();});
 connect(process,&QProcess::errorOccurred,this,[this,process](QProcess::ProcessError error){if(error==QProcess::FailedToStart){m_busy=false;m_message="shadow21 CLI is unavailable. Build or install it first.";emit changed();process->deleteLater();}});
 process->start(cliPath(),args);timer->start();
}
void Workspace::finish(const QString &action,int exitCode,const QByteArray &output){
 if(action=="prepare-tor"&&exitCode==0){
  const auto status=QJsonDocument::fromJson(output).object();
  if(!status.value("prepared").toBool()||!QStringList{"tor.service","tor@default.service"}.contains(status.value("unit").toString()))exitCode=1;
 }
 m_busy=false;
 if(exitCode!=0){
  m_pendingMode.clear();m_pendingSession.clear();
  if(action=="prepare-tor"){
   const auto detail=QJsonDocument::fromJson(output).object().value("detail").toString();
   terminalFailure("Tor could not be started. The mode was not changed.",detail.isEmpty()?translate("Check Tor installation, service state and authorization."):detail);return;
  }
  const auto code=QJsonDocument::fromJson(output).object().value("error").toString();
  if(code=="proxy-readiness")m_message="The Tor proxy connection check failed. Check connectivity and Tor bootstrap; the session was not started.";
  else if(code=="tor-conflict")m_message="An existing Tor service is running. Stop it explicitly before starting managed proxy mode.";
  else if(code=="port-conflict")m_message="Port 9050 is occupied or cannot be inspected. Check the existing Tor listener.";
  else if(code=="config-conflict")m_message="Tor configuration was edited externally. It was preserved; resolve recovery before switching modes.";
  else if(code=="mode-conflict")m_message="Stop or recover system routing before starting proxy mode.";
  else if(code=="busy")m_message="Another lifecycle operation is in progress. Wait and refresh.";
  else if(code=="dependency")m_message="A required dependency is missing. Check Tor, tmux, systemd and iproute2.";
  else if(code=="owner")m_message="The root session belongs to another user.";
  else if(code=="bootstrap")m_message="Tor bootstrap timed out. Check connectivity and bridge configuration.";
  else if(code=="unsupported")m_message="This Tor configuration or platform requires manual migration. No supported start was completed.";
  else if(action=="mac"){
   const auto detail=QJsonDocument::fromJson(output).object().value("detail").toString();
   m_message="MAC operation failed. Stop both modes and VPNs, use a supported NetworkManager connection, and check recovery or authorization.";
   if(!detail.isEmpty())m_message += "\n"+translate("Technical details")+": "+detail.left(1000);
  }
  else m_message="Operation failed. Check dependencies, proxy readiness and authorization.";
  if(action!="mac"){
   const auto detail=QJsonDocument::fromJson(output).object().value("detail").toString();
   if(!detail.isEmpty())m_message+="\n"+translate("Technical details")+": "+detail.left(1000);
  }
  if(action=="mac")m_macMessage=m_message;
  refreshModeStatus();emit changed();return;
 }
 const auto doc=QJsonDocument::fromJson(output);
 if(action=="prepare-tor"){
  if(m_pendingMode!="proxy")return;
  save(language(),"proxy","127.0.0.1:9050",true);return;
 }
 if(action=="mode-cleanup"){
  const auto target=m_pendingMode;m_pendingMode.clear();
  if(m_modeProcess){m_modeProcess->kill();m_modeProcess=nullptr;}
  invalidateProxyIP();refreshModeStatus();closeExternalTerminal();emit terminalSessionStopped();
  if(!target.isEmpty())completeModeSelection(target);
  return;
 }
 if(action=="mac"){m_macMessage="MAC operation completed.";invalidateIP();invalidate(m_baselineIP,true);m_nextIP=0;}
 if(action=="preferences"){
  const bool initialLoad=!m_loaded;
  const auto previous=language();if(doc.isObject()){auto next=doc.object();const bool routeChanged=next.value("endpoint")!=m_preferences.value("endpoint")||next.value("mode")!=m_preferences.value("mode");m_preferences=next;m_loaded=true;if(routeChanged)invalidateIP();}
  if(!m_pendingMode.isEmpty()&&mode()==m_pendingMode){const auto selected=m_pendingMode;m_pendingMode.clear();emit modeReady(selected);}
  else if(initialLoad&&m_loaded)emit modeReady(mode());
  if(previous!=language())emit languageChanged();
  QTimer::singleShot(0,this,&Workspace::refreshHealth);
 }else if(action=="config-read")m_network=doc.object();
 else if(action=="health")m_health=doc.object();
 
 else if(action=="sessions"){
  m_sessions.clear();for(const auto &value:doc.array())m_sessions.append(value.toString());
 }else if(action=="session"){if(m_pendingSession=="stop"){closeExternalTerminal();emit terminalSessionStopped();m_pendingSession.clear();}QTimer::singleShot(0,this,&Workspace::refreshSessions);}
 else if(action=="session-start"){
  refreshModeStatus();
  m_message="Root session started. Authorize attachment in the external terminal.";
  const auto name=m_pendingSession;m_pendingSession.clear();emit changed();
  emit terminalAttachmentRequested(name);openExternalTerminal(name);return;
 }
 else if(action=="export"){
  const QString dir=QStandardPaths::writableLocation(QStandardPaths::DocumentsLocation);QDir().mkpath(dir);
  QSaveFile file(dir+"/shadow21-diagnostics.json");if(file.open(QIODevice::WriteOnly)&&file.write(output)==output.size()&&file.commit()){m_message="Redacted report saved to Documents/shadow21-diagnostics.json";emit changed();return;}
  m_message="Diagnostic export could not be saved.";emit changed();return;
 }
 m_message="Operation completed";emit changed();
}

QString Workspace::currentMAC()const{
 static const QRegularExpression safe("^[a-zA-Z0-9_][a-zA-Z0-9_.:-]{0,14}$");
 if(!safe.match(m_interface).hasMatch()||m_interface=="lo")return {};
 QFile f("/sys/class/net/"+m_interface+"/address");if(!f.open(QIODevice::ReadOnly))return {};
 return QString::fromLatin1(f.read(64)).trimmed();
}
bool Workspace::terminalAvailable()const{return !ExternalTerminal::executable().isEmpty();}
void Workspace::installTerminal(){privileged("install","/usr/lib/shadow21/shadow21-install",{"terminal"});}
void Workspace::changeMAC(bool restore){
 if(m_busy||m_routingActive||m_routingBusy||m_modeStatus.value("phase").toString()=="active"){
  m_message="Stop both connection modes before changing the MAC address.";m_macMessage=m_message;emit changed();return;
 }
 if(!m_routingKnown||m_modeStatus.value("phase").toString()!="inactive"){
  m_message="Network state is unknown or recovery is required. Refresh status or recover the previous mode before changing MAC.";m_macMessage=m_message;emit changed();refreshModeStatus();return;
 }
 if(terminalRunning()){
  m_message="Stop both connection modes before changing the MAC address.";m_macMessage=m_message;emit changed();return;
 }
 m_macMessage="Working…";
 invalidateIP();invalidate(m_baselineIP,true);
 privileged("mac","/usr/lib/shadow21/shadow21-helper",{restore?"mac-restore":"mac-random",m_interface});
}
QString Workspace::terminalProgram()const{return ExternalTerminal::executable();}
void Workspace::terminalFailure(const QString &summary,const QString &detail){
 m_terminalMessage=summary;
 if(!detail.trimmed().isEmpty())m_terminalMessage+="\n"+detail.trimmed().left(2000);
 m_message=summary;emit changed();emit terminalLaunchFailed(terminalMessage());
}
bool Workspace::openExternalTerminal(const QString &name,bool create){
 if(geteuid()==0){terminalFailure("Start the desktop interface as a regular user. Only the terminal session runs as root.");return false;}
 if(!ExternalTerminal::validSession(name)){terminalFailure("Invalid owned session identifier.");return false;}
 if(terminalRunning()){terminalFailure("The owned terminal is already open.");return false;}
 for(const auto &path:QStringList{"/usr/bin/pkexec","/usr/lib/shadow21/shadow21-helper"}){
  if(!QFileInfo(path).isExecutable()){terminalFailure("The installation is incomplete. Reinstall Shadow21 before opening a root session.",path);return false;}
 }
 const auto program=ExternalTerminal::executable();const auto args=ExternalTerminal::arguments(program,name,create);
 if(program.isEmpty()||args.isEmpty()){terminalFailure("No supported terminal found. Install xterm first.");return false;}
 return launchTerminal(program,args);
}
bool Workspace::launchTerminal(const QString &program,const QStringList &arguments){
 if(terminalRunning())return false;
 auto *process=new QProcess(this);m_terminalProcess=process;
 // Emulator stderr contains launch/display/library errors. The root shell uses
 // the emulator's PTY; its input and output are never captured by the GUI.
 connect(process,&QProcess::errorOccurred,this,[this,process,program](QProcess::ProcessError error){
  if(error==QProcess::FailedToStart){
   if(m_terminalProcess==process)m_terminalProcess=nullptr;
   terminalFailure("The terminal could not be started.",program+": "+process->errorString());process->deleteLater();
  }
 });
 connect(process,&QProcess::finished,this,[this,process,program](int code,QProcess::ExitStatus status){
  if(m_terminalProcess==process)m_terminalProcess=nullptr;
  const auto error=QString::fromUtf8(process->readAllStandardError()).trimmed();
  if(process->property("ownedStop").toBool()){
   m_terminalMessage="The owned terminal was closed after Stop.";emit changed();
  }else if(code==0&&status==QProcess::NormalExit){
   m_terminalMessage="Terminal closed. The session may remain detached.";m_message=m_terminalMessage;emit changed();
  }else{
   terminalFailure("Terminal attachment failed. The details below identify the terminal error.",program+" ("+QString::number(code)+")\n"+error);
  }
  refreshModeStatus();process->deleteLater();
 });
 connect(process,&QProcess::started,this,[this](){
  m_terminalMessage="Authorize the external terminal. Tor readiness is checked before tmux starts.";
  m_message=m_terminalMessage;emit changed();refreshModeStatus();
 });
 m_terminalMessage="Opening the system terminal…";emit changed();
 process->start(program,arguments);return true;
}
void Workspace::closeExternalTerminal(){
 if(!m_terminalProcess)return;
 QPointer<QProcess> process=m_terminalProcess;
 process->setProperty("ownedStop",true);
 if(process->state()!=QProcess::NotRunning){process->terminate();QTimer::singleShot(3000,this,[process](){if(process&&process->state()!=QProcess::NotRunning)process->kill();});}
}
