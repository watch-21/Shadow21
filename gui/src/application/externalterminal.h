#pragma once
#include <QString>
#include <QStringList>

// Fixed emulator commands create independent windows. They never reuse a shared
// terminal process, accept shell text, or change the terminal title.
namespace ExternalTerminal {
QString executable();
QStringList arguments(const QString &executable,const QString &session,bool create=false);
bool validSession(const QString &session);
}
