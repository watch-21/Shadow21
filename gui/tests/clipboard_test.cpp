#include "application/workspace.h"
#include <QGuiApplication>
#include <QClipboard>
#include <cstdio>
int main(int argc,char **argv){
 QGuiApplication app(argc,argv);
 Workspace workspace;
 const QString commands[]={
  "while true; do pkill -HUP tor; sleep 1; done",
  "curl --socks5-hostname localhost:9050 https://api.ipify.org/",
  "socks5://127.0.0.1:9050"
 };
 for(int i=0;i<3;++i){
  if(workspace.presetCommand(i)!=commands[i]||!workspace.copyPresetCommand(i)||QGuiApplication::clipboard()->text()!=commands[i])return 1;
 }
 const auto previous=QGuiApplication::clipboard()->text();
 for(int i:{-1,3,99})if(!workspace.presetCommand(i).isEmpty()||workspace.copyPresetCommand(i)||QGuiApplication::clipboard()->text()!=previous)return 2;
 if(workspace.busy()||workspace.terminalRunning()||!workspace.sessions().isEmpty())return 3;
 std::puts("PASS: all three exact snippets copied to Qt clipboard; invalid indices preserve clipboard; no terminal, session or privileged action started");
}
