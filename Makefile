.PHONY: build run migrate test web-install web-dev web-build tidy

build:
	go build -o mira ./cmd/mira

run: build
	./mira serve --migrate -v

migrate:
	go run ./cmd/mira migrate

test:
	go vet ./...
	go build ./...

tidy:
	go mod tidy
