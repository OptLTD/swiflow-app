.PHONY: dev dev-backend dev-frontend build image test migrate tidy

# Local dev: API :8000 + Vite :5173 (proxies /api)
dev:
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	go run ./cmd/mira serve --migrate -v

dev-frontend:
	cd webui && pnpm install && pnpm dev

# Production build: webui/dist embedded into Go binary
build:
	cd webui && pnpm install && pnpm build
	go build -o mira ./cmd/mira

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
