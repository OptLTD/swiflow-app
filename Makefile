.PHONY: dev dev-backend dev-frontend build image test migrate tidy desktop wails3 wails3-frontend wails3-app

# Wails CGO objects must match the linker min macOS version to avoid ld warnings.
DESKTOP_LDFLAGS := CGO_CFLAGS="-mmacosx-version-min=11.0" CGO_LDFLAGS="-mmacosx-version-min=11.0"
FRONTEND_DEVSERVER_URL ?= http://localhost:5173

# Local dev: API :8000 + Vite :5173 (proxies /api)
dev:
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	go run ./cmd/mira serve --migrate -v

dev-frontend:
	cd webui && pnpm install && pnpm dev

# Production build: frontend embedded into Go binary
build:
	cd webui && pnpm install && pnpm build
	go build -o swiflow ./cmd/mira

image:
	docker build -t mira:latest .

migrate:
	go run ./cmd/mira migrate

test:
	cd webui && pnpm install && pnpm build
	go vet ./...
	go test ./...
	go build ./...

tidy:
	go mod tidy

# Desktop app: wails3 native window with embedded backend + Vue UI
desktop:
	cd webui && pnpm install && pnpm build
	$(DESKTOP_LDFLAGS) go build -o swiflow-desktop ./cmd/desktop

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
