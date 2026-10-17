# Architecture

Shadow21 extends ParrotSec AnonSurf. Upstream repository, branch and revision are recorded in UPSTREAM.json; legal notices are retained in NOTICE and debian/copyright.

The backend and CLI use Go 1.24 with vendored Cobra/pflag. The desktop uses C++17, Qt 6 and QML. The GUI runs unprivileged and reads CLI JSON. Fixed authorized helper actions perform networking, MAC changes and optional root tmux lifecycle operations. A shared lifecycle lock serializes changes.

System routing uses owned firewall chains, DNS and IPv6 recovery records. Proxy sessions use a dedicated root tmux socket and systemd cgroup. Restoration is journaled and refuses conflicting external changes. The desktop maintains independent asynchronous system, baseline and SOCKS observations, validates IP addresses and rejects stale generations. Manual identity control uses Tor's authenticated control interface.

Distribution detection reads /etc/os-release. Platform adapters and preflight checks define support boundaries; their presence does not establish integration compatibility. See SUPPORT.md and VM-VALIDATION.md.
