#!/usr/bin/env bash
set -euo pipefail
PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PATH
unset CDPATH BASH_ENV ENV
lang=${SHADOW21_LANG:-en}
say(){ if [[ $lang == de ]];then printf '%s\n' "$2";else printf '%s\n' "$1";fi; }
fail(){ say "$1" "$2" >&2;exit 1;}
[[ $EUID == 0 ]] || fail 'Run this installer with sudo.' 'Dieses Installationsprogramm mit sudo starten.'
action=${1:-install}
[[ $action == install || $action == uninstall || $action == tmux || $action == terminal || $action == dependencies ]] || exit 2
[[ -f /etc/os-release ]] || exit 2
source /etc/os-release
family=''
for candidate in "$ID" ${ID_LIKE:-};do
 case "$candidate" in
 debian|ubuntu|kali|parrot) family=debian;;
 fedora|rhel) family=fedora;;
 arch) family=arch;;
 opensuse*|suse) family=suse;;
 esac
 [[ -z $family ]] || break
done
[[ -n $family ]] || fail 'Unsupported distribution.' 'Nicht unterstützte Distribution.'
install_packages(){
 case "$family" in
 debian) apt-get update; apt-get install -y --no-install-recommends "$@";;
 fedora) dnf install -y "$@";;
 arch) pacman -S --needed "$@";;
 suse) zypper --non-interactive install "$@";;
 esac
}
if [[ $action == terminal ]];then command -v xterm >/dev/null || install_packages xterm;exit 0;fi
if [[ $action == tmux ]];then command -v tmux >/dev/null || install_packages tmux;exit 0;fi
if [[ $action == dependencies ]];then
 case "$family" in
 debian) install_packages build-essential cmake qt6-base-dev qt6-declarative-dev qml6-module-qtquick qml6-module-qtquick-controls qml6-module-qtquick-templates qml6-module-qtquick-layouts qml6-module-qtquick-shapes qml6-module-qtquick-window qml6-module-qtqml-workerscript libqt6svg6 tor tmux iproute2 iptables network-manager policykit-1;;
 fedora) install_packages gcc-c++ cmake qt6-qtbase-devel qt6-qtdeclarative-devel qt6-qtsvg tor tmux iproute iptables NetworkManager polkit;;
 arch) install_packages base-devel cmake qt6-base qt6-declarative qt6-svg tor tmux iproute2 iptables networkmanager polkit;;
 suse) install_packages gcc-c++ cmake qt6-base-devel qt6-declarative-devel qt6-svg-devel tor tmux iproute2 iptables NetworkManager polkit;;
 esac
 say 'Install Go 1.24+ separately if the distribution package is older. Optional: nyx, bleachbit, obfs4proxy, snowflake-client.' 'Go 1.24+ separat installieren, falls das Distributionspaket älter ist. Optional: nyx, bleachbit, obfs4proxy, snowflake-client.'
 exit 0
fi
[[ -d /run/systemd/system ]] || fail 'System installation requires systemd.' 'Systeminstallation erfordert systemd.'
for unit in anonsurfd.service wvpn21d.service shadow21d.service;do
 if systemctl is-active --quiet "$unit";then fail 'Disconnect active routing before installation or removal.' 'Aktives Routing vor Installation oder Entfernung trennen.';fi
done
if [[ $action == install ]] && { [[ -x /usr/bin/wvpn21 ]] || [[ -e /var/lib/wvpn21/install-torrc.json ]]; };then
 fail 'Migrate wvpn21 explicitly first; see docs/MIGRATION.md.' 'Zuerst wvpn21 migrieren; siehe docs/MIGRATION.md.'
fi
if [[ $action == install && -e /var/lib/shadow21/proxy-state.json ]];then
 fail 'Stop or recover proxy mode before upgrading.' 'Proxy-Modus vor dem Upgrade stoppen oder wiederherstellen.'
