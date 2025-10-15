# agent - Bot Management and Execution

This module manages AI bots (agents), their context, and execution workflows in Swiflow.

## Overview

The `agent` module provides:
- Bot lifecycle management and persistence
- Context management for conversations
- Action execution orchestration
- Integration with storage and external services

## Key Components

### Manager
Central bot management:
- **Bot initialization** - Load and configure bots from storage
- **Session management** - Active bot sessions and contexts
- **Storage integration** - Database persistence for bots and messages
- **Executor coordination** - Manage bot execution workflows

### Executor
Bot execution engine:
- **Context building** - Construct prompt context from history
- **Action parsing** - Parse and execute AI-generated actions
- **Tool integration** - MCP tool management and execution
- **Result processing** - Handle action results and errors

### Context
Conversation context management:
- **Message history** - Maintain conversation thread
- **Memory integration** - Incorporate bot memories
- **Tool context** - Available tools and capabilities
- **System context** - Environment and configuration state

## Core Features

### Bot Management
- **Bot persistence** - Storage in SQLite database
- **Multi-bot support** - Multiple concurrent bots
- **Bot configuration** - Settings, prompts, and capabilities
- **Bot switching** - Dynamic bot selection

### Context Building
- **Message history** - Conversation context preservation
- **Memory integration** - Relevant memory recall
- **Tool descriptions** - Available tool documentation
- **System state** - Environment and configuration context

### Action Execution
- **XML action parsing** - Parse AI-generated action plans
- **Ability integration** - Delegate to ability modules
- **Error handling** - Graceful error recovery
- **Result processing** - Action result collection

## Storage Integration

Uses entity models for data persistence:
- **`BotEntity`** - Bot configurations and settings
- **`MsgEntity`** - Message history and context
- **`TaskEntity`** - Scheduled tasks and reminders

## Integration Points

- **MCP integration** - Model Context Protocol tool management
- **Ability modules** - Command execution and file operations
- **Action system** - XML action parsing and execution
- **Storage system** - Database persistence
- **Configuration** - Environment and settings management