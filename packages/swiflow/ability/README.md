# ability - Command Execution and File System Abilities

This module provides the low-level capabilities for command execution and file system operations in Swiflow.

## Overview

The `ability` module contains concrete implementations for executing system commands and managing files, with built-in security sandboxing and process management.

## Key Components

### DevCommonAbility
Base functionality for command execution with security features:
- **Sandbox support** - macOS `sandbox-exec` and Linux `firejail` integration
- **Process management** - Timeout handling, process group killing
- **Output capture** - Combined stdout/stderr capture with logging
- **Cross-platform** - Windows and Unix support

### DevCommandAbility
Synchronous command execution:
- **`Run()`** - Execute command with timeout and return output
- **`Exec()`** - Execute shell command string
- **`Start()`** - Start command and return PID
- **`Status()`** - Check process status
- **`Stop()`** - Terminate running process

### DevAsyncCmdAbility
Asynchronous command execution:
- **`Start()`** - Start long-running command in background
- **`Query()`** - Query status and output of async command
- **`Abort()`** - Terminate async command
- **`Clear()`** - Clean up all async commands
- **Session management** - Named sessions for command tracking

### FileSystemAbility
File system operations with security restrictions:
- **Allowed extensions** - Whitelist of supported file types
- **Path validation** - Security checks for file access
- **Directory operations** - File listing and management
- **Hidden file filtering** - Exclusion of system files

## Security Features

- **Sandboxing** - Automatic sandbox execution on macOS/Linux
- **File type restrictions** - Only allowed extensions can be accessed
- **Path confinement** - Commands restricted to designated directories
- **Process isolation** - Proper cleanup of child processes

## Integration

Used by the `action` module for:
- Command execution actions
- File system operations
- Process management
- Async task handling