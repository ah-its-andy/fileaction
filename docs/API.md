# FileAction API Documentation

Base URL: `http://localhost:8080/api`

## Workflows

### List All Workflows

```
GET /workflows
```

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "workflow-name",
    "description": "Workflow description",
    "yaml_content": "...",
    "enabled": true,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z"
  }
]
```

### Create Workflow

```
POST /workflows
```

**Request Body:**
```json
{
  "name": "my-workflow",
  "description": "Optional description",
  "yaml_content": "name: my-workflow\n...",
  "enabled": true
}
```

**Response:** 201 Created
```json
{
  "id": "uuid",
  "name": "my-workflow",
  ...
}
```

### Get Workflow

```
GET /workflows/:id
```

**Response:**
```json
{
  "id": "uuid",
  "name": "workflow-name",
  ...
}
```

### Update Workflow

```
PUT /workflows/:id
```

**Request Body:**
```json
{
  "name": "updated-name",
  "description": "Updated description",
  "yaml_content": "...",
  "enabled": false
}
```

**Response:**
```json
{
  "id": "uuid",
  "name": "updated-name",
  ...
}
```

### Delete Workflow

```
DELETE /workflows/:id
```

**Response:**
```json
{
  "message": "Workflow deleted"
}
```

### Scan Workflow

```
POST /workflows/:id/scan
```

Triggers a scan of all paths defined in the workflow. Creates tasks for new or changed files.

**Response:**
```json
{
  "message": "Scan started"
}
```

## Tasks

### List Tasks

```
GET /tasks?workflow_id=:id&status=:status&limit=:limit&offset=:offset
```

**Query Parameters:**
- `workflow_id` (optional): Filter by workflow ID
- `status` (optional): Filter by status (pending, running, completed, failed, cancelled)
- `limit` (optional): Number of results (default: 50, max: 1000)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
{
  "tasks": [
    {
      "id": "uuid",
      "workflow_id": "uuid",
      "file_id": "uuid",
      "input_path": "/path/to/input.jpg",
      "output_path": "/path/to/output.png",
      "status": "pending",
      "log_text": "...",
      "error_message": null,
      "started_at": "2023-01-01T00:00:00Z",
      "completed_at": null,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

### Get Task

```
GET /tasks/:id
```

**Response:**
```json
{
  "id": "uuid",
  "workflow_id": "uuid",
  "file_id": "uuid",
  "input_path": "/path/to/input.jpg",
  "output_path": "/path/to/output.png",
  "status": "completed",
  "log_text": "Full execution log...",
  "error_message": null,
  "started_at": "2023-01-01T00:00:00Z",
  "completed_at": "2023-01-01T00:05:00Z",
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-01-01T00:05:00Z"
}
```

### Get Task Steps

```
GET /tasks/:id/steps
```

**Response:**
```json
[
  {
    "id": "uuid",
    "task_id": "uuid",
    "name": "convert-image",
    "command": "convert input.jpg output.png",
    "status": "completed",
    "exit_code": 0,
    "stdout": "...",
    "stderr": "",
    "started_at": "2023-01-01T00:00:00Z",
    "completed_at": "2023-01-01T00:05:00Z",
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:05:00Z"
  }
]
```

### Tail Task Log

```
GET /tasks/:id/log/tail?offset=:offset
```

Stream task logs, useful for real-time monitoring of running tasks.

**Query Parameters:**
- `offset` (optional): Byte offset to start reading from (default: 0)

**Response:**
```json
{
  "content": "New log content since offset...",
  "offset": 1234,
  "completed": false
}
```

### Retry Task

```
POST /tasks/:id/retry
```

Resets a failed or cancelled task to pending status and submits it for execution.

**Response:**
```json
{
  "message": "Task retry initiated"
}
```

### Cancel Task

```
POST /tasks/:id/cancel
```

Cancels a running task.

**Response:**
```json
{
  "message": "Task cancelled"
}
```

### Delete Task

```
DELETE /tasks/:id
```

**Response:**
```json
{
  "message": "Task deleted"
}
```

## Files

### List Files

```
GET /files?workflow_id=:id&limit=:limit&offset=:offset
```

**Query Parameters:**
- `workflow_id` (required): Workflow ID
- `limit` (optional): Number of results (default: 50, max: 1000)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
{
  "files": [
    {
      "id": "uuid",
      "workflow_id": "uuid",
      "file_path": "/path/to/file.jpg",
      "file_md5": "abc123...",
      "file_size": 1024000,
      "last_scanned_at": "2023-01-01T00:00:00Z",
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

## Error Responses

All endpoints return errors in the following format:

```json
{
  "error": "Error message description"
}
```

**HTTP Status Codes:**
- `400` - Bad Request (invalid input)
- `404` - Not Found
- `500` - Internal Server Error

## Authentication

Currently, the API does not require authentication. For production deployments, consider adding authentication middleware (e.g., JWT, API keys) or placing the service behind a reverse proxy with authentication.

## Rate Limiting

No rate limiting is currently implemented. For production use, consider implementing rate limiting at the application or reverse proxy level.

## CORS

CORS is enabled for all origins by default. Adjust the CORS configuration in the server code for production deployments.