fi
if [[ $action == uninstall ]];then
 /usr/lib/shadow21/shadow21-helper recover
 /usr/lib/shadow21/shadow21-helper uninstall-proxy-config
 systemctl disable shadow21d.service shadow21-recovery.service 2>/dev/null || true
 rm -f /usr/bin/shadow21 /usr/bin/shadow21-gui /usr/bin/shadow21-dnstool
 rm -f /usr/share/applications/org.shadow21.Desktop.desktop /usr/share/applications/org.shadow21.Cli.desktop
 rm -f /usr/share/icons/hicolor/scalable/apps/shadow21.svg
 rm -f /lib/systemd/system/shadow21d.service /lib/systemd/system/shadow21-recovery.service /lib/systemd/system/shadow21-identity.service
 rm -f /lib/systemd/system/tor@default.service.d/shadow21-recovery.conf
 rm -f /usr/share/polkit-1/actions/org.shadow21.helper.policy
 rm -f /usr/lib/shadow21/shadow21-helper /usr/lib/shadow21/make-torrc /usr/lib/shadow21/safekill /usr/lib/shadow21/shadow21-install
 rm -f /usr/share/doc/shadow21/LICENSE /usr/share/doc/shadow21/LICENSE-GPL-3 /usr/share/doc/shadow21/NOTICE /usr/share/doc/shadow21/README.md
 rmdir /usr/share/doc/shadow21 2>/dev/null || true
 rmdir /usr/lib/shadow21 2>/dev/null || true
 systemctl daemon-reload
 say 'Removed owned application files after recovery. User preferences and unrelated sessions are preserved.' 'Eigene Programmdateien nach Wiederherstellung entfernt. Benutzereinstellungen und fremde Sitzungen bleiben erhalten.'
 exit 0
fi
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
[[ -x $root/build/release/shadow21 && -x $root/build/release/shadow21-helper && -x $root/build/gui/shadow21-gui ]] || fail 'Build both CLI and GUI before installation.' 'CLI und Oberfläche vor der Installation bauen.'
if [[ -d /var/lib/shadow21 ]];then
 "$root/build/release/shadow21-helper" recover
fi
for p in /run/wvpn21/active /run/wvpn21-mode.json /var/lib/wvpn21/proxy-state.json /var/lib/wvpn21/firewall.owned /var/lib/wvpn21/torrc.backup /var/lib/wvpn21/dns.changed /run/anonsurf/active /run/anonsurf/iptables.rules /var/lib/anonsurf/torrc.backup;do
 [[ ! -e $p ]] || fail 'Restore upstream AnonSurf state before migrating.' 'Vor Migration den AnonSurf-Zustand wiederherstellen.'
done
missing=()
for dependency in tor tmux;do command -v "$dependency" >/dev/null || missing+=("$dependency");done
if ((${#missing[@]}));then install_packages "${missing[@]}";fi
command -v tmux >/dev/null || fail 'tmux installation failed.' 'tmux-Installation fehlgeschlagen.'
command -v tor >/dev/null || fail 'Tor installation failed; configuration was not changed.' 'Tor-Installation fehlgeschlagen; Konfiguration unverändert.'
"$root/build/release/shadow21-helper" install-proxy-config
install -d -m 755 /usr/lib/shadow21 /usr/share/applications /usr/share/icons/hicolor/scalable/apps /lib/systemd/system/tor@default.service.d /usr/share/polkit-1/actions
install -d -m 700 /etc/shadow21 /var/lib/shadow21
install -m 755 "$root/build/release/shadow21" /usr/bin/shadow21
install -m 755 "$root/build/gui/shadow21-gui" /usr/bin/shadow21-gui
install -m 755 "$root/build/release/shadow21-helper" "$root/build/release/make-torrc" "$root/build/release/safekill" /usr/lib/shadow21/
install -m 755 "$root/build/release/shadow21-dnstool" /usr/bin/shadow21-dnstool
install -m 755 "$root/scripts/install.sh" /usr/lib/shadow21/shadow21-install
for cfg in shadow21.cfg bridges.txt onion.pac;do
 [[ -e /etc/shadow21/$cfg ]] || install -m 600 "$root/configs/$cfg" "/etc/shadow21/$cfg"
done
install -m 644 "$root/sys-units/shadow21d.service" "$root/sys-units/shadow21-recovery.service" "$root/sys-units/shadow21-identity.service" /lib/systemd/system/
install -m 644 "$root/sys-units/tor@default.service.d/shadow21-recovery.conf" /lib/systemd/system/tor@default.service.d/
install -m 644 "$root/gui/org.shadow21.Desktop.desktop" "$root/gui/org.shadow21.Cli.desktop" /usr/share/applications/
install -m 644 "$root/icons/shadow21.svg" /usr/share/icons/hicolor/scalable/apps/
install -m 644 "$root/configs/org.shadow21.helper.policy" /usr/share/polkit-1/actions/
install -d -m 755 /usr/share/doc/shadow21
install -m 644 "$root/NOTICE" "$root/LICENSE" "$root/LICENSE-GPL-3" /usr/share/doc/shadow21/
systemctl daemon-reload
systemctl enable shadow21-recovery.service
say 'Installed. Local SOCKS configuration saved; no Tor reload was requested. Launch shadow21-gui as a regular user.' 'Installiert. Lokale SOCKS-Konfiguration gespeichert; kein Tor-Neuladen angefordert. shadow21-gui als normalen Benutzer öffnen.'
