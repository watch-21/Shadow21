#include "application/workspace.h"
#include "systemd/systemdclient.h"
#include "anonsurf/anonsurfbackend.h"
#include <QCoreApplication>
#include <QEventLoop>
#include <QThread>
#include <functional>
#include <cstdio>

class ModeTest {
 static void require(bool v,const char *m){if(!v){std::fprintf(stderr,"FAIL: %s\n",m);std::exit(1);}}
 static bool waitFor(const std::function<bool()> &c){QElapsedTimer t;t.start();while(!c()&&t.elapsed()<3000){QCoreApplication::processEvents(QEventLoop::AllEvents,10);QThread::msleep(1);}return c();}
public:
 static void run(){
  {
   CommandClient cleanup;
   int finished=0;QString result;
   QObject::connect(&cleanup,&CommandClient::safeKillFinished,&cleanup,[&](bool success,const QString &text){require(!success,"Canceled cleanup cannot report success");++finished;result=text;});
   cleanup.m_safeKillProcess.start("/bin/sleep",{"60"});require(cleanup.m_safeKillProcess.waitForStarted(),"Cleanup cancellation fixture started");
   cleanup.cancelSafeKill();require(waitFor([&](){return finished==1;}),"Cleanup cancellation completes");
   require(result=="SafeKill was canceled.","Cleanup cancellation reason preserved");
   ++cleanup.m_safeKillGeneration;cleanup.m_safeKillCanceled=false;
   cleanup.m_safeKillProcess.start("/bin/sleep",{"60"});require(cleanup.m_safeKillProcess.waitForStarted(),"Second cleanup fixture started");
   QElapsedTimer elapsed;elapsed.start();while(elapsed.elapsed()<1700){QCoreApplication::processEvents(QEventLoop::AllEvents,10);QThread::msleep(1);}
   require(cleanup.m_safeKillProcess.state()!=QProcess::NotRunning,"Old cancellation timer cannot kill a new cleanup");
   cleanup.m_safeKillProcess.disconnect(&cleanup);cleanup.m_safeKillProcess.kill();cleanup.m_safeKillProcess.waitForFinished(1000);
  }

  {
   Workspace initial;initial.m_pollTimer.stop();int ready=0;QString selected;
   QObject::connect(&initial,&Workspace::modeReady,&initial,[&](const QString &mode){++ready;selected=mode;});
   initial.finish("preferences",0,"{\"mode\":\"proxy\",\"language\":\"en\",\"auto_ip\":false}");
   require(initial.ready()&&ready==1&&selected=="proxy","Initial view must match saved proxy mode without starting Tor");
   require(initial.m_pendingMode.isEmpty()&&!initial.terminalRunning(),"Restoring a view must not start Tor or a terminal");
   initial.testProxy();require(!initial.proxyIPBusy(),"Inactive proxy button must not issue an external request");
   initial.finish("preferences",0,"{\"mode\":\"proxy\"}");require(ready==1,"Background preference refresh must not change views");
   initial.finish("session",1,"{\"error\":\"proxy-readiness\",\"detail\":\"SOCKS connection timed out\"}");
   require(initial.message().contains("connection check failed")&&initial.message().contains("SOCKS connection timed out")&&!initial.message().contains("Stop it explicitly"),"Readiness failures retain their cause without false service-conflict guidance");
  }

  Workspace w;w.m_routingKnown=true;w.m_preferences={{"mode","proxy"},{"language","en"},{"tmux_enabled",true}};w.m_loaded=true;w.m_modeStatus={{"phase","inactive"}};
  w.switchMode("system");require(waitFor([&](){return !w.busy();}),"Preference switch completed");require(w.mode()=="system","Inactive proxy must return to system");
  w.m_routingActive=true;w.switchMode("proxy");require(!w.busy()&&w.mode()=="system","Active routing guard");w.m_routingActive=false;
  w.switchMode("unknown");require(!w.busy()&&w.mode()=="system","Invalid mode rejected");
  // Exercise completion paths without running privileged commands.
  w.m_preferences.insert("mode","proxy");w.m_modeStatus={{"phase","active"}};w.m_pendingMode="system";w.m_busy=true;w.finish("mode-cleanup",1,"{\"error\":\"config-conflict\"}");
  require(w.mode()=="proxy"&&!w.busy()&&w.m_pendingMode.isEmpty(),"Failed cleanup retains mode and allows recovery");
  w.m_pendingMode="system";w.m_busy=true;w.finish("mode-cleanup",126,"");require(w.mode()=="proxy"&&!w.busy()&&w.m_pendingMode.isEmpty(),"Denied authorization cannot change mode");
  int stopped=0;QObject::connect(&w,&Workspace::terminalSessionStopped,&w,[&](){++stopped;});w.m_pendingMode="system";w.m_busy=true;w.finish("mode-cleanup",0,"");
  require(waitFor([&](){return !w.busy()&&w.mode()=="system";}),"Successful cleanup selects target");require(stopped==1,"Cleanup closes owned terminal");
  w.m_pendingSession="stop";w.finish("session",0,"");require(stopped==2,"Stop closes owned terminal after detach");
  AnonSurfBackend b;b.setWorkspace(&w);b.m_statusAvailable=false;b.selectMode("proxy");require(!b.m_errorMessage.isEmpty()&&w.mode()=="system","Unknown state cannot switch");
  b.m_pendingMode="proxy";b.finishAction("stop",false,"fixture restoration failure");require(b.m_pendingMode.isEmpty()&&w.mode()=="system","Failed system stop cannot select proxy");
  require(waitFor([&](){return !w.busy();}),"Pending fixture status completed");
  w.m_modeStatus={{"phase","inactive"}};b.m_statusAvailable=true;b.m_anonsurfActive=true;b.m_pendingMode="proxy";
  b.finishAction("stop",true,"");
  require(w.mode()=="system"&&w.m_pendingMode=="proxy","Tor preparation is required before selecting proxy");
  w.finish("prepare-tor",126,"");require(w.mode()=="system"&&w.m_pendingMode.isEmpty(),"Denied Tor authorization leaves mode unchanged");
  w.m_pendingMode="proxy";w.finish("prepare-tor",0,"{}");require(w.mode()=="system"&&w.m_pendingMode.isEmpty(),"Malformed Tor success cannot select proxy");
  int ready=0;QObject::connect(&w,&Workspace::modeReady,&w,[&](const QString &mode){if(mode=="proxy")++ready;});
  w.m_pendingMode="proxy";w.finish("prepare-tor",0,"{\"prepared\":true,\"unit\":\"tor@default.service\"}");
  require(waitFor([&](){return !w.busy()&&w.mode()=="proxy";}),"Successful Tor preparation selects proxy after cleanup");
  require(ready==1,"Proxy view readiness is emitted only after verified preparation and preference save");
  require(!w.systemRoutingActive(),"System routing is inactive before selecting proxy");
  w.m_routingKnown=true;w.m_routingActive=true;w.changeMAC(false);
  require(!w.busy()&&w.message()=="Stop both connection modes before changing the MAC address.","MAC change must reject active system routing before authorization");
  w.m_routingActive=false;w.m_modeStatus={{"phase","active"}};w.changeMAC(true);
  require(!w.busy()&&w.message()=="Stop both connection modes before changing the MAC address.","MAC restore must reject an active proxy");
  w.finish("mac",1,"{\"error\":\"operation\",\"detail\":\"fixture MAC failure\"}");
  const auto macReason=w.macMessage();w.finish("health",0,"{}");require(!macReason.isEmpty()&&w.macMessage()==macReason,"Background health refresh preserves MAC failure details");
  const auto failure=SystemdClient::failureMessage("start","failed","Failed: active VPN conflicts with system routing");
  require(failure.contains("active VPN conflicts")&&failure.contains("journalctl -b -u shadow21d.service"),"Systemd failure preserves reason and recovery log guidance");
  const auto identityFailure=SystemdClient::failureMessage("identity","failed","");
  require(identityFailure.contains("shadow21-identity.service"),"Identity failure uses its own service log");
  {
   Workspace launch;
   launch.m_pollTimer.stop();launch.m_loaded=true;launch.m_routingKnown=true;
   launch.m_preferences={{"mode","proxy"},{"tmux_enabled",true},{"auto_ip",false}};
   launch.m_modeStatus={{"phase","inactive"}};
   launch.m_baselineIP.busy=true;
   require(launch.sessionStartAvailable(),"Slow automatic baseline must not block session Start");
   launch.captureBaseline();require(!launch.baselineIPBusy(),"Start cancels a pending baseline check");
   launch.m_routingActive=true;require(!launch.sessionStartAvailable(),"Real routing still blocks Start");launch.m_routingActive=false;
   int failures=0;QString detail;
   QObject::connect(&launch,&Workspace::terminalLaunchFailed,&launch,[&](const QString &text){++failures;detail=text;});
   require(launch.launchTerminal("/bin/sh",{"-c","printf 'display unavailable fixture' >&2; exit 7"}),"Real emulator failure fixture starts");
   require(waitFor([&](){return !launch.terminalRunning()&&failures==1;}),"Terminal exit produces one failure");
   require(detail.contains("display unavailable fixture")&&detail.contains("7"),"Emulator stderr and exit code preserved");
   launch.finish("health",0,"{}");require(launch.terminalMessage().contains("display unavailable fixture"),"Background health cannot erase terminal error");
   require(launch.launchTerminal("/nonexistent/shadow21-terminal",{}),"Failed exec is asynchronous");
   require(waitFor([&](){return failures==2&&!launch.terminalRunning();}),"Missing executable is visible and clears launch state");
   require(launch.launchTerminal("/bin/sleep",{"60"}),"Owned terminal lifetime fixture starts");
   require(waitFor([&](){return launch.terminalRunning();}),"Terminal Running state");
   launch.m_baselineFrozen=true;launch.m_preferences.insert("auto_ip",true);launch.automaticTick(999999);
   require(!launch.baselineIPBusy(),"No baseline restart while external authorization is pending");
   launch.closeExternalTerminal();require(waitFor([&](){return !launch.terminalRunning();}),"Owned Stop closes terminal");
   require(failures==2,"Intentional Stop must not raise an attachment error");
  }
  QProcess owned,unrelated;owned.start("/bin/sleep",{"60"});unrelated.start("/bin/sleep",{"60"});
  require(owned.waitForStarted()&&unrelated.waitForStarted(),"Terminal ownership fixture processes");
  w.m_terminalProcess=&owned;w.closeExternalTerminal();require(owned.waitForFinished(2000),"Owned terminal process must close");
  require(unrelated.state()!=QProcess::NotRunning,"Unrelated terminal must remain alive");
  w.m_terminalProcess=nullptr;unrelated.kill();unrelated.waitForFinished(2000);
  std::puts("PASS: slow-IP Start regression, real terminal failure/missing exec/Stop fixtures, retained stderr, inactive mode return, active/unknown state guards, denied authorization, failed/successful cleanup, owned terminal closure and failed system-stop coordination (fixtures)");
 }
};
int main(int argc,char **argv){QCoreApplication app(argc,argv);ModeTest::run();}
