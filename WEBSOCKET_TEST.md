# WebSocket Real-time Log Streaming Test Plan

## Overview
This document describes how to test the newly implemented WebSocket service for real-time task log streaming.

## Feature Description
When viewing logs of a running task, the frontend establishes a WebSocket connection to receive real-time log updates instead of polling via HTTP.

## Architecture
- **Backend**: WebSocketHub manages client connections and subscriptions
- **Frontend**: Connects to `/api/ws/logs`, sends subscription message with task ID
- **Communication**: Unicast model - logs are pushed only to subscribed clients, not broadcast

## Test Steps

### 1. Start the Application
```bash
./fileaction
```

### 2. Create a Long-Running Task
Create a workflow that runs for at least 30 seconds to test real-time streaming:

**Example workflow** (`test-workflow.yaml`):
```yaml
name: test-realtime-log
watch_dir: ./examples
file_pattern: "*.txt"
steps:
  - name: slow-process
    command: |
      for i in {1..30}; do
        echo "Processing step $i/30"
        sleep 1
      done
      echo "Complete!"
```

### 3. Trigger Task Execution
1. Place a test file in the `./examples` directory
2. Wait for the watcher to detect and create a task
3. Click the task in the UI to view its log

### 4. Verify Real-time Streaming

**Expected Behavior:**
- ✅ Log modal opens with "Loading log..." message
- ✅ Console shows: "WebSocket connected for task: <task_id>"
- ✅ Console shows: "Subscribed to task: <task_id>"
- ✅ Log content appears line by line in real-time (no 1-second delay)
- ✅ Content auto-scrolls to bottom as new logs arrive
- ✅ Console shows: "Task completed" when task finishes
- ✅ Task list refreshes automatically after completion

**Browser Console Checks:**
```javascript
// Open browser console (F12) and verify:
// 1. WebSocket connection established
WebSocket connection to 'ws://localhost:3000/api/ws/logs' established

// 2. Messages received
WebSocket message: subscribed
WebSocket message: log
WebSocket message: complete
```

### 5. Test Multiple Clients
1. Open the application in two different browser windows
2. Start a long-running task
3. View the same task log from both windows
4. **Expected**: Both windows receive logs independently (unicast, not broadcast)

### 6. Test Connection Cleanup
1. Open a running task's log
2. Close the modal before task completes
3. **Expected**: 
   - Console shows "WebSocket closed"
   - Backend removes client from subscribers
   - No memory leaks

### 7. Test Idle Timeout
1. Open a task log
2. Leave it open for more than 5 minutes without activity
3. **Expected**: Server closes connection with idle timeout message

### 8. Test Fallback to HTTP
1. Stop the backend server
2. Try to view a task log
3. **Expected**: WebSocket connection fails, falls back to HTTP polling

## Debugging

### Backend Logs
Watch for these messages in the server output:
```
Client connected: <client_id>
Client subscribed to task <task_id>
Broadcasting log to task <task_id>
Task <task_id> completed
Client disconnected: <client_id>
```

### Frontend Console
Check browser console for:
```javascript
WebSocket connected for task: <task_id>
WebSocket message: subscribed
WebSocket message: log
Task completed
Server closing WebSocket
```

### Network Tab
Open browser DevTools → Network → WS (WebSocket) to see:
- Connection establishment
- Messages sent/received
- Connection close

## Common Issues

### Issue: Logs not appearing in real-time
**Check:**
- WebSocket connection established? (Network → WS tab)
- Subscription message sent? (Check browser console)
- Task actually running? (Check task status)

### Issue: "WebSocket connection error"
**Check:**
- Server running on correct port?
- Firewall blocking WebSocket connections?
- Using HTTPS without WSS support?

### Issue: Multiple connections opened
**Check:**
- Modal closed properly when switching between tasks?
- `state.logWebSocket` cleaned up in `closeModal()`?

## Success Criteria
- ✅ Real-time log streaming works without delays
- ✅ Multiple clients can view same task independently
- ✅ Connections cleaned up properly on modal close
- ✅ Fallback to HTTP polling works when WebSocket unavailable
- ✅ No console errors during normal operation
- ✅ Server handles connection lifecycle correctly

## Performance Notes
- WebSocket uses persistent connection (lower overhead than polling)
- Ping/pong every 20 seconds keeps connection alive
- Server cleans up idle connections after 5 minutes
- Each client maintains independent subscription
