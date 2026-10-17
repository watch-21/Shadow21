# IP display review

A different address is not, by itself, evidence that either observation is incorrect. Firefox proxy/VPN settings, address family, different Tor circuits and the time of each request can produce different results.

Request handling:

- The original GUI used a separate blocking-style command lifecycle for IP checks, while Workspace had another checker. The original IP process was removed and both GUI controls now use one asynchronous implementation.
- Workspace previously held one result and cancellation generation for system and SOCKS checks. It now has independent requests, results, state, cancellation generations and timestamps.
- GUI response validation accepted arbitrary strings as IPs. Qt now validates/normalizes addresses; Go validates the external API response.
- A failed check clears the displayed current address and retains the last successful timestamp for context. Results invalidated by connection changes are marked stale and not shown as current.
- Proxy comparisons distinguish first observation, unchanged and changed. They compare two completed requests; they do not assert guaranteed identity rotation.
- Checks disable HTTP proxy environment handling, redirects and connection reuse, request uncached responses, and never retry SOCKS failures directly.

`shadow21 ip --json` exercises the system route. `shadow21 proxy test` explicitly uses SOCKS5 with remote hostname resolution. The checking service is https://check.torproject.org/api/ip; it provides both the observed address and whether that request was recognized as Tor. The manually copied ipify command uses a separate service and may use another circuit.

The original overview shows the selected/observed mode's IP. The tmux view always shows its independent SOCKS observation. A successful system check does not prove a proxy check, nor does either prove protection of every application. Existing system-mode post-lifecycle checks remain; manual checks disclose the external service before sending a request. Inspection mode sends no IP requests.

## Compare browser and application routes

In a disposable test VM, start one mode and record the GUI's scope and check time. Open https://check.torproject.org/ in Firefox and record the site's Tor status, address and time. Inspect Firefox Settings > Network Settings and any proxy/VPN extension; do not change them without recording their original values. Compare the same service and address family. In proxy mode Firefox will use its ordinary route unless explicitly configured for SOCKS; a tmux proxy session does not change Firefox.

For an independent proxy request:

```bash
curl --socks5-hostname 127.0.0.1:9050 https://check.torproject.org/api/ip
```

Even two correctly proxied requests can use different exit addresses. A one-second HUP loop is not an IP rotation mechanism and is not implemented.

## Desktop display policy

Manual IP buttons are enabled only for an observed active mode and write their results into that view. Stopped modes hide current observations and do not trigger automatic IP requests. An independent baseline checker is retained internally for diagnostic/regression use; it is not scheduled or displayed by the stopped desktop. Active system requests run every 15 seconds and active SOCKS requests every second, with cancellation and in-flight guards.
