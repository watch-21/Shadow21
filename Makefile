GO ?= go
CMAKE ?= cmake
VERSION ?= v1
BUILD = build/release
GUI_BUILD = build/gui
GOENV = GO111MODULE=on GOPROXY=off GOTOOLCHAIN=local
GOFLAGS = -trimpath -mod=vendor -buildvcs=false
.PHONY: all build cli gui gui-tests check clean install uninstall vendor
all: build
cli:
	mkdir -p $(BUILD)
	cd src && $(GOENV) $(GO) build $(GOFLAGS) -ldflags='-s -w -X main.version=$(VERSION)' -o ../$(BUILD)/shadow21 ./cmd/anonsurf-cli
	cd src && $(GOENV) $(GO) build $(GOFLAGS) -o ../$(BUILD)/shadow21-helper ./cmd/anondaemon
	cd src && $(GOENV) $(GO) build $(GOFLAGS) -o ../$(BUILD)/shadow21-dnstool ./cmd/dnstool
	cd src && $(GOENV) $(GO) build $(GOFLAGS) -o ../$(BUILD)/make-torrc ./cmd/make-torrc
	cd src && $(GOENV) $(GO) build $(GOFLAGS) -o ../$(BUILD)/safekill ./cmd/safekill
gui:
	$(CMAKE) -S gui -B $(GUI_BUILD) -DCMAKE_BUILD_TYPE=Release -DANONSURF_VERSION=$(VERSION)
	$(CMAKE) --build $(GUI_BUILD) --parallel 4
gui-tests:
	$(CMAKE) --build $(GUI_BUILD) --target shadow21-externalterminal-test shadow21-workspace-test shadow21-telemetry-test shadow21-mode-test shadow21-clipboard-test --parallel 4
	$(GUI_BUILD)/shadow21-clipboard-test
	$(GUI_BUILD)/shadow21-externalterminal-test
	python3 tests/ip_workspace.py
	python3 tests/telemetry_workspace.py
	python3 tests/mode_workspace.py
	python3 tests/relocated_ui.py
	python3 tests/ui_active.py
build: cli gui
check:
	cd src && $(GOENV) $(GO) test -mod=vendor ./...
	cd src && $(GOENV) $(GO) vet -mod=vendor ./...
	python3 tests/check_localization.py
install:
	./scripts/install.sh install
uninstall:
	/usr/lib/shadow21/shadow21-install uninstall
vendor:
	cd src && $(GO) mod vendor && $(GO) mod verify
clean:
	rm -rf build
