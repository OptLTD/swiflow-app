# action - XML Action Parsing and Execution

This module handles the parsing of XML-based action definitions from AI responses and their execution through appropriate abilities.

## Overview

The `action` module provides:
- XML parsing of AI-generated action plans
- Structured action interfaces with handler methods
- Integration with ability modules for execution
- Result collection and error handling

## Core Action Types

### Command Execution Actions
- **`ExecuteCommand`** - Synchronous command execution
- **`StartAsyncCmd`** - Start asynchronous command with session management
- **`QueryAsyncCmd`** - Query status of async command
- **`AbortAsyncCmd`** - Terminate async command

### Basic Interaction Actions
- **`UserInput`** - User input and file upload handling
- **`ToolResult`** - Tool execution results
- **`WaitTodo`** - Scheduled task reminders
- **`Complete`** - Task completion signaling
- **`MakeAsk`** - User question prompts
- **`Thinking`** - AI reasoning steps
- **`Memorize`** - Memory storage actions
- **`Annotate`** - Context annotation

## Action Structure

All actions implement the `IAct` interface:
```go
type IAct interface {
    Handle(super *SuperAction) any
}
```

### SuperAction Context
Contains execution context:
- **`Payload`** - UUID, home directory, path, timestamp
- **`UseTools`** - List of tools to use
- **`Thinking`** - AI reasoning content
- **`Errors`** - Collection of execution errors

## XML Parsing

Actions are defined in XML format and parsed from AI responses:
```xml
<execute-command>
    <command>ls -la</command>
</execute-command>
```

## Execution Flow

1. **XML Parsing** - Parse AI response into action objects
2. **Payload Initialization** - Set up execution environment
3. **Ability Integration** - Delegate to appropriate ability module
4. **Result Collection** - Capture outputs and errors
5. **Error Handling** - Graceful error management

## Integration

- Uses `ability` module for command execution
- Integrated with `agent` module for AI planning
- Supports file system operations through abilities
- Provides structured results back to AI system