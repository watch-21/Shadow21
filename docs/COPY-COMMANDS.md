# Fixed tmux command cards — v1 / 20261016-03

The tmux view has three read-only command/snippet cards. Each Copy button copies the exact text to the desktop clipboard and briefly shows Copied. Copying does not open a terminal, request privileges, send input to a session, or execute a command. The cards remain available when a session is stopped. English and German labels follow the selected language.

```sh
while true; do pkill -HUP tor; sleep 1; done
```

```sh
curl --socks5-hostname localhost:9050 https://api.ipify.org/
```

The third card copies the proxy URL exactly:

```text
socks5://127.0.0.1:9050
```

HUP reloads Tor configuration and does not guarantee another exit IP. If manually executed, the first command signals all permitted matching Tor processes repeatedly; Ctrl+C stops that loop. The second command, when manually executed, sends an external IP request to api.ipify.org with hostname resolution through SOCKS. These cards do not change the existing automatic IP-check workflow.

Verification: the Qt offscreen clipboard test checks byte-for-byte command values, clipboard content, rejected invalid indices, and absence of session/terminal actions. EN/DE interface renders are inspected for layout. A real X11/Wayland clipboard paste into a desktop terminal remains a desktop integration check; offscreen clipboard success alone does not verify that path.

Build and run the clipboard test with the regular Qt development environment:

```sh
cmake --build build/gui --target shadow21-clipboard-test
QT_QPA_PLATFORM=offscreen build/gui/shadow21-clipboard-test
```
