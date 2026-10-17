#include "application/externalterminal.h"
#include <QCoreApplication>
#include <cstdio>
#include <QTemporaryDir>
#include <QFile>
#include <QProcess>
int main(int argc,char **argv){
 QCoreApplication app(argc,argv);
 for(const auto &exe:QStringList{"/usr/bin/xfce4-terminal","/usr/bin/konsole","/usr/bin/qterminal","/usr/bin/xterm"}){
  const auto args=ExternalTerminal::arguments(exe,"0");
  if(args.size()<5||args.last()!="0"||!args.contains("session-attach")||!args.contains("/usr/bin/pkexec")||!args.contains("/usr/lib/shadow21/shadow21-helper"))return 1;
  for(const auto &a:args)if(a=="-T"||a.startsWith("--title")||a.contains("root tmux")||a=="tor")return 2;
  if(exe.contains("xfce")&&!args.contains("--disable-server"))return 3;
  if(exe.contains("konsole")&&(!args.contains("--separate")||!args.contains("--nofork")))return 4;
  if(!ExternalTerminal::arguments(exe,"0;id").isEmpty())return 5;
 }
 if(!ExternalTerminal::arguments("/tmp/terminal","0").isEmpty())return 6;
 if(!ExternalTerminal::arguments("/usr/bin/xterm","0",true).contains("session-console"))return 7;
 const auto xargs=ExternalTerminal::arguments("/usr/bin/xterm","0");
 if(!xargs.contains("-xrm")||!xargs.join(" ").contains("copy-selection(CLIPBOARD)")||!xargs.join(" ").contains("insert-selection(CLIPBOARD)"))return 8;
 if(ExternalTerminal::arguments("/bin/xterm","0")!=ExternalTerminal::arguments("/usr/bin/xterm","0"))return 9;
 if(!ExternalTerminal::arguments("/usr/bin/../tmp/xterm","0").isEmpty())return 10;
 const auto qargs=ExternalTerminal::arguments("/usr/bin/qterminal","0",true);
 if(qargs.first()!="-e"||qargs.contains("-s")||qargs.contains("--size"))return 11;
 QTemporaryDir temp;
 QFile fixture(temp.filePath("qterminal"));
 if(!fixture.open(QIODevice::WriteOnly))return 12;
 fixture.write("#!/bin/sh\n[ \"$1\" = -e ] || { echo invalid-option >&2;exit 2; }\nshift\nprintf '%s\\n' \"$@\"\n");fixture.close();
 QProcess emulator;emulator.start("/bin/sh",QStringList{fixture.fileName()}+qargs);
 if(!emulator.waitForFinished(3000)||emulator.exitCode()!=0)return 13;
 if(emulator.readAllStandardOutput()!=QByteArray("/usr/bin/pkexec\n/usr/lib/shadow21/shadow21-helper\nsession-console\n0\n"))return 14;
 std::puts("PASS: QTerminal command-only compatibility fixture, fixed authorized external terminal argv, automatic numeric session, dedicated emulator flags, useful initial geometry, no custom title and rejected injection/untrusted executable");
}
