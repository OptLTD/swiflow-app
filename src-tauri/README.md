# src-tauri - Tauri Desktop Application

This module contains the Tauri desktop application framework that packages the Swiflow frontend and backend into a native desktop application.

## Overview

The `src-tauri` module uses the Tauri framework to create a cross-platform desktop application that bundles:

- Vue.js frontend from `src-front`
- Go backend binary from `src-core`
- Native system integration
- Desktop window management

## Key Components

### Directory Structure

- **`capabilities/`** - Tauri capability definitions
- **`gen/`** - Generated code and schemas
- **`icons/`** - Application icons for all platforms
- **`src/`** - Rust source code
- **`src/plugins/`** - Tauri plugin implementations

### Tauri Configuration

The application is configured through:

- **`tauri.conf.json`** - Main configuration file
- **`Cargo.toml`** - Rust dependencies and metadata
- **`src-tauri/src/main.rs`** - Application entry point

## Features

### Desktop Integration

- **System Tray** - Tray icon and menu integration
- **File Dialogs** - Native file open/save dialogs
- **Deep Linking** - Custom URL scheme support (`swiflow://`)
- **Window Management** - Custom window styling and behavior
- **Shell Integration** - Command execution and process management

### Security

- **Asset Protocol** - Secure local asset loading
- **API Permissions** - Granular permission system
- **Code Signing** - Application signing for distribution
- **Content Security Policy** - Strict CSP for web content

### Platform Support

The application supports:

- **macOS** - DMG packages with Apple notarization
- **Windows** - MSI installers
- **Linux** - AppImage and DEB packages (planned)

## Build Process

The Tauri build process:

1. Builds the Go backend (`src-core`) 
2. Builds the Vue frontend (`src-front`)
3. Bundles both with the Rust application
4. Creates platform-specific installers

## Dependencies

Key Rust dependencies include:

- **`tauri`** - Main Tauri framework
- **`tauri-plugin-*`** - Various Tauri plugins
- **`serde`** - Serialization/deserialization
- **`tokio`** - Async runtime

## Configuration

Application settings are managed through:

- **Environment variables** - Runtime configuration
- **Tauri config** - Build-time settings
- **Frontend stores** - User preferences

## Distribution

The application supports automated distribution workflows:

- **macOS notarization** - Apple Developer Program integration
- **Windows signing** - Code signing certificate support
- **Update server** - Application update mechanism