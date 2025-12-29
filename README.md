# Online Docs

A real-time collaborative document editing backend built with Go. Features Operational Transformation (OT) for conflict-free concurrent editing, WebSocket communication, and role-based access control.

## Architecture

```
internal/
├── acl/        # Access control (Owner, Editor, Viewer roles)
├── collab/     # Session management and operation coordination
├── handler/    # HTTP handlers (REST + WebSocket)
├── ot/         # Operational Transformation engine
├── storage/    # Document persistence (in-memory)
└── ws/         # WebSocket client/hub management
```

## Quick Start

### Prerequisites

- Go 1.21+

### Run the Server

```bash
go run main.go
```

The server starts on `http://localhost:8080`.

## API Reference

All endpoints require the `X-User-Id` header for authentication.

### REST Endpoints

#### Create Document

```bash
curl -X POST http://localhost:8080/documents \
  -H "Content-Type: application/json" \
  -H "X-User-Id: alice" \
  -d '{"id": "my-doc"}'
```

Response: `201 Created`
```json
{"id": "my-doc"}
```

#### Get Document

```bash
curl http://localhost:8080/documents/my-doc \
  -H "X-User-Id: alice"
```

Response: `200 OK`
```json
{"id": "my-doc", "content": "hello", "revision": 5}
```

#### Delete Document

```bash
curl -X DELETE http://localhost:8080/documents/my-doc \
  -H "X-User-Id: alice"
```

Response: `204 No Content`

### WebSocket Endpoint

Connect to `ws://localhost:8080/ws?docId={document-id}` with the `X-User-Id` header.

#### Message Types

**Client to Server:**

| Type | Description |
|------|-------------|
| `operation` | Submit an edit operation |
| `sync` | Request current document state |

**Server to Client:**

| Type | Description |
|------|-------------|
| `ack` | Confirms operation was applied |
| `broadcast` | Pushes another user's operation |
| `state` | Full document state |
| `error` | Error message |

#### Operation Payload

```json
{
  "type": "operation",
  "payload": {
    "docId": "my-doc",
    "baseRevision": 5,
    "opType": 0,
    "position": 5,
    "char": "!"
  }
}
```

- `opType`: `0` = insert, `1` = delete
- `position`: Character index in document
- `char`: Character to insert (omit for delete)
- `baseRevision`: Client's last known revision

#### Example Session

```bash
# Install wscat if needed: npm install -g wscat
wscat -c "ws://localhost:8080/ws?docId=my-doc" -H "X-User-Id: alice"
```

On connect, you receive the current state:
```json
{"type":"state","payload":{"docId":"my-doc","content":"","revision":0}}
```

Insert character "H" at position 0:
```json
{"type":"operation","payload":{"docId":"my-doc","baseRevision":0,"opType":0,"position":0,"char":"H"}}
```

Server acknowledges:
```json
{"type":"ack","payload":{"revision":1}}
```

## Testing

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Run the coverage threshold check:
```bash
go-test-coverage --config=.testcoverage.yml
```

## Access Control

The creator of a document is automatically granted the **Owner** role. Roles and permissions:

| Role   | Read | Write | Delete |
|--------|------|-------|--------|
| Owner  | Yes  | Yes   | Yes    |
| Editor | Yes  | Yes   | No     |
| Viewer | Yes  | No    | No     |

Users without any role cannot access the document (when ACL is enabled).

## How OT Works

Operational Transformation ensures consistency when multiple users edit simultaneously:

1. Each operation is tagged with a `baseRevision` (the revision the client last saw)
2. The server transforms the operation against any concurrent operations
3. The transformed operation is applied and assigned a new revision number
4. The operation is broadcast to all other connected clients

This allows users to continue editing without waiting for server confirmation, while the server resolves conflicts automatically.

## License

MIT
