# Required host validation before operational use

Run these in disposable VMs with snapshots and a local console. Do not test host routing through an SSH-only connection. Record distribution/version, kernel, Tor version, firewall backend, resolver ownership and exact results.

1. Build and install on Kali, Parrot, Debian and Ubuntu. Test Fedora/Arch/openSUSE proxy-only startup separately. Verify unsupported host routing fails before any change.
2. Save iptables-save, IPv6 sysctl values, /etc/resolv.conf link/contents, Tor configuration and NetworkManager configuration. Include unrelated rules/counters in the fixture.
3. Start routing. Verify authenticated Tor bootstrap reaches 100%, DNS and TCP use expected routing, arbitrary UDP is blocked with protection enabled, IPv6 is disabled and private-network exceptions behave as documented.
4. Observe the firewall from a privileged local console. Verify existing chains/policies/rules remain and only WVPN21 chains/jumps were added. Verify unrelated INPUT/FORWARD policy remains untouched.
5. Stop Tor while routing remains active. Confirm public non-exempt traffic cannot fall back directly. Restore Tor. Restart routing and confirm no protection gap. Repeat with pre-existing direct connections.
6. Test explicit Kill Switch disable, then re-enable on the next connection. Confirm UI messages and behavior match. Intentional disconnect should restore the original network.
7. Inject failures after Tor configuration, IPv6 change, filter insertion, NAT insertion and DNS mutation. Confirm rollback, retained recovery data on rollback failure and successful later recover.
8. Terminate the helper during each phase; recover from a local console. Reboot with an interrupted session and verify persistent state restoration before Tor starts. Do not claim a boot-time privacy guarantee beyond those tests.
9. Start/stop/restart repeatedly and concurrently. Confirm serialization and consistent state. Verify missing dependencies and conflicting AnonSurf/VPN/native nft/firewall-manager state are refused before changes.
10. Test manual obfs4/plain bridges, automatic bridge list, missing transport dependencies and invalid settings. Confirm configuration backups, preserved credentials and restoration.
11. As a desktop user, select proxy mode and enable tmux. Test denied/canceled authorization, then authorized root start/open/detach/stop. Confirm the private socket is root-only and the shell id is root. Confirm only one owned session is allowed. Close/reopen the GUI, verify actual observed service state, and preserve an unrelated user tmux session throughout. Leave tmux disabled and prove no session/server starts.
12a. Inside the owned session run an ordinary nohup sleep descendant and record its PID/cgroup. Detach and Stop from the GUI; verify the descendant, private server, attachment client and dedicated terminal window are gone. Keep an unrelated terminal window and a general tmux session open and verify both survive.
12b. Try system start while proxy mode is active or failed, and proxy start while system mode is active/interrupted. Both GUI and CLI must reject. Concurrent requests must serialize. Verify Stop restores before the selector allows switching.
12c. Record torrc bytes, mode and any symlink before proxy start. Verify the effective single listener is 127.0.0.1:9050, authenticated bootstrap reaches 100% and no DNS/transparent listeners are created by proxy mode. Test existing active Tor/occupied port, unsupported includes/listener configuration, invalid config, startup timeout and canceled startup. They must refuse or roll back safely.
12d. Edit torrc externally during each mode and attempt Stop/recovery. Verify the external file/symlink is preserved, backup retained and mode switching remains blocked. Resolve locally and retry recovery. Reboot during each startup/stop phase; verify persistent journals recover without relying on transient unit presence.
12e. Exercise first/same/changed proxy IP observations, malformed responses, failures, timeouts, cancellation and delayed responses. Verify system IP is untouched. Compare Firefox using the same service/address family and recorded proxy settings; distinguish circuit variation from a UI defect.
12. Test curl with explicit SOCKS hostname resolution. Confirm a stopped proxy fails that request. Demonstrate that an application ignoring ALL_PROXY can connect directly in proxy-only mode; do not market it as enforced isolation.
13. Exercise NEWNYM cooldown and Tor rejection. Verify existing connections remain unchanged and no different-IP guarantee is displayed.
14. Test English/German UI, dialogs, notifications, tray menu, CLI, keyboard focus, resize, scrolling and real terminal attachment.
15. Upgrade with customized config. Uninstall from both source and .deb installations. Confirm only owned files are removed, network restored and user data/unrelated sessions preserved.

Capture actual evidence. Mark each unexecuted check as untested, not passed.

16. On a disposable VM with Tor absent, install dependencies/app and confirm Tor is available before SOCKS preparation. Record any service startup performed by the distribution Tor package itself. Verify exactly one copy of both directives, protected original backup and no shadow21-requested reload. Stop both modes and verify the installed SOCKS baseline persists. Reinstall/upgrade and verify the first backup remains unchanged. Remove and verify original bytes/absence, owner and permissions. Inject invalid configuration, advanced listener/include conflicts, external edits and missing Tor; failure must preserve the existing file.

## 1.2.0 checks

Verify installation with neither, either, and both Tor/tmux already present; only missing packages are installed. Use a regular desktop user and confirm the automatic numeric session is on the private root socket. Keep an unrelated tor-named tmux session on the user's general server and verify Stop preserves it. Test terminal closure after detach and survival of unrelated windows.

While stopped in either selected mode, verify IP controls are disabled, observations are hidden and no automatic external IP requests occur. Start a mode, verify its IP button is enabled and writes the result into the interface, then stop and verify invalidation/hiding. Measure active request frequency: SOCKS approximately every second and system routing approximately every 15 seconds, without overlap. Disable automatic checks and confirm manual checks remain available only during an active mode. Tor failure must not produce a direct proxy retry.

Compare the interface byte/packet totals with ip -s link and /proc/net/dev under controlled traffic. Test no-traffic rates, interface selection persistence, removal/recreation, counter resets, window resize and long German text. Confirm unrelated interface traffic is counted and the panel describes interface-wide totals. This monitor is not an Internet speed benchmark.

## Current external-terminal and MAC checks

Test xfce4-terminal, Konsole and xterm separately: start/attach from a regular-user desktop, verify root inside the dedicated tmux session and the emulator's default title, then detach and Stop. Test graphical and terminal Polkit agents, denied/canceled authorization, missing emulator, and occupied/unverified SOCKS listeners. Keep unrelated emulator windows and general tmux sessions alive. Verify both mode buttons work after success and failure and after restarting the GUI.

For physical/VM hardware Ethernet and Wi-Fi profiles, record the observed MAC and cloned-address policy, randomize while both modes are stopped, verify the observed address, then restore and compare both fields. Test multi-connect/VPN/virtual-interface refusal, rollback on activation error, external profile changes, interrupted recovery, reboot, upgrade and removal. Do not do this over a remote connection that the test may disconnect.

## Bounded cleanup checks

Run SafeKill from a normal desktop account in a disposable VM. Verify missing BleachBit or an empty supported cleaner list fails before closing apps. Open disposable browser profiles with no valuable data; test graceful closure, refusal to delete cache while a selected app stays alive, cancellation during each phase, and no surviving cleaner subprocesses. A failed cleanup must block Start/Restart, but networking Stop must still execute with a warning. Verify no other account's processes/profiles are affected. Test tmux Stop with detached sessions and stubborn owned descendants; only the private socket and owned cgroup may be removed. Verify MAC messages survive background status refreshes and no Tor Monitor action appears in either view or the tray.
