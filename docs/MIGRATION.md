# Migration to Shadow21 v1

Shadow21 can coexist with an inactive legacy installation only after its managed networking has been restored. It does not silently rename privileged state or delete the old installation. Recovery journals must remain with the old helper until restoration completes.

1. Stop the active mode in the old interface, then Quit through its tray. Save user preferences if desired.
2. Restore and remove the old package with its own helper:

```bash
sudo wvpn21 stop
sudo /usr/lib/wvpn21/wvpn21-helper recover
sudo apt remove wvpn21
```

For an old source installation, use its documented `sudo /usr/lib/wvpn21/wvpn21-install uninstall` instead of apt. If recovery/removal fails, keep current configuration and protected backups; resolve the reported conflict before continuing. Do not remove journals manually or flush firewall tables.

3. Install the new package from this archive:

```bash
sudo apt install ./dist/shadow21_v1_amd64.deb
shadow21-gui
```

4. Optionally preserve old user settings, without overwriting an existing new preference file:

```bash
mkdir -p ~/.config/shadow21
if [ -f ~/.config/wvpn21/preferences.json ] && [ ! -e ~/.config/shadow21/preferences.json ]; then
    cp ~/.config/wvpn21/preferences.json ~/.config/shadow21/preferences.json
    chmod 600 ~/.config/shadow21/preferences.json
fi
```

Preferences are validated on load. Selecting tmux in the GUI prepares the system Tor service; changing CLI preferences alone does not start a session. Do not copy root networking state, sockets or journals to new names. The old user preference directory is preserved.

For later Shadow21 upgrades, stop modes and Quit, then install the changed v1 package. An internal Debian revision identifies the build. Upgrade preserves configuration and the first-install Tor backup. Removal uses `sudo apt remove shadow21` or the source installer uninstall action. Recovery precedes owned-file removal. External Tor edits are preserved and may require manual reconciliation.
