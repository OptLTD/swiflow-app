# Chat System Core Logic Documentation

## Overview

This document describes the core logic and message handling system for the chat interface in the SwiFlow application. The system is built around a reactive message store (`msg.ts`) and a main chat component (`ChatBox.vue`) that work together to handle real-time messaging with LLM agents.

## Architecture

### Message Store (`stores/msg.ts`)

The message store is the central hub for all message-related state management and processing. It uses Pinia for state management and includes:

#### State Properties
- `taskid`: Current active task/chat ID
- `errmsg`: Error message display
- `running`: Boolean indicating if a task is currently running
- `subtasks`: Array of subtask IDs for priority handling
- `nextMsg`: Current message being processed for display
- `streams`: Object storing stream data for each task/subtask

#### Key Methods

**Message Processing:**
- `processMessage(msg: SocketMsg)`: Main message router that handles different message types
- `processNextMsg()`: Throttled method that processes stream data and emits UI updates
- `handleUserInput()`, `handleControl()`, `handleRespond()`, `handleStream()`: Specific handlers for different message types

**Subtasks Management:**
- `addSubtask(subtaskId)`: Add a new subtask to the priority queue
- `removeSubtask(subtaskId)`: Remove a subtask from the queue
- `clearSubtasks()`: Clear all subtasks
- Priority logic: First subtask with stream data gets display priority

**Stream Management:**
- `updateStream(taskid, idx, str)`: Update stream data for a specific task
- `clearStream(taskid)`: Clear stream data for a task
- Stream data is stored as indexed objects for incremental updates

#### Event System

The store uses a custom event emitter (`MsgEventEmitter`) to communicate with UI components:

- `user-input`: Emitted when user sends a message
- `respond`: Emitted when LLM provides a complete response
- `next-msg`: Emitted when stream processing updates the display (replaces old `stream` event)

### Chat Component (`views/ChatBox.vue`)

The main chat interface component that handles user interactions and displays messages.

#### Key Features

**Message Display:**
- `messages`: Local array of complete messages for display
- `inputMsg`: Current user input state
- Real-time message updates through event listeners

**Event Handling:**
- Listens to `user-input`, `respond`, and `next-msg` events from the message store
- `startPlayAction()`: Processes message actions for display
- Auto-scroll functionality for new messages

**User Interactions:**
- Message sending with validation
- File upload handling
- Tool selection and configuration
- Task switching and management

## Message Flow

### 1. User Input Flow
```
User types message → handleSend() → WebSocket send → 
msg.handleUserInput() → emit 'user-input' → 
ChatBox adds to messages array → Auto-scroll
```

### 2. LLM Response Flow
```
WebSocket receives → msg.processMessage() → 
handleRespond() → emit 'respond' → 
ChatBox adds complete message → startPlayAction()
```

### 3. Stream Processing Flow
```
WebSocket stream → msg.handleStream() → 
updateStream() → processNextMsg() → 
Priority check (subtasks first) → Parse stream data → 
emit 'next-msg' → ChatBox updates display → Auto-scroll
```

## Subtasks Priority System

The system implements a priority-based display system for subtasks:

1. **Priority Order**: Subtasks are processed in the order they were added to the `subtasks` array
2. **Stream Detection**: The system checks each subtask for available stream data
3. **Display Logic**: The first subtask with stream data gets display priority
4. **Fallback**: If no subtasks have stream data, the main task stream is used

### Implementation Details

```typescript
// In processNextMsg method
let activeTaskId = ''
if (this.subtasks.length > 0) {
  // Find the first subtask that has stream data
  for (const subtaskId of this.subtasks) {
    if (this.streams[subtaskId] && Object.keys(this.streams[subtaskId]).length > 0) {
      activeTaskId = subtaskId
      break
    }
  }
}
// If no subtask has stream data, use the main taskid
if (!activeTaskId) {
  activeTaskId = this.taskid
}
```

## Stream Data Format

Stream data is received incrementally and stored as indexed objects:

```typescript
streams: {
  "task-id-1": {
    0: "msgid=======>unique-message-id",
    1: "First chunk of data",
    2: "Second chunk of data",
    // ... more chunks
  }
}
```

The `processNextMsg` method reconstructs the complete message by concatenating chunks in order.

## Error Handling

The system includes comprehensive error handling:

- **Connection Errors**: WebSocket connection issues
- **LLM Errors**: Various LLM response errors (empty response, exceeded turns, etc.)
- **Parsing Errors**: XML/message parsing failures
- **Task Errors**: Task termination and timeout handling

## Performance Optimizations

1. **Throttling**: `processNextMsg` is throttled to 180ms to prevent excessive updates
2. **Auto-scroll Throttling**: Scroll updates are throttled to 500ms
3. **Event Cleanup**: Proper event listener cleanup in component unmount
4. **Stream Cleanup**: Automatic cleanup of completed task streams

## Integration Points

### WebSocket Integration
- Receives messages in `SocketMsg` format
- Handles different message actions: `user-input`, `control`, `respond`, `stream`, `errors`, `change`

### Parser Integration
- Uses `parser.Parse()` to convert raw stream data into structured `ActionMsg` objects
- Handles XML-based message format from LLM responses

### Task Management
- Integrates with `useTaskStore()` for task lifecycle management
- Handles task switching and history management

## Best Practices

1. **State Management**: All message state should go through the store, not component state
2. **Event Handling**: Use the event emitter for loose coupling between store and components
3. **Error Handling**: Always handle errors gracefully and provide user feedback
4. **Performance**: Use throttling for high-frequency updates
5. **Cleanup**: Always clean up event listeners and streams when components unmount

## Future Considerations

- **Message Persistence**: Consider adding message persistence for offline scenarios
- **Message Search**: Implement search functionality across message history
- **Performance Monitoring**: Add metrics for message processing performance
- **Accessibility**: Ensure proper ARIA labels and keyboard navigation support