#!/usr/bin/env bash
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
version=${1:-1:1-2026101707}
[[ $version =~ ^[0-9]+:[0-9]+-[0-9]+$ ]] || exit 2
[[ -x $root/build/release/shadow21 && -x $root/build/gui/shadow21-gui ]] || { printf 'Build CLI and GUI first.\n' >&2; exit 1; }
stage="$root/build/package-$version"
[[ ! -e $stage ]] || { printf 'Package staging directory already exists.\n' >&2; exit 1; }
mkdir -p "$stage/DEBIAN" "$stage/usr/bin" "$stage/usr/lib/shadow21" "$stage/etc/shadow21" "$stage/usr/share/doc/shadow21" "$stage/usr/share/applications" "$stage/usr/share/icons/hicolor/scalable/apps" "$stage/lib/systemd/system/tor@default.service.d" "$stage/usr/share/polkit-1/actions"
install -m 755 "$root/build/release/shadow21" "$root/build/release/shadow21-dnstool" "$root/build/gui/shadow21-gui" "$stage/usr/bin/"
install -m 755 "$root/build/release/shadow21-helper" "$root/build/release/make-torrc" "$root/build/release/safekill" "$stage/usr/lib/shadow21/"
install -m 755 "$root/scripts/install.sh" "$stage/usr/lib/shadow21/shadow21-install"
install -m 600 "$root/configs/shadow21.cfg" "$root/configs/bridges.txt" "$root/configs/onion.pac" "$stage/etc/shadow21/"
install -m 644 "$root/sys-units/shadow21d.service" "$root/sys-units/shadow21-recovery.service" "$root/sys-units/shadow21-identity.service" "$stage/lib/systemd/system/"
install -m 644 "$root/sys-units/tor@default.service.d/shadow21-recovery.conf" "$stage/lib/systemd/system/tor@default.service.d/"
install -m 644 "$root/gui/org.shadow21.Desktop.desktop" "$root/gui/org.shadow21.Cli.desktop" "$stage/usr/share/applications/"
install -m 644 "$root/icons/shadow21.svg" "$stage/usr/share/icons/hicolor/scalable/apps/"
install -m 644 "$root/configs/org.shadow21.helper.policy" "$stage/usr/share/polkit-1/actions/"
install -m 644 "$root/LICENSE" "$root/LICENSE-GPL-3" "$root/NOTICE" "$root/README.md" "$stage/usr/share/doc/shadow21/"
install -m 644 "$root/debian/copyright" "$stage/usr/share/doc/shadow21/copyright"
cat > "$stage/DEBIAN/control" <<CONTROL
Package: shadow21
Version: $version
Section: net
Priority: optional
Architecture: amd64
Maintainer: shadow21 contributors
Depends: libc6 (>= 2.38), libstdc++6 (>= 13), tor, tmux (>= 3.2), xterm, iptables, network-manager, iproute2, systemd, sudo, polkitd, pkexec, libqt6core6 (>= 6.4) | libqt6core6t64 (>= 6.4), libqt6gui6 (>= 6.4) | libqt6gui6t64 (>= 6.4), libqt6network6 (>= 6.4) | libqt6network6t64 (>= 6.4), libqt6dbus6 (>= 6.4) | libqt6dbus6t64 (>= 6.4), libqt6widgets6 (>= 6.4) | libqt6widgets6t64 (>= 6.4), libqt6qml6, libqt6quick6, libqt6quickcontrols2-6, libqt6svg6, qt6-qpa-plugins, qml6-module-qtquick, qml6-module-qtquick-controls, qml6-module-qtquick-templates, qml6-module-qtquick-layouts, qml6-module-qtquick-shapes, qml6-module-qtquick-window, qml6-module-qtqml-workerscript
Suggests: xterm, nyx, bleachbit, obfs4proxy, snowflake-client
Description: Tor routing and privacy management with optional tmux
 English and German desktop interface and CLI. Host routing adapters
 require systemd and NetworkManager. Networking is experimental until
 validated in a disposable VM. No conventional VPN tunnel is provided.
CONTROL
printf '/etc/shadow21/shadow21.cfg\n/etc/shadow21/bridges.txt\n/etc/shadow21/onion.pac\n' > "$stage/DEBIAN/conffiles"
cat > "$stage/DEBIAN/preinst" <<'SCRIPT'
#!/bin/sh
set -eu
if [ -d /run/systemd/system ];then
 for u in anonsurfd.service wvpn21d.service shadow21d.service;do
  if systemctl is-active --quiet "$u";then printf 'Disconnect active routing before installing shadow21.\n' >&2;exit 1;fi
 done
fi
if [ -x /usr/bin/wvpn21 ] || [ -e /var/lib/wvpn21/install-torrc.json ];then
 printf 'Migrate wvpn21 explicitly before installing Shadow21; see docs/MIGRATION.md.\n' >&2;exit 1
fi
for p in /run/wvpn21/active /run/wvpn21-mode.json /var/lib/wvpn21/proxy-state.json /var/lib/wvpn21/firewall.owned /var/lib/wvpn21/torrc.backup /var/lib/wvpn21/dns.changed /run/anonsurf/active /run/anonsurf/iptables.rules /var/lib/anonsurf/torrc.backup;do
 [ ! -e "$p" ] || { printf 'Restore upstream managed state first.\n' >&2;exit 1; }
done
[ ! -e /var/lib/shadow21/proxy-state.json ] || { printf 'Stop or recover proxy mode before installation.\n' >&2;exit 1; }
if [ -x /usr/lib/shadow21/shadow21-helper ];then /usr/lib/shadow21/shadow21-helper recover;fi
SCRIPT
cat > "$stage/DEBIAN/postinst" <<'SCRIPT'
#!/bin/sh
set -eu
# dpkg configures Tor and tmux dependencies before this package.
command -v tmux >/dev/null || { printf 'tmux must be installed before configuration.\n' >&2;exit 1; }
command -v tor >/dev/null || { printf 'Tor must be installed before configuration.\n' >&2;exit 1; }
/usr/lib/shadow21/shadow21-helper install-proxy-config
if [ -d /run/systemd/system ];then
 systemctl daemon-reload
 systemctl enable shadow21-recovery.service
fi
printf 'shadow21 installed. Local SOCKS configuration saved; no Tor reload was requested. Launch the interface as a regular user.\n'
SCRIPT
cat > "$stage/DEBIAN/prerm" <<'SCRIPT'
#!/bin/sh
set -eu
if [ -d /run/systemd/system ];then
 systemctl stop shadow21d.service
 /usr/lib/shadow21/shadow21-helper recover
fi
# Upgrades retain installation settings and the first-install backup.
if [ "${1:-}" = remove ] || [ "${1:-}" = deconfigure ];then
 /usr/lib/shadow21/shadow21-helper uninstall-proxy-config
fi
if [ -d /run/systemd/system ] && { [ "${1:-}" = remove ] || [ "${1:-}" = deconfigure ]; };then
 systemctl disable shadow21d.service shadow21-recovery.service || true
fi
SCRIPT
cat > "$stage/DEBIAN/postrm" <<'SCRIPT'
#!/bin/sh
set -eu
if [ -d /run/systemd/system ];then systemctl daemon-reload;fi
SCRIPT
chmod 755 "$stage/DEBIAN/preinst" "$stage/DEBIAN/postinst" "$stage/DEBIAN/prerm" "$stage/DEBIAN/postrm"
mkdir -p "$root/dist"
dpkg-deb --build --root-owner-group "$stage" "$root/dist/shadow21_v1_amd64.deb"
