# src-core - Go Backend Core

This module contains the core Go backend functionality for Swiflow, providing the main server, API endpoints, and business logic.

## Overview

The `src-core` module is written in Go and serves as the backend server for the Swiflow desktop application. It handles:

- HTTP API endpoints
- WebSocket communication
- Database operations
- File system operations
- External tool integration
- Scheduled tasks
- AI agent coordination

## Key Components

### Main Packages

- **`ability/`** - Core capabilities and functionalities
- **`action/`** - Action handling and execution
- **`agent/`** - AI agent management
- **`config/`** - Configuration management
- **`entity/`** - Data models and entities
- **`entry/`**  - Business logic entry
- **`errors/`** - Custom error types
- **`httpd/`** - HTTP server implementation
- **`model/`** - Data models and database operations
- **`storage/`** - Data storage abstractions
- **`support/`** - Utility functions and helpers
- **`initial/`** - Initialization and setup

## Development

### Prerequisites

- Go 1.24.0 or later
- SQLite database
- Required Go dependencies

### Running the Server

```bash
cd src-core

# Development mode
go run . -m serve

# Chat mode (for testing)
go run . -m chat

# Debug mode
go run . -m debug

# Schedule mode
go run . -m schedule
```

### Building

```bash
# Build for current platform
go build -o swiflow-core .

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o swiflow-core-linux .
GOOS=windows GOARCH=amd64 go build -o swiflow-core-windows.exe .
GOOS=darwin GOARCH=arm64 go build -o swiflow-core-macos .
```

## Configuration

Environment variables are loaded from `.env` file:

```bash
# Database configuration
DB_PATH=./data/swiflow.db

# Server configuration
SERVER_PORT=11235
SERVER_HOST=127.0.0.1

# AI API configuration
OPENAI_API_KEY=your_api_key_here
```

## API Endpoints

The server provides RESTful API endpoints for:

- Agent management
- File operations
- Tool execution
- Memory management
- Schedule management
- System status

WebSocket connections are used for real-time communication with the frontend.

## Dependencies

Key Go dependencies include:

- `gorm.io/gorm` - ORM for database operations
- `github.com/robfig/cron/v3` - Scheduled tasks
- `github.com/gorilla/websocket` - WebSocket support
- `github.com/sashabaranov/go-openai` - OpenAI API client
- `github.com/modelcontextprotocol/go-sdk` - MCP protocol support

## Testing

Run tests with:

```bash
go test ./...
```

Test files are located in the `.test/` directory.