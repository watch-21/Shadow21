#include "externalterminal.h"
#include <QFileInfo>
#include <QRegularExpression>
#include <QDir>
#include <sys/stat.h>

bool ExternalTerminal::validSession(const QString &name){
 static const QRegularExpression allowed("^(0|tor|shadow21-[a-zA-Z0-9_-]{1,48})$");
 return allowed.match(name).hasMatch();
}
QString ExternalTerminal::executable(){
 QStringList candidates;
 const auto preferred=QFileInfo("/usr/bin/x-terminal-emulator").canonicalFilePath();
 const QStringList supported{"xfce4-terminal","qterminal","konsole","xterm"};
 if(supported.contains(QFileInfo(preferred).fileName()))candidates.append(preferred);
 const auto desktop=qEnvironmentVariable("XDG_CURRENT_DESKTOP").toLower();
 if(desktop.contains("kde"))candidates.append("/usr/bin/konsole");
 if(desktop.contains("lxqt"))candidates.append("/usr/bin/qterminal");
 candidates.append({"/usr/bin/xfce4-terminal","/usr/bin/qterminal","/usr/bin/konsole","/usr/bin/xterm"});
 candidates.removeDuplicates();
 for(const auto &path:candidates){
  struct stat info{};const auto real=QFileInfo(path).canonicalFilePath().toLocal8Bit();
  if(!real.isEmpty()&&::stat(real.constData(),&info)==0&&S_ISREG(info.st_mode)&&info.st_uid==0&&(info.st_mode&0111)&&!(info.st_mode&0022))return path;
 }
 return {};
}
QStringList ExternalTerminal::arguments(const QString &program,const QString &session,bool create){
 if(!validSession(session))return {};
 // Alternatives may resolve through /bin on merged-/usr systems.
 const auto name=QFileInfo(program).fileName();
 if(!QStringList{"xfce4-terminal","qterminal","konsole","xterm"}.contains(name)||(program!="/usr/bin/"+name&&program!="/bin/"+name))return {};
 QStringList result;
 if(name=="xfce4-terminal")result={"--disable-server","--geometry=120x36","-x"};
 else if(name=="konsole")result={"--separate","--nofork","--geometry","1100x740","-e"};
 else if(name=="qterminal")result={"-e"};
 else if(name=="xterm")result={"-geometry","120x36","-xrm","XTerm*VT100.translations: #override\nCtrl Shift<Key>C: copy-selection(CLIPBOARD)\nCtrl Shift<Key>V: insert-selection(CLIPBOARD)","-e"};
 else return {};
 return result+QStringList{"/usr/bin/pkexec","/usr/lib/shadow21/shadow21-helper",create?"session-console":"session-attach",session};
}
