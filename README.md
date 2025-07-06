# Go Task Queue

A minimal in-memory task queue written in Go. Tasks are submitted via HTTP, queued in memory, and processed asynchronously by a background worker.

## Features

- RESTful API using Go 1.22+ native `http.ServeMux` with path parameters
- In-memory task queue using a buffered Go channel
- Background worker that processes tasks concurrently
- In-memory task store with status/result tracking

## Requirements

- Go 1.22 or newer (uses `r.PathValue` and ServeMux path patterns)

## Project Structure

```

go-taskqueue/
├── api/         # HTTP server and handler functions
├── model/       # Core domain models (e.g., Task)
├── queue/       # In-memory buffered task queue
├── worker/      # Background task processor
├── main.go      # Application entry point
├── go.mod       # Go module definition

```

## API Endpoints

### `POST /tasks`

Submit a new task. The payload must include `status` and `result` fields.

**Request:**
```json
{
  "status": "new",
  "result": 21
}
```

**Response:**

```json
{
  "id": 1,
  "status": "queued",
  "result": 21
}
```

### `GET /tasks/{id}`

Retrieve the status and result of a task.

**Response:**

```json
{
  "id": 1,
  "status": "completed",
  "result": 42
}
```

## How It Works

* `POST /tasks` stores the task in memory and enqueues it in a buffered channel.
* A background worker dequeues tasks and performs a dummy computation (`result *= 2`).
* Task status is updated to `"completed"` after processing.
* `GET /tasks/{id}` returns the current state of the task.

## Running the Server

```bash
go run main.go
```

## Testing

Run unit tests and check code coverage:

```bash
go test -v -cover ./...
```

To generate an HTML report:

```bash
go test -coverprofile=cover.out ./...
go tool cover -html=cover.out
```

## Example Usage

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"status":"new","result":21}'

curl http://localhost:8080/tasks/1
```

## Roadmap

* [x] In-memory queue and single worker
* [x] Task result/status tracking
* [x] Unit tests for handlers and worker
* [ ] Persistent task store (e.g., SQLite/PostgreSQL)
* [ ] Redis-based queue with multiple workers
* [ ] Retry logic for failed tasks
* [ ] API filtering and task cancellation
* [ ] Metrics (Prometheus) and structured logging
* [ ] Deployment (Docker + CI/CD)

