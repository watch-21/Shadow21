# MAC and layout corrections — v1, internal build 20261017-01

## Findings

The previous Apply path refused any existing MAC recovery journal. Therefore a second randomization required manual restoration. The Restore path compared NetworkManager literal MAC strings case-sensitively; an uppercase rendition of the application's own address could be rejected as an external modification. Equivalent upper/lowercase policy addresses are covered by command fixtures. Live adapter behavior requires integration testing.

The confirmation dialog allocated only 170 pixels for its translated action. Long German labels overflowed. Status details and timestamps used small fonts and elision.

## Changes

Literal MAC policies are canonicalized through net.ParseMAC. Symbolic policies retain their meaning. Repeated randomization by the journal owner or root first uses the existing guarded restore transaction, then applies a new, distinct address with a fresh journal. The original runtime address and original policy remain the restore target. This can briefly reconnect the interface twice. If restoration fails, no new change is attempted; the recovery journal remains. A failed new activation rolls back to the original settings. Interrupted operations, different owners and external policy/connection changes are rejected. Existing lifecycle locking and mode/VPN guards remain in effect.

Confirmation buttons use a responsive grid with stacked actions at narrow widths. Button labels wrap within the button and its height expands. Status text wraps instead of truncating, with larger detail fonts and two status columns on narrower windows. The main header displays Shadow21 · v1; the internal build identifier remains in About and installation metadata. The three fixed copy cards remain available.

## Recovery and verification

Stop both application connection modes before changing MAC. Use Restore previous MAC for a saved operation, including failed operations. If the helper still reports an external-policy conflict, do not delete its journal or force-write a network profile: compare the selected connection and saved state with current NetworkManager settings before deciding which external edits to preserve. A real external edit is intentionally not bypassed by this update.

Automated command fixtures cover Ethernet/Wi-Fi repeated randomization, uppercase policy restoration, original restoration, second-operation failure rollback, different users, external policies/connections and interrupted state. No live network interface is changed by these tests. Desktop integration on Kali/VirtualBox, other distribution VMs, and actual MAC behavior of its adapter remain unverified here.
