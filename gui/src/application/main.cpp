#include "anonsurf/anonsurfbackend.h"
#include "application/instance.h"
#include "application/workspace.h"
#include <QSettings>
#include <QScreen>
#include <qqml.h>
#include <QQuickWindow>
#include <QTimer>
#include <unistd.h>
#include "tray/traycontroller.h"

#include <QApplication>
#include <QPalette>
#include <QColor>
#include <QDebug>
#include <QIcon>
#include <QQmlApplicationEngine>
#include <QQuickStyle>
#include <QWindow>

#ifndef ANONSURF_VERSION
#define ANONSURF_VERSION "1"
#endif

int main(int argc, char *argv[]) {
  QQuickStyle::setStyle("Fusion");

  QApplication app(argc, argv);
  auto palette=app.palette();
  palette.setColor(QPalette::Window,QColor("#081821"));
  palette.setColor(QPalette::Base,QColor("#0C2633"));
  palette.setColor(QPalette::AlternateBase,QColor("#081821"));
  palette.setColor(QPalette::Text,QColor("#F4FAFC"));
  palette.setColor(QPalette::WindowText,QColor("#F4FAFC"));
  palette.setColor(QPalette::Button,QColor("#0A202A"));
  palette.setColor(QPalette::ButtonText,QColor("#BDEEFF"));
  palette.setColor(QPalette::Highlight,QColor("#007CA6"));
  palette.setColor(QPalette::HighlightedText,QColor("#F7FCFF"));
  palette.setColor(QPalette::Disabled,QPalette::Text,QColor("#A0B9C4"));
  palette.setColor(QPalette::Disabled,QPalette::ButtonText,QColor("#A0B9C4"));
  app.setPalette(palette);
  app.setApplicationName("shadow21-gui");
  app.setApplicationDisplayName("Shadow21");
  app.setOrganizationName("shadow21");
  app.setApplicationVersion(ANONSURF_VERSION);
  app.setDesktopFileName("org.shadow21.Desktop");
  app.setWindowIcon(
      QIcon::fromTheme("shadow21", QIcon(":/icons/shadow21.svg")));

  const bool inspect = app.arguments().contains("--inspect-ui");
  if(geteuid()==0 && !inspect){qCritical()<<"Start the desktop interface as a regular user.";return 1;}
  DesktopInstance instance;
  const auto startup = instance.start();
  if (startup == DesktopInstance::Startup::Activated)
    return 0;
  if (startup == DesktopInstance::Startup::Failed)
    return 1;

  Workspace workspace;
  AnonSurfBackend backend;
  backend.setWorkspace(&workspace);
  TrayController desktop(backend);
  QQmlApplicationEngine engine;
  // Qt 6.4 does not include this embedded module prefix in its default paths.
  engine.addImportPath(QStringLiteral("qrc:/qt/qml"));
  engine.setInitialProperties(
      {{"workspace", QVariant::fromValue(static_cast<QObject *>(&workspace))},
       {"backend", QVariant::fromValue(static_cast<QObject *>(&backend))},
       {"desktop", QVariant::fromValue(static_cast<QObject *>(&desktop))}});
  engine.load(QUrl("qrc:/qt/qml/Shadow21/Desktop/qml/app/Main.qml"));
  if (engine.rootObjects().isEmpty())
    return 1;

  auto *window = qobject_cast<QWindow *>(engine.rootObjects().first());
  if (!window) {
    qCritical() << "The main shadow21 window is unavailable.";
    return 1;
  }
  if(!inspect){
    const auto area=window->screen()->availableGeometry();
    QSettings settings("shadow21","window");
    window->setWidth(qBound(qMin(720,area.width()-32),settings.value("width",1100).toInt(),area.width()-32));
    window->setHeight(qBound(qMin(540,area.height()-48),settings.value("height",880).toInt(),area.height()-48));
    auto *saveTimer=new QTimer(&app);saveTimer->setSingleShot(true);saveTimer->setInterval(400);
    QObject::connect(window,&QWindow::widthChanged,saveTimer,[saveTimer](){saveTimer->start();});
    QObject::connect(window,&QWindow::heightChanged,saveTimer,[saveTimer](){saveTimer->start();});
    QObject::connect(saveTimer,&QTimer::timeout,&app,[window](){if(window->visibility()==QWindow::Windowed){QSettings s("shadow21","window");s.setValue("width",window->width());s.setValue("height",window->height());}});
  }
  if(inspect){
    const auto args=app.arguments();
    const int pageIndex=args.indexOf("--page");if(pageIndex>=0){const int page=args.value(pageIndex+1).toInt();if(page>=0&&page<=5)window->setProperty("page",page);}
    const int sizeIndex=args.indexOf("--size");if(sizeIndex>=0){const auto size=args.value(sizeIndex+1).split('x');if(size.size()==2){window->setWidth(qMax(480,size[0].toInt()));window->setHeight(qMax(240,size[1].toInt()));}}
  }
  if(inspect && app.arguments().contains("--tmux"))QMetaObject::invokeMethod(window,"openTmux");
  if(inspect&&app.arguments().contains("--dialog")){
   const auto kind=app.arguments().value(app.arguments().indexOf("--dialog")+1);
   QMetaObject::invokeMethod(window,"openInspectionDialog",Q_ARG(QVariant,QVariant(kind)));
  }
  desktop.attachWindow(window);
  QObject::connect(&instance, &DesktopInstance::activationRequested, &desktop,
                   &TrayController::showWindow);

  if(inspect){
    QTimer::singleShot(2500,&app,[&](){auto *quick=qobject_cast<QQuickWindow *>(window);QString path=app.arguments().value(app.arguments().indexOf("--inspect-ui")+1);if(quick && !path.isEmpty())quick->grabWindow().save(path);app.quit();});
  }
  return app.exec();
}
