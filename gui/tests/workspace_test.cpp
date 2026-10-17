#include "application/workspace.h"
#include <QCoreApplication>
#include <QTimer>
#include <cstdio>

int main(int argc,char **argv){
 QCoreApplication app(argc,argv);
 Workspace workspace;
 QString attached;
 QObject::connect(&workspace,&Workspace::terminalAttachmentRequested,&app,[&](const QString &name){attached=name;});
 workspace.sessionAction("open","tor");
 if(attached!="tor"||workspace.busy()){std::fprintf(stderr,"Embedded attachment was not dispatched locally\n");return 1;}
 QString stage="initial";
 auto fail=[&](const char *message){std::fprintf(stderr,"%s\n",message);app.exit(1);};
 QObject::connect(&workspace,&Workspace::changed,&app,[&](){
  const auto system=workspace.systemCheck();const auto proxy=workspace.proxyCheck();
  if(stage=="initial"&&workspace.systemIPState()=="ready"&&workspace.proxyIPState()=="ready"){
   if(system.value("ip").toString()!="203.0.113.1"||proxy.value("ip").toString()!="198.51.100.42"||proxy.value("comparison").toString()!="first"||system.value("scope").toString()!="system"||proxy.value("scope").toString()!="proxy"){fail("System/proxy result isolation failed");return;}
   stage="same";QTimer::singleShot(0,&app,[&](){workspace.refreshIP(true);});
  }else if(stage=="same"&&workspace.proxyIPState()=="ready"&&!workspace.proxyIPBusy()){
   if(proxy.value("comparison").toString()!="unchanged"){fail("Unchanged comparison failed");return;}
   stage="change";QTimer::singleShot(0,&app,[&](){workspace.refreshIP(true);});
  }else if(stage=="change"&&workspace.proxyIPState()=="ready"&&!workspace.proxyIPBusy()){
   if(proxy.value("ip").toString()!="198.51.100.43"||proxy.value("comparison").toString()!="changed"){fail("Changed comparison failed");return;}
   if(workspace.systemCheck().value("ip").toString()!="203.0.113.1"){fail("Proxy check overwrote system observation");return;}
   stage="invalid";QTimer::singleShot(0,&app,[&](){workspace.refreshIP(true);});
  }else if(stage=="invalid"&&workspace.proxyIPState()=="failed"){
   if(!proxy.value("ip").toString().isEmpty()||proxy.value("last_successful_timestamp").toString().isEmpty()){fail("Invalid address shown or successful timestamp lost");return;}
   if(workspace.systemCheck().value("ip").toString()!="203.0.113.1"){fail("Failure affected system observation");return;}
   stage="cancel";QTimer::singleShot(0,&app,[&](){workspace.refreshIP(true);QTimer::singleShot(80,&app,[&](){workspace.refreshIP(true);});});
  }else if(stage=="cancel"&&workspace.proxyIPState()=="ready"){
   stage="waiting";
   if(proxy.value("ip").toString()!="198.51.100.44"){fail("Canceled result accepted");return;}
   QTimer::singleShot(600,&app,[&](){
    if(workspace.proxyCheck().value("ip").toString()!="198.51.100.44"){fail("Late response overwrote current result");return;}
    workspace.invalidateProxyIP();
    if(!workspace.proxyCheck().isEmpty()||workspace.proxyIPBusy()||workspace.proxyIPState()!="stale"||workspace.systemCheck().value("ip").toString()!="203.0.113.1"){fail("Context invalidation affected the other route");return;}
    std::puts("PASS: independent system/proxy results, first/unchanged/changed comparisons, malformed IP rejection, preserved successful timestamps, asynchronous progress, cancellation and stale-context invalidation");app.exit(0);
   });
  }
 });
 workspace.refreshIP(false);workspace.refreshIP(true);
 if(!workspace.systemIPBusy()||!workspace.proxyIPBusy()||workspace.busy()){std::fprintf(stderr,"IP checks blocked general actions\n");return 1;}
 QTimer::singleShot(8000,&app,[&](){fail("Workspace integration timed out");});return app.exec();
}
