# src-front - Vue Frontend

This module contains the Vue.js frontend application for Swiflow, providing the user interface for interacting with the AI assistant.

## Overview

The `src-front` module is built with Vue 3, TypeScript, and Vite. It provides a modern, responsive user interface for the Swiflow desktop application.

## Key Components

### Directory Structure

- **`config/`** - Application configuration
- **`hooks/`** - Vue composition API hooks
- **`layouts/`** - Page layout components
- **`locales/`** - Internationalization files
- **`logics/`** - Business logic and state management
- **`modals/`** - Modal dialog components
- **`stores/`** - Pinia state stores
- **`styles/`** - CSS styles and themes
- **`support/`** - Utility functions
- **`types/`** - TypeScript type definitions
- **`views/`** - Main view components
- **`widgets/`** - Reusable UI components

### Main Views

- **`browser/`** - File browser interface
- **`chatbox/`** - Main chat interface
- **`widgets/`** - Various UI widgets

## Features

### UI Framework

- **Vue 3** - Progressive JavaScript framework
- **Pinia** - State management
- **Vue I18n** - Internationalization
- **Vue Final Modal** - Modal dialogs
- **Vue Tippy** - Tooltips
- **Vite** - Fast build tool and development server
- **TypeScript** - Type safety and better development experience

### UI Components

- **CodeMirror** - Code editor component
- **Markdown** - Markdown rendering
- **Mermaid** - Diagram rendering
- **Emoji Picker** - Emoji selection
- **PDF Viewer** - PDF document viewing
- **Data Tables** - Tabular data display

### Styling

- **CSS Variables** - Theming support
- **Normalize.css** - CSS reset
- **Responsive Design** - Mobile-friendly layout

## State Management

State is managed using Pinia stores:

- **App Store** - Application state and settings
- **Chat Store** - Conversation history and messages
- **File Store** - File system operations
- **Agent Store** - AI agent management
- **Tool Store** - Available tools and capabilities

## Internationalization

The application supports multiple languages through Vue I18n. Translation files are located in `locales/` directory.

## Integration

The frontend communicates with the backend through:

- **REST API** - HTTP requests to `src-core` server
- **WebSocket** - Real-time communication
- **Tauri API** - Desktop application integration