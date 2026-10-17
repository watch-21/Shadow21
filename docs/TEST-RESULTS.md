# Validation

Local checks use Go 1.24.6 and Qt 6.4.2. Build/test operations do not change active host networking or delete browser data.

| Check | Result and scope |
|---|---|
| All Go package tests and go vet | Passed; command fixtures and isolated state |
| SafeKill preflight/allowlist and stage ordering | Passed; missing cleaners never trigger application closure, obsolete cleaner IDs are filtered, failed closure/cancellation prevents deletion |
| Cleaner cancellation | Passed with an owned helper and descendant heartbeat; the descendant stopped writing after timeout |
| Real selected-application closure | Skipped because the test PID namespace does not match the mounted process table; production refuses this configuration before signaling |
| Stalled tmux management command | Passed; owned session service cleanup continued within the bounded wait and borrowed Tor remained untouched |
| SOCKS destination deadline/cancellation | Passed with in-memory sockets; the greeting deadline is reset and cancellation closes the socket |
| Qt lifecycle, terminal, IP and telemetry fixtures | Passed; includes canceled cleanup, stale cancellation-timer isolation, retained MAC messages, mode guards, generation cancellation and inactive automatic-check suspension |
| Workspace translation lifetime | Passed; releasing a workspace clears the shared translation reference before later cleanup callbacks |
| Source text scan | Passed; no Arabic-script characters in source text, English application identifiers/comments, German runtime translations retained |
| Fixed clipboard snippets | Passed; exact strings preserved in the Qt clipboard |
| Desktop controls/localization | Passed; four system actions, no desktop/tray Tor Monitor, 403 consistent translation entries |
| UI rendering | Twenty EN/DE regular/compact/dialog layouts and four active-mode fixture layouts passed; selected running layouts visually inspected. Active layouts and relocated loading rechecked after compact spacing, automatic MAC display and status-word-wrapping changes |
| Package maintenance/migration/CLI | Passed with isolated command/path fixtures and CLI smoke checks |
| Live Tor/systemd/Polkit/root tmux, actual MAC and browser cleanup | Unverified; requires disposable desktop VMs |
| Current distribution integration | Unverified; see SUPPORT.md |

Reproduce with make check, make gui-tests, tests/ui_render.py, tests/ui_active.py, tests/cli_smoke.py, tests/package_install.py and tests/package_preinst.py after building. Set QT_QPA_PLATFORM=offscreen for headless checks. No live browsing, active adapter change or application-profile deletion is part of the automated suite.

Release validation also compares packaged GUI/CLI/helper binaries with current outputs and retains legal files and third-party notices unchanged. Source and unpacked package scans check known private-value markers and build-environment paths. Raw build logs are not distributed. These targeted checks are not a complete security audit or a guarantee against every possible sensitive value.
