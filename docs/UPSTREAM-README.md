# Build
- Build dependencies: cmake, g++, golang-go, git, qt6-base-dev, qt6-declarative-dev, qml6-module-qtquick-shapes
- Runtime dependencies: bleachbit, nyx, sudo, tor, iptables, network-manager, procps, systemd
- Commands to build:
1. `git clone https://gitlab.com/parrotsec/packages/anonsurf/`
2. `cd anonsurf && make build && sudo make install`
3. `make check`

# Source layout
- `src/cmd/`: Go command-line entry points.
- `src/internal/anonsurf/`: lifecycle orchestration and transactional recovery.
- `src/internal/dns/`, `firewall/`, `ipv6/`, `tor/`: network domains.
- `src/internal/command/`, `fileops/`, `systemd/`: operating-system adapters.
- `gui/`: Qt 6/QML interface for KDE Plasma.
- `gui/src/application/`: GUI startup and single-instance activation.
- `gui/src/anonsurf/`: GUI state and AnonSurf action orchestration.
- `gui/src/cli/`, `systemd/`: command-line and systemd D-Bus adapters.
- `gui/src/tray/`, `notifications/`: tray integration and desktop notifications.
- `icons/`: AnonSurf and Nyx logos retained for the future interface.

Interrupted sessions are recovered automatically at boot before Tor starts.
Lifecycle progress is published to systemd and shown by the interactive CLI.

The Qt interface stays unprivileged: it reads the CLI's JSON status and sends
authorized lifecycle requests through systemd D-Bus. Build it with `make gui`
and run `build/gui/anonsurf-gui`.

- List of default bridges are at https://gitlab.torproject.org/tpo/anti-censorship/team/-/wikis/Default-Bridges

# Connection handling
Existing direct TCP/UDP connections to public IPv4 destinations are blocked
when routing starts, except for Tor-owned traffic and the DHCP exception.
They are not migrated into Tor; TCP applications must reconnect.
Tor carries supported TCP traffic and DNS lookups, not arbitrary UDP.
UDP DNS requests use Tor's UDP-only DNSPort. TCP DNS requests to public
IPv4 resolvers use TransPort, following the same routing as other TCP traffic.
Private-network and loopback routing exceptions remain in effect.

# Thank to
- Vivek Gite, who wrote tutorial of hook script for dhclient https://www.cyberciti.biz/faq/dhclient-etcresolvconf-hooks/