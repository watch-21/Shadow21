#include "application/workspace.h"
#include <QCoreApplication>
#include <QEventLoop>
#include <QElapsedTimer>
#include <QFile>
#include <QThread>
#include <cstdio>
#include <functional>

static bool waitFor(const std::function<bool()> &condition){
 QElapsedTimer timer;timer.start();
 while(!condition()&&timer.elapsed()<2500){QCoreApplication::processEvents(QEventLoop::AllEvents,10);QThread::msleep(1);}
 return condition();
}
static QByteArray counters(int rx,int tx,int rp,int tp){
 return QString("fixture0: %1 %3 0 0 0 0 0 0 %2 %4 0 0 0 0 0 0\n").arg(rx).arg(tx).arg(rp).arg(tp).toLatin1();
}
int main(int argc,char **argv){
 QCoreApplication app(argc,argv);Workspace w;
 auto require=[](bool ok,const char *message){if(!ok){std::fprintf(stderr,"FAIL: %s\n",message);std::exit(1);}};
 w.sampleTraffic(counters(1000,2000,10,20),1000);
 if(w.networkInterface()!="fixture0"){
  w.selectInterface("fixture0");w.sampleTraffic(counters(1000,2000,10,20),1000);
 }
 require(w.traffic().value("state")=="sampling","First sample must not be a rate");
 w.sampleTraffic(counters(3000,3000,30,25),3000);
 require(w.traffic().value("download_bps").toDouble()==1000&&w.traffic().value("upload_bps").toDouble()==500,"Elapsed-time transfer rates");
 require(w.traffic().value("received_packets").toString()=="20"&&w.traffic().value("sent_bytes").toString()=="1000","Monitoring deltas");
 w.sampleTraffic(counters(10,20,1,2),4000);
 require(w.traffic().value("state")=="sampling"&&w.traffic().value("received_bytes").toString()=="0","Counter reset must rebase");
 w.sampleTraffic("fixture0: malformed\n",5000);
 require(w.traffic().value("state")=="unavailable","Malformed counters must not be successful");
 w.load();require(waitFor([&](){return w.ready()&&!w.busy();}),"Preferences");
 w.refreshModeStatus();require(waitFor([&](){return w.modeStatus().value("phase")=="inactive";}),"Inactive status");
 w.setRoutingState(true,false,false);
 require(!w.baselineIPBusy(),"Stopped modes must not trigger automatic external requests");
 w.refreshBaseline();
 require(waitFor([&](){return w.baselineIPState()=="ready";}),"Explicit legacy baseline check");
 require(w.baselineCheck().value("ip")=="203.0.113.1"&&w.systemCheck().isEmpty()&&w.proxyCheck().isEmpty(),"Baseline isolation");
 w.captureBaseline();w.setRoutingState(true,true,false);
 require(waitFor([&](){return w.systemIPState()=="ready";}),"System check after start");
 require(w.baselineCheck().value("ip")=="203.0.113.1","Baseline frozen across Start");
 w.automaticTick(100000);require(w.systemIPBusy(),"Due system check");
 w.automaticTick(130000); // A request already in flight is not replaced.
 require(waitFor([&](){return !w.systemIPBusy();}),"System request completion");
 const auto count=w.systemCheck().value("sequence").toInt();
 w.automaticTick(114999);require(!w.systemIPBusy(),"No system polling before 15 seconds");
 w.automaticTick(115000);require(w.systemIPBusy(),"System poll at 15 seconds");
 require(waitFor([&](){return !w.systemIPBusy();}),"Second system completion");
 require(w.systemCheck().value("sequence").toInt()==count+1,"No overlapping automatic system request");
 QFile phase(qEnvironmentVariable("TELEMETRY_PHASE"));require(phase.open(QIODevice::WriteOnly),"Phase fixture");phase.write("active");phase.close();
 w.refreshModeStatus();require(waitFor([&](){return w.modeStatus().value("phase")=="active";}),"Active proxy status");
 w.setRoutingState(true,false,false);
 require(waitFor([&](){return w.proxyIPState()=="ready";}),"Proxy check");
 w.automaticTick(200000);require(w.proxyIPBusy(),"Due proxy request");
 w.automaticTick(201000);
 require(waitFor([&](){return !w.proxyIPBusy();}),"Proxy completion");
 const auto proxyCount=w.proxyCheck().value("sequence").toInt();
 w.automaticTick(200999);require(!w.proxyIPBusy(),"No proxy polling before 1 second");
 w.automaticTick(201000);require(w.proxyIPBusy(),"Proxy polling at 1 second");
 require(waitFor([&](){return !w.proxyIPBusy();}),"Second proxy completion");
 require(w.proxyCheck().value("sequence").toInt()==proxyCount+1,"No overlapping proxy request");
 require(w.baselineCheck().value("ip")=="203.0.113.1","Proxy result cannot overwrite baseline");
 w.setRoutingState(true,false,true);w.automaticTick(300000);
 require(!w.proxyIPBusy()&&!w.systemIPBusy(),"Stop suspends periodic active checks");
 w.setAutoIP(false);require(waitFor([&](){return !w.busy()&&!w.autoIP();}),"Disable automatic checks");
 w.setRoutingState(true,false,false);w.automaticTick(400000);
 require(!w.baselineIPBusy()&&!w.proxyIPBusy()&&!w.systemIPBusy(),"Disabled automatic checks must send no request");
 w.refreshBaseline();require(!w.baselineIPBusy(),"An active proxy must block direct baseline requests");
 require(phase.open(QIODevice::WriteOnly),"Inactive phase fixture");phase.write("inactive");phase.close();
 w.refreshModeStatus();require(waitFor([&](){return w.modeStatus().value("phase")=="inactive";}),"Stopped proxy status");
 w.refreshBaseline();require(w.baselineIPBusy(),"Manual baseline check available with automatic checks off");
 require(waitFor([&](){return w.baselineIPState()=="ready";}),"Manual baseline result");
 std::puts("PASS: baseline isolation/freeze, system 15 s and proxy 1 s scheduling, no overlapping requests, stop suspension, elapsed-time rates, packet/byte totals, reset and malformed counters");
}
