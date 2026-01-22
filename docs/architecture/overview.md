## Architecture overview

Arcaflow MCP uses a hybrid architecture:

- Go for MCP protocol, transports, and Skill 1 schema handling
- Python for Skill 2 result analysis and suggestion generation

Inter-service communication is expected to be gRPC.

See `docs/architecture/persistence.md` for the canonical inventory of
server-mode state that must survive restarts.
