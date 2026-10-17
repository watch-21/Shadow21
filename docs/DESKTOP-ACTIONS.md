# Desktop actions and bounded cleanup

IP checks are displayed only while the selected managed mode is active. The selected interface MAC is displayed automatically in both views, including when modes are stopped. The manual IP control updates the interface without a separate confirmation/result dialog; its label and existing service disclosure identify the external check. Stopped modes hide IP observations and suspend automatic external checks. MAC changes and restoration still require both managed connection modes stopped.

The desktop and tray no longer offer Tor Monitor. CLI monitoring remains available for compatibility. Four labeled system actions use a responsive two/four-column grid instead of narrow icon-only controls. Both views share observed network state, traffic counters and MAC controls.

SafeKill is scoped to selected applications owned by the invoking desktop account. It checks BleachBit and supported cleaner names before requesting application closure, verifies that selected processes finish closing, and skips obsolete cleaner IDs. If applications remain alive, cache deletion is not started. It does not terminate every system process or unrelated users' applications. Root invocation is refused. A mismatched mounted process namespace is refused before signals are sent.

Cleanup has a 25-second overall budget, a five-second cleaner-list check and a three-second application-closing wait. The GUI exposes Cancel cleanup and retains timeout/cancellation reasons. Cleaner subprocess groups are terminated when their context ends; raw cleaner output is discarded because it can contain personal paths. Failed cleanup blocks Start/Restart, but never prevents the requested networking Stop from running. A warning is retained in the operation result if Stop proceeds after cleanup failure.

The optional root tmux Stop path remains limited to its dedicated socket and owned service/cgroup. Individual tmux management commands have three-second deadlines; session termination has a twenty-second budget. Failure keeps recovery state and does not report successful cleanup or affect unrelated tmux servers.

SOCKS destination setup receives a fresh twenty-second deadline after the initial greeting, bounded by the request context. Cancellation closes the owned socket. This avoids accidentally reusing the short greeting deadline for circuit creation/remote DNS, without adding direct fallback.

Verification uses command fixtures, owned helper processes, in-memory sockets and offscreen EN/DE GUI rendering. Actual application closure is skipped when the test process PID namespace differs from the mounted process table. Full desktop, Tor, driver/MAC and systemd/cgroup integration still requires disposable VM testing. See TEST-RESULTS.md.

A manual IP button remains available during an in-flight automatic check while the mode is active. A new manual request cancels/replaces the old generation; late responses cannot overwrite the new result. Lifecycle changes still disable the control. MAC operation messages remain separate from periodic health/status messages.
