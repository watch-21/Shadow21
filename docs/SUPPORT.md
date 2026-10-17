# Platform support

| Platform | Proxy/session mode | System routing | Current verification |
|---|---|---|---|
| Kali, Parrot, Debian, Ubuntu | Experimental adapter | Experimental adapter with strict preflight | Build, command fixtures and offscreen Qt checks; current full desktop integration unverified |
| Fedora, Arch, openSUSE | Experimental adapter | Unsupported; refused before mutation | Current distribution integration unverified |
| Other distributions | No compatibility claim | Unsupported | Unverified |

Managed modes require systemd, supported Tor service/account layout, iproute2, tmux, a supported terminal, desktop D-Bus and Polkit authorization. System routing and MAC controls additionally require supported NetworkManager and firewall/DNS ownership. Active VPNs and incompatible firewall managers are refused. Native nftables configurations and general UDP tunneling are not supported by system routing.

The prebuilt amd64 Debian package uses an Ubuntu 24.04 ABI: glibc >= 2.38, libstdc++ >= 13 and Qt >= 6.4. Older systems and other architectures need local builds. Dependency resolution requires configured distribution repositories. Do not mix repositories from different distributions or force missing dependencies.

No current full-distribution compatibility pass is claimed. Offscreen rendering and command fixtures do not establish live routing, root terminal attachment, Kill Switch behavior, clipboard integration or actual MAC changes. Use VM-VALIDATION.md in disposable desktop VMs.

SafeKill additionally requires a normal desktop account, BleachBit with supported cleaner IDs, and a process table matching the running PID namespace. It does not run as root or clean another account. A disposable desktop VM is required to verify actual browser closure/profile cleanup.
