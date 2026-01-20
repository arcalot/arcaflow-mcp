## Server mode

Server mode exposes MCP over HTTP/SSE for multi-tenant deployments. The HTTP
POST endpoint is wired to the JSON-RPC handler; SSE streaming provides session
binding and relays responses when a session header is present.

### Endpoints (partial)

- `POST /mcp` handles JSON-RPC client-to-server messages (including `ping`)
- `POST /mcp` returns `204 No Content` for notifications (no `id`)
- `GET /mcp/events` streams server-to-client SSE events (`session`, `message`)
- `GET /healthz` returns a simple health response

### Session binding

Server mode uses an `Mcp-Session-Id` header to bind HTTP POST requests to an SSE
stream:

1. Connect to `GET /mcp/events` to open the SSE stream.
2. Read the `Mcp-Session-Id` response header (also sent as an SSE `session`
   event).
3. Include `Mcp-Session-Id` on `POST /mcp` requests.

If an SSE session exists, `POST /mcp` requests without the session header are
rejected with `400 Bad Request`.

### Example flow

Start the SSE stream (capture the `Mcp-Session-Id` header from the response):

```
curl -N http://127.0.0.1:8080/mcp/events
```

Send a JSON-RPC request with the session header (response is returned in the
HTTP body and also emitted as an SSE `message` event):

```
curl -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}' \
  http://127.0.0.1:8080/mcp
```

### Running server mode

```
./server/arcaflow-mcp --mode server --address 127.0.0.1:8080
```

### Notes

- Authentication, authorization, and multi-tenancy are planned for Phase 2.5.
- Use local stdio mode for full MCP capabilities until HTTP/SSE is complete.
