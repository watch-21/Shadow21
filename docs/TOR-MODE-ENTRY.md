# Tor service start before opening tmux — Shadow21 v1 / 20261016-01


## Behavior

The tmux mode button now performs asynchronous authorized preparation before changing the visible view or persisting the selection. Successful cleanup of the previous mode precedes preparation. The scoped helper starts `tor.service`; Debian-family adapters also start `tor@default.service`, whose main process provides the service behind Debian's wrapper. Only fixed known unit names are accepted. It checks the actual active service, positive MainPID and local SOCKS binding. Supported current Tor configuration is checked before starting; no configuration, routing rules or DNS settings are rewritten by preparation.

The recovery unit is started before the caller takes the lifecycle lock, because that unit's own helper needs the same lock. Guards/preflight are repeated after acquiring the lock. Active/unfinished modes, unsupported configuration/layout, missing dependencies, failed service start, missing PID, unsafe/missing listener, failed recovery, malformed success or denied authorization block mode entry. The previous view remains selected.

Successful preparation returns structured verified status. Only then are proxy preferences saved and `modeReady` emitted to the GUI. Selecting tmux creates no terminal or tmux session. Start session still controls that separate operation. Its existing authenticated process/SOCKS/HTTPS checks remain authoritative for a running session. Merely starting Tor or verifying a bound port is not proof of bootstrap completion, a successful exit request, or protection for the entire host.

Inactive saved proxy preferences do not trigger service startup when opening the app; the user clicks tmux. Re-clicking an inactive tmux selection can prepare it again. An already active owned session may be viewed directly without restarting it. OS Tor service startup is an explicit user action and is intentionally retained after leaving the view/Stop; existing cleanup only stops owned session/private Tor resources. No stop/reload/enable is performed on the system Tor service by the preparation helper.

## Verification

Fixed service command ordering, listener validation, mode guards, authorization failure and UI selection timing are covered by command fixtures. See TEST-RESULTS.md. Live systemd/Tor/Polkit and host networking integration require disposable desktop VMs.

## Reproduce on a disposable desktop VM

Start from stopped routing/session, record `systemctl is-active tor@default.service`, then select tmux. Verify the authorization request, background service start and active service before the view changes. Confirm no owned tmux session/terminal exists until Start session. Repeat with Tor already active, denied authorization, a masked/failed unit, occupied/nonlocal listener, incomplete recovery and repeated clicks. Keep unrelated processes/sessions alive. After leaving the view/Stop verify system Tor remains running and unrelated settings remain intact. An external service-start job may continue after a client timeout, as with systemctl; the app must leave selection unchanged and report failure rather than claim success.

Test upgrade, fresh installation/recovery dependency startup, system-to-proxy cleanup ordering, rollback/refusal and recovery from failed prior mode cleanup. Preserve journals on errors; use the existing recovery documentation rather than deleting ownership state.

