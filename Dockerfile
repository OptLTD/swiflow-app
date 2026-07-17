# --- frontend ---
FROM node:22-bookworm AS web
WORKDIR /app/webui
COPY webui/package.json webui/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY webui/ ./
RUN pnpm build

# --- go binary ---
FROM golang:1.22-bookworm AS gobuild
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/embed/frontend ./embed/frontend
RUN CGO_ENABLED=0 go build -o /swiflow ./cmd/swiflow

# --- runtime: Chromium + Python + Node for agent tools ---
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    python3 \
    python3-pip \
    python3-venv \
    nodejs \
    npm \
    chromium \
    fonts-liberation \
    fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/bin/python3 /usr/local/bin/python \
    && chromium --version && python3 --version && node --version

# go-rod / puppeteer-style discovery
ENV CHROME_PATH=/usr/bin/chromium \
    CHROMIUM_PATH=/usr/bin/chromium \
    PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium \
    # Container runs as root; Chromium requires --no-sandbox
    SWIFLOW_BROWSER_NO_SANDBOX=1

WORKDIR /app
COPY --from=gobuild /swiflow /app/swiflow

EXPOSE 8000
VOLUME ["/app/data"]

# Chromium needs shared memory; prefer: docker run --shm-size=256m …
CMD ["/app/swiflow", "serve", "-c", "/app/config.json"]
