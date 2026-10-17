# Build and verification

Requirements: Go 1.24+, CMake 3.22+, C++17, Qt 6.4+ Core/DBus/Gui/Network/Qml/Quick/QuickControls2/Widgets, Python 3 and platform networking dependencies. Go dependencies are vendored.

From the project root:

```bash
sudo ./scripts/install.sh dependencies
# Install Go 1.24+ separately if the distribution package is older.
make build
make check
make gui-tests
python3 tests/ui_render.py
python3 tests/cli_smoke.py
./scripts/package.sh
python3 tests/package_install.py
```

The package is written to `dist/shadow21_v1_amd64.deb`. Install it using:

```bash
sudo apt install ./dist/shadow21_v1_amd64.deb
shadow21-gui
```

Packaging requires a fresh staging directory. Source installation is available with `sudo ./scripts/install.sh install` after building. See [platform support](SUPPORT.md) for host requirements and binary ABI limitations.

Local checks use fixtures and offscreen UI rendering. They do not establish full desktop/network compatibility. Read [actual test results](TEST-RESULTS.md) and use [disposable desktop VMs](VM-VALIDATION.md) for live networking verification. Do not change the development machine's active networking during tests.
