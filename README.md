<h1 align="center">Shadow21 v1</h1>

<p align="center"><img src="icons/shadow21.svg" alt="Shadow21 logo" width="240"></p>

A Linux Tor routing manager with a compact desktop interface in **English and German**. Initial public release, based on ParrotSec AnonSurf.

## Features

- **System mode:** manage supported system traffic routing through Tor.
- **tmux mode:** open an external root terminal session for applications configured to use Tor SOCKS.
- **IP monitoring:** check the proxy IP every second and the system-route IP every 15 seconds while active.
- **MAC controls:** display, randomize and restore the selected adapter's MAC address.
- **Network activity:** view transfer rates, packet counts and transferred data.
- **Connection controls:** Start, Stop, Restart, New Identity, Refresh, startup integration and optional SafeKill.

Only one connection mode can run at a time. Detaching keeps a tmux session running; **Stop session** terminates the owned session and closes its tracked terminal.

## Demo

https://github.com/user-attachments/assets/3de279dd-2368-4211-bda3-ad3e183d2172

Silent 2:13 animation; IP/MAC values and terminal responses labeled **Demo data** are fictional.

## Install and launch

[Download Shadow21 v1 (.deb)](https://github.com/watch-21/Shadow21/releases/download/v1/shadow21_v1_amd64.deb)

Install the downloaded `.deb` package on a compatible amd64 Debian-family system, then run from its download directory:

```bash
sudo apt install ./shadow21_v1_amd64.deb
shadow21-gui
```

Run the GUI as your regular desktop user. Privileged actions request system authorization. For a source build, see [build instructions](docs/BUILD.md).

## Common commands

```bash
shadow21 status
shadow21 start
shadow21 ip --json
shadow21 stop
```

With system routing stopped, use the GUI's tmux controls or see the [session and recovery guide](docs/MODES-AND-RECOVERY.md). The local SOCKS endpoint is `127.0.0.1:9050`. Check a request from a separate terminal with:

```bash
curl --socks5-hostname 127.0.0.1:9050 https://api.ipify.org/
```

## Important behavior

Proxy monitoring checks the IP every second; it does not guarantee a different exit address. tmux itself does not rotate IPs, and the HUP copy-card command reloads Tor configuration. Each application needs supported proxy options. Shadow21 is not a conventional VPN and does not provide general UDP tunneling or complete anonymity.

MAC changes require both modes stopped and a supported NetworkManager adapter. A MAC change does not itself change the public IP.

Kali, Parrot, Debian and Ubuntu have experimental adapters. Other platform support and unverified live networking behavior are documented in the [support matrix](docs/SUPPORT.md) and [test results](docs/TEST-RESULTS.md). Managed modes require systemd and compatible host networking.

## Origin and licensing

Shadow21 modifications by **watch-21**. Derived from [ParrotSec AnonSurf](https://gitlab.com/parrotsec/packages/anonsurf). Upstream copyrights, licenses and third-party notices are preserved. See [NOTICE](NOTICE), [debian/copyright](debian/copyright), [LICENSE](LICENSE) and [LICENSE-GPL-3](LICENSE-GPL-3).
