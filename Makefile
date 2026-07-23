.PHONY: dev dev-backend dev-frontend build image test migrate tidy \
	macos macos-app windows windows-exe \
	wails3 wails3-frontend wails3-app

# Wails CGO objects must match the linker min macOS version to avoid ld warnings.
DESKTOP_LDFLAGS := CGO_CFLAGS="-mmacosx-version-min=11.0" CGO_LDFLAGS="-mmacosx-version-min=11.0"
FRONTEND_DEVSERVER_URL ?= http://localhost:5173

APP_NAME := Swiflow
APP_VERSION := 0.1.0
VERSION_PKG := github.com/OptLTD/swiflow/internal/version
VERSION_LDFLAGS := -X $(VERSION_PKG).Version=$(APP_VERSION)

# Local dev: API :8000 + Vite :5173 (proxies /api)
dev:
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	go run ./cmd/swiflow serve --migrate -v

dev-frontend:
	cd webui && pnpm install && pnpm dev

# Production build: frontend embedded into Go binary
build:
	cd webui && pnpm install && pnpm build
	go build -o swiflow ./cmd/swiflow

image:
	docker build -t swiflow:latest .

migrate:
	go run ./cmd/swiflow migrate

test:
	cd webui && pnpm install && pnpm build
	CGO_ENABLED=0 go vet $$(go list ./... | grep -v '/cmd/desktop$$')
	CGO_ENABLED=0 go test -short $$(go list ./... | grep -v '/cmd/desktop$$')
	CGO_ENABLED=0 go build -o /dev/null ./cmd/swiflow

tidy:
	go mod tidy

# macOS desktop app (.app)
macos:
	cd webui && pnpm install && pnpm build
	$(DESKTOP_LDFLAGS) go build -trimpath -ldflags="$(VERSION_LDFLAGS) -s -w" \
		-o swiflow-desktop ./cmd/desktop
	@$(MAKE) macos-app

macos-app:
	mkdir -p bin/Swiflow.app/Contents/{MacOS,Resources}
	cp build/darwin/Info.plist bin/Swiflow.app/Contents/
	cp build/darwin/icons.icns bin/Swiflow.app/Contents/Resources/
	cp swiflow-desktop bin/Swiflow.app/Contents/MacOS/Swiflow
	codesign --force --deep --sign - bin/Swiflow.app

# Windows desktop: amd64 + arm64 binaries in one NSIS installer
# Requires: wails3, makensis (brew install makensis / choco install nsis)
#   go install github.com/wailsapp/wails/v3/cmd/wails3@latest
# Usage: make windows
# Output: bin/Swiflow-installer.exe
windows:
	@command -v wails3 >/dev/null 2>&1 || { echo "error: wails3 not found (go install github.com/wailsapp/wails/v3/cmd/wails3@latest)"; exit 1; }
	@command -v makensis >/dev/null 2>&1 || { echo "error: makensis not found (brew install makensis / choco install nsis)"; exit 1; }
	cd webui && pnpm install && pnpm build
	mkdir -p bin
	$(MAKE) windows-exe ARCH=amd64
	$(MAKE) windows-exe ARCH=arm64
	wails3 generate webview2bootstrapper -dir build/windows/nsis
	# NSIS File on Windows fails with D:/ abs paths and ../../../ relative paths
	# (especially under Git Bash). Stage binaries next to project.nsi.
	cp bin/$(APP_NAME)-amd64.exe build/windows/nsis/$(APP_NAME)-amd64.exe
	cp bin/$(APP_NAME)-arm64.exe build/windows/nsis/$(APP_NAME)-arm64.exe
	cd build/windows/nsis && makensis \
		-DARG_WAILS_AMD64_BINARY=$(APP_NAME)-amd64.exe \
		-DARG_WAILS_ARM64_BINARY=$(APP_NAME)-arm64.exe \
		project.nsi
	rm -f build/windows/nsis/$(APP_NAME)-amd64.exe build/windows/nsis/$(APP_NAME)-arm64.exe
	@test -f bin/$(APP_NAME)-installer.exe || { echo "error: missing bin/$(APP_NAME)-installer.exe"; exit 1; }
	@echo "Installer: bin/$(APP_NAME)-installer.exe"

# Build one Windows arch (ARCH=amd64|arm64). Used by `make windows`.
windows-exe:
	@test -n "$(ARCH)" || { echo "error: ARCH=amd64|arm64 required"; exit 1; }
	wails3 generate syso -arch $(ARCH) \
		-icon build/windows/icon.ico \
		-manifest build/windows/wails.exe.manifest \
		-info build/windows/info.json \
		-out cmd/desktop/wails_windows_$(ARCH).syso
	GOOS=windows GOARCH=$(ARCH) CGO_ENABLED=0 \
		go build -trimpath -ldflags="-H windowsgui -s -w $(VERSION_LDFLAGS)" \
		-o bin/$(APP_NAME)-$(ARCH).exe ./cmd/desktop
	rm -f cmd/desktop/wails_windows_$(ARCH).syso
	@test -f bin/$(APP_NAME)-$(ARCH).exe || { echo "error: missing bin/$(APP_NAME)-$(ARCH).exe"; exit 1; }
	@echo "Built bin/$(APP_NAME)-$(ARCH).exe"
	@ls -la bin/$(APP_NAME)-$(ARCH).exe

# Wails3 desktop development mode: Vite HMR + live desktop window
# Uses FRONTEND_DEVSERVER_URL so AssetFileServerFS proxies to Vite (non-production build).
wails3:
	@$(MAKE) -j2 wails3-frontend wails3-app

wails3-frontend:
	cd webui && pnpm install && pnpm dev -- --host localhost --port 5173 --strictPort

wails3-app:
	@echo "Waiting for Vite at $(FRONTEND_DEVSERVER_URL) ..."
	@until curl -sf "$(FRONTEND_DEVSERVER_URL)" >/dev/null 2>&1; do sleep 0.3; done
	FRONTEND_DEVSERVER_URL="$(FRONTEND_DEVSERVER_URL)" $(DESKTOP_LDFLAGS) go run ./cmd/desktop
