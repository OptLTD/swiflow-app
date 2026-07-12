.PHONY: build run migrate test web-install web-dev web-build tidy

web-install:
	cd webui && pnpm install

web-dev:
	cd webui && pnpm dev

web-build:
	cd webui && pnpm install && pnpm build

build: web-build
	go build -o mira ./cmd/mira

run: build
	./mira serve --migrate -v

migrate:
	go run ./cmd/mira migrate

test: web-build
	go vet ./...
	go test ./...
	go build ./...

tidy:
	go mod tidy
