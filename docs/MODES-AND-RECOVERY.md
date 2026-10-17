# Connection modes and recovery — Shadow21 v1

The application has mutually exclusive system routing and proxy-session modes. The two mode buttons offer cleanup first. Authorization denial or restoration failure retains the old selection and blocks the new start. Selecting tmux starts the system Tor service, but does not create a session.

System routing preserves existing lifecycle behavior, supported transparent TCP/DNS rules, managed chain ownership and IPv6 restoration. Proxy mode never creates device routing or DNS rules. Both starts validate ownership journals and share the lifecycle lock. Old wvpn21 state is refused until its own helper restores it.

## tmux

The GUI uses an automatically numbered root-owned session, initially `0`. Legacy named sessions are accepted for recovery. The backend starts a dedicated `tmux -D -f /dev/null -S /run/shadow21/0.sock` server in its own systemd cgroup, creates `new-session -d` and attaches the authorized external terminal client. This has the session behavior of `tmux new`; detached creation keeps ownership/cleanup independent of client authorization. No further shell commands are injected. Detach preserves it. Root permissions and SOCKS variables do not enforce proxy routing for applications.

Stop executes `kill-session` on that exact dedicated socket, stops the owned cgroup, verifies the unit/session are gone and emits terminal closure. Detached Stop works through the same backend. A root user may intentionally escape its cgroup; this is not a root security sandbox. No general tmux kill-server is used. The GUI opens an independent xfce4-terminal, Konsole or xterm process, without title overrides. Start prepares and attaches through one authorized `session-console` helper. Stop closes only the tracked emulator process after successful backend cleanup. Closing or detaching a terminal does not switch modes.

If Tor was already running, the listener must be exactly loopback, the executable must match the installed tor, its PID must match a supported active Tor unit and an HTTPS proxy check must report Tor. Cleanup does not stop or modify the borrowed service. Otherwise an owned client uses separate runtime/data/control paths, drops to the Tor account and waits for authenticated bootstrap 100%, SOCKS readiness and verified binding. Its Tor configuration changes are backed up and hash-guarded.

Installation's SOCKS baseline persists through Stop. Uninstall restores the first-install baseline only when content hashes allow it. `SocksPolicy accept *` is unnecessary for local tmux; it is retained on a loopback-only listener. Unsupported includes/relay/listener configurations are refused.

## Recovery

```bash
shadow21 tmux status
shadow21 tmux recover
sudo /usr/lib/shadow21/shadow21-helper recover
```

Recovery stops owned session/cgroup/Tor resources, restores managed configuration and removes completed ownership state. If current Tor configuration has been edited externally, it is preserved. Keep `/var/lib/shadow21/proxy-torrc.backup`, `/var/lib/shadow21/proxy-state.json` and the current file; reconcile the reported changes from a local console and retry. Installation backup/journal are separate (`install-torrc.backup`, `install-torrc.json`) and survive Stop/upgrades. Do not purge journals to force switching.

Mode transitions, root authorization, cgroup cleanup, failure/rollback and preservation of unrelated sessions require the disposable desktop VM checklist. Local fixture passes are not proof of successful host routing or a Tor exit. See TEST-RESULTS.md and SUPPORT.md.

## MAC recovery

The shared MAC controls require both managed modes stopped. Only supported physical Ethernet/Wi-Fi interfaces and an active, non-shared NetworkManager profile are accepted. The previous observed MAC and cloned-address policy are recorded in root-only `/var/lib/shadow21/mac/<interface>.json`. Runtime profile changes use `nmcli connection modify --temporary`; restoration reconnects with the previous observed address, verifies it, then reinstates the original runtime policy without another reconnect.

```bash
shadow21 mac restore eth0
shadow21 mac recover
sudo /usr/lib/shadow21/shadow21-helper recover
```

Replace `eth0` with the actual interface. Restoring can interrupt connectivity; use a local console. Failed rollback keeps its journal and blocks new managed connections until recovery. An externally changed connection or cloned-address policy is preserved and requires reconciliation with the reported journal. Do not delete journals to bypass this protection. After a reboot, a vanished temporary override is recognized rather than replaying an old runtime address. Ordinary routing Stop leaves a successfully applied MAC override in place; explicit recovery, removal and upgrade restore it.

Live driver and NetworkManager behavior, desktop authorization and terminal closure require a disposable VM. Fixture tests are not a substitute.
