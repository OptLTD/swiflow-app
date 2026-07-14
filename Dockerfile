FROM node:22-bookworm AS web
WORKDIR /app/webui
COPY webui/package.json webui/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY webui/ ./
RUN pnpm build

FROM golang:1.22-bookworm AS gobuild
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/embed/frontend ./embed/frontend
RUN go build -o /swiflow ./cmd/swiflow

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=gobuild /swiflow /app/swiflow
EXPOSE 8000
VOLUME ["/app/data"]
CMD ["/app/swiflow", "serve", "-c", "/app/config.json"]
