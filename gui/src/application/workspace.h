#pragma once
#include <QObject>
#include <QVariantMap>
#include <QStringList>
#include <QJsonObject>
#include <QProcess>
#include <QTranslator>
#include <QPointer>
#include <QElapsedTimer>
#include <QTimer>

class Workspace final : public QObject {
 Q_OBJECT
 Q_PROPERTY(QString language READ language NOTIFY changed)
 Q_PROPERTY(QString mode READ mode NOTIFY changed)
 Q_PROPERTY(bool torPreparing READ torPreparing NOTIFY changed)
 Q_PROPERTY(QString endpoint READ endpoint NOTIFY changed)
 Q_PROPERTY(bool systemRoutingActive READ systemRoutingActive NOTIFY changed)
 Q_PROPERTY(bool tmuxEnabled READ tmuxEnabled NOTIFY changed)
 Q_PROPERTY(bool connectionActive READ connectionActive NOTIFY changed)
 Q_PROPERTY(bool busy READ busy NOTIFY changed)
 Q_PROPERTY(bool ipBusy READ ipBusy NOTIFY changed)
 Q_PROPERTY(bool autoIP READ autoIP NOTIFY changed)
 Q_PROPERTY(bool ready READ ready NOTIFY changed)
 Q_PROPERTY(QString ipState READ ipState NOTIFY changed)
 Q_PROPERTY(QVariantMap systemCheck READ systemCheck NOTIFY changed)
 Q_PROPERTY(QVariantMap proxyCheck READ proxyCheck NOTIFY changed)
 Q_PROPERTY(QString systemIPState READ systemIPState NOTIFY changed)
 Q_PROPERTY(QString proxyIPState READ proxyIPState NOTIFY changed)
 Q_PROPERTY(bool systemIPBusy READ systemIPBusy NOTIFY changed)
 Q_PROPERTY(bool proxyIPBusy READ proxyIPBusy NOTIFY changed)
 Q_PROPERTY(QVariantMap modeStatus READ modeStatus NOTIFY changed)
 Q_PROPERTY(QString message READ message NOTIFY changed)
 Q_PROPERTY(QStringList sessions READ sessions NOTIFY changed)
 Q_PROPERTY(QVariantMap health READ health NOTIFY changed)
 Q_PROPERTY(QVariantMap check READ check NOTIFY changed)
 Q_PROPERTY(QVariantMap networkSettings READ networkSettings NOTIFY changed)
 Q_PROPERTY(QVariantMap baselineCheck READ baselineCheck NOTIFY changed)
 Q_PROPERTY(bool baselineIPBusy READ baselineIPBusy NOTIFY changed)
 Q_PROPERTY(QString baselineIPState READ baselineIPState NOTIFY changed)
 Q_PROPERTY(QVariantMap traffic READ traffic NOTIFY changed)
 Q_PROPERTY(QStringList interfaces READ interfaces NOTIFY changed)
 Q_PROPERTY(QString networkInterface READ networkInterface NOTIFY changed)
 Q_PROPERTY(QString macMessage READ macMessage NOTIFY changed)
 Q_PROPERTY(QString currentMAC READ currentMAC NOTIFY changed)
 Q_PROPERTY(bool sessionStartAvailable READ sessionStartAvailable NOTIFY changed)
 Q_PROPERTY(QString terminalMessage READ terminalMessage NOTIFY changed)
 Q_PROPERTY(QString terminalProgram READ terminalProgram NOTIFY changed)
 Q_PROPERTY(bool terminalAvailable READ terminalAvailable NOTIFY changed)
 Q_PROPERTY(bool terminalRunning READ terminalRunning NOTIFY changed)
public:
 explicit Workspace(QObject *parent=nullptr);
 ~Workspace() override;
 QString language()const{return m_preferences.value("language").toString("en");}
 bool torPreparing()const{return m_pendingMode=="proxy"&&m_busy;}
 QString mode()const{return m_preferences.value("mode").toString("system");}
 QString endpoint()const{return m_preferences.value("endpoint").toString("127.0.0.1:9050");}
 bool systemRoutingActive()const{return m_routingActive;}
 bool tmuxEnabled()const{return m_preferences.value("tmux_enabled").toBool(false);}
 bool busy()const{return m_busy;}
 bool connectionActive()const{return m_routingKnown&&!m_routingBusy&&((m_routingActive&&m_modeStatus.value("phase").toString()=="inactive")||(!m_routingActive&&m_modeStatus.value("phase").toString()=="active"));}
 bool ipBusy()const{return m_systemIP.busy||m_proxyIP.busy;}
 bool autoIP()const{return m_preferences.value("auto_ip").toBool(true);}
 bool ready()const{return m_loaded;}
 QString ipState()const{return m_lastProxy?m_proxyIP.state:m_systemIP.state;}
 QString systemIPState()const{return m_systemIP.state;}
 QString proxyIPState()const{return m_proxyIP.state;}
 bool systemIPBusy()const{return m_systemIP.busy;}
 bool proxyIPBusy()const{return m_proxyIP.busy;}
 QVariantMap systemCheck()const{return m_systemIP.result.toVariantMap();}
 QVariantMap proxyCheck()const{return m_proxyIP.result.toVariantMap();}
 QVariantMap modeStatus()const{return m_modeStatus.toVariantMap();}
 QString message()const{const auto split=m_message.indexOf('\n');return split<0?translate(m_message):translate(m_message.left(split))+m_message.mid(split);}
 QStringList sessions()const{return m_sessions;}
 QVariantMap health()const{return m_health.toVariantMap();}
 QVariantMap check()const{return m_lastProxy?proxyCheck():systemCheck();}
 QVariantMap networkSettings()const{return m_network.toVariantMap();}
 QVariantMap baselineCheck()const{return m_baselineIP.result.toVariantMap();}
 bool baselineIPBusy()const{return m_baselineIP.busy;}
 QString baselineIPState()const{return m_baselineIP.state;}
 QVariantMap traffic()const{return m_traffic;}
 QStringList interfaces()const{return m_interfaces;}
 QString networkInterface()const{return m_interface;}
 QString currentMAC()const;
 QString macMessage()const{const auto split=m_macMessage.indexOf('\n');return split<0?translate(m_macMessage):translate(m_macMessage.left(split))+m_macMessage.mid(split);}
 bool sessionStartAvailable()const{return m_loaded&&tmuxEnabled()&&mode()=="proxy"&&m_routingKnown&&!m_routingActive&&!m_routingBusy&&m_modeStatus.value("phase").toString()=="inactive"&&!m_busy&&!terminalRunning();}
 QString terminalMessage()const{const auto n=m_terminalMessage.indexOf('\n');return n<0?translate(m_terminalMessage):translate(m_terminalMessage.left(n))+m_terminalMessage.mid(n);}
 QString terminalProgram()const;
 bool terminalAvailable()const;
 bool terminalRunning()const{return m_terminalProcess&&m_terminalProcess->state()!=QProcess::NotRunning;}
 Q_INVOKABLE void changeMAC(bool restore);
 Q_INVOKABLE void installTerminal();
 Q_INVOKABLE void setRoutingState(bool known,bool active,bool busy);
 Q_INVOKABLE void selectInterface(const QString &name);
 Q_INVOKABLE void captureBaseline();
 Q_INVOKABLE void refreshBaseline();
 void automaticTick(qint64 now);
 void sampleTraffic(const QByteArray &data,qint64 now);
 Q_INVOKABLE QString translate(const QString &source)const;
 Q_INVOKABLE void load();
 Q_INVOKABLE void save(const QString &language,const QString &mode,const QString &endpoint,bool tmux);
 Q_INVOKABLE void refreshHealth();
 Q_INVOKABLE void switchMode(const QString &target);
 Q_INVOKABLE void refreshSessions();
 Q_INVOKABLE void sessionAction(const QString &action,const QString &name);
 Q_INVOKABLE void testProxy();
 Q_INVOKABLE void refreshIP(bool useProxy);
 Q_INVOKABLE void invalidateIP();
 Q_INVOKABLE void invalidateSystemIP();
 Q_INVOKABLE void invalidateProxyIP();
 Q_INVOKABLE void refreshModeStatus();
 Q_INVOKABLE void recoverProxy();
 Q_INVOKABLE void setAutoIP(bool enabled);
 Q_INVOKABLE void copyIP();
 Q_INVOKABLE QString presetCommand(int index) const;
 Q_INVOKABLE bool copyPresetCommand(int index);
 Q_INVOKABLE void exportDiagnostics();
 Q_INVOKABLE void applyNetworkSettings(const QString &bridge,bool autoBridge,bool safeSocks,bool killSwitch);
 Q_INVOKABLE void installTmux();
 Q_INVOKABLE void loadNetworkSettings();
 static QString cliPath();
signals:
 void changed();
 void languageChanged();
 void terminalAttachmentRequested(const QString &name);
 void terminalSessionStopped();
 void modeReady(const QString &mode);
 void terminalLaunchFailed(const QString &message);
private:
 friend class ModeTest;
 void completeModeSelection(const QString &target);
 void run(const QString &action,const QStringList &args);
 void privileged(const QString &action,const QString &program,const QStringList &args,const QByteArray &input={});
 void finish(const QString &action,int exitCode,const QByteArray &output);
 struct IPRequest {QJsonObject result;QPointer<QProcess> process;quint64 generation=0;bool busy=false;QString state="idle";QString previousIP;QString lastSuccessfulTimestamp;};
 void invalidate(IPRequest &request,bool resetComparison);
 void refreshRequest(int kind);
 IPRequest &ipRequest(int kind);
 void readTraffic();
 bool openExternalTerminal(const QString &name,bool create=false);
 void closeExternalTerminal();
 bool launchTerminal(const QString &program,const QStringList &arguments);
 void terminalFailure(const QString &summary,const QString &detail={});
 QJsonObject m_preferences,m_health,m_network,m_modeStatus;
 IPRequest m_systemIP,m_proxyIP,m_baselineIP;
 QTimer m_pollTimer;
 QElapsedTimer m_clock;
 bool m_routingKnown=false,m_routingActive=false,m_routingBusy=false,m_baselineFrozen=false;
 QString m_autoContext,m_interface;
 qint64 m_nextIP=0,m_trafficTime=-1;
 QStringList m_interfaces;
 QVariantMap m_traffic;
 quint64 m_rx=0,m_tx=0,m_rp=0,m_tp=0,m_firstRx=0,m_firstTx=0,m_firstRp=0,m_firstTp=0;
 bool m_lastProxy=false;
 QPointer<QProcess> m_modeProcess,m_terminalProcess;
 QStringList m_sessions;
 QJsonObject m_translations;

 bool m_loaded=false;

 bool m_busy=false;
 QString m_macMessage;
 QString m_message,m_pendingSession,m_pendingMode,m_terminalMessage;
};

QString WText(const QString &source);
