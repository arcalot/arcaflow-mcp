# Debugging Guide

**Techniques and Tools for Debugging Arcaflow MCP**

This guide covers common debugging scenarios, tools, and techniques for both the Go MCP server and Python analysis engine.

---

## Quick Debugging Reference

### Common Issues

| Symptom | Component | Solution |
|---------|-----------|----------|
| Server won't start | Go Server | Check logs, verify dependencies |
| Tool call fails | Go Server | Enable debug logging, check tool implementation |
| Analysis returns errors | Python Engine | Check analysis service logs, verify result format |
| Permission denied | Server Mode | Check data directory permissions, verify environment variables |
| MCP protocol errors | Transport Layer | Enable protocol debugging, verify MCP handshake |

---

## Logging

### Go Server Logging

**Enable debug logging:**

```bash
# Via environment variable
export ARCAFLOW_MCP_LOG_LEVEL=debug
./arcaflow-mcp --mode server

# Via command line
./arcaflow-mcp --mode server --log-level debug

# Via configuration file
cat > config.yaml <<EOF
logging:
  level: debug
EOF
./arcaflow-mcp --config config.yaml
```

**Log levels:**
- `debug` - Verbose, includes all operations (use for development)
- `info` - Normal operations (default)
- `warn` - Warnings and errors
- `error` - Errors only

**Log format (JSON structured):**
```json
{
  "time": "2026-01-28T12:00:00Z",
  "level": "DEBUG",
  "msg": "processing tool call",
  "component": "tools",
  "tool": "workflow_load",
  "tenant_id": "admin"
}
```

**Reading logs:**
```bash
# Filter by level
./arcaflow-mcp --mode server 2>&1 | grep '"level":"ERROR"'

# Filter by component
./arcaflow-mcp --mode server 2>&1 | grep '"component":"protocol"'

# Pretty-print JSON logs
./arcaflow-mcp --mode server 2>&1 | jq -r '. | "\(.time) [\(.level)] \(.msg)"'
```

### Python Analysis Engine Logging

**Enable debug logging:**

```bash
cd analysis

# Via environment variable
export LOG_LEVEL=DEBUG
poetry run python -m arcaflow_analysis.server.app

# Check logs
tail -f logs/analysis-engine.log
```

**Python log format:**
```
2026-01-28 12:00:00,123 [DEBUG] arcaflow_analysis.parser: Parsing result file
2026-01-28 12:00:00,456 [INFO] arcaflow_analysis.analyzer: Extracted 15 metrics
```

---

## Debugging Go Server

### Using Print Debugging

**Add debug prints:**
```go
import "log"

func MyFunction() {
    log.Printf("DEBUG: entering MyFunction")
    log.Printf("DEBUG: variable value: %+v", myVar)
}
```

**Caveat:** Remove debug prints before committing (use structured logging instead).

### Using Delve Debugger

**Install delve:**
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**Debug server:**
```bash
cd server

# Start server under debugger
dlv debug ./cmd/arcaflow-mcp -- --mode local

# Set breakpoints
(dlv) break workflow.go:42
(dlv) break pkg/tools/workflowtools/load.go:LoadWorkflow

# Continue execution
(dlv) continue

# Inspect variables
(dlv) print myVar
(dlv) locals

# Step through code
(dlv) next
(dlv) step
```

**Debug tests:**
```bash
# Debug a specific test
dlv test ./pkg/protocol -- -test.run TestInitialize

# Set breakpoint
(dlv) break protocol_test.go:25
(dlv) continue
```

### Common Go Debugging Scenarios

**nil pointer dereference:**
```bash
# Error: panic: runtime error: nil pointer dereference

# Enable stack traces
export GOTRACEBACK=all
go test ./pkg/protocol -v

# Or use debugger
dlv test ./pkg/protocol -- -test.run TestThatPanics
(dlv) break protocol.go:100  # Line before panic
(dlv) continue
(dlv) print myPointer  # Check if nil
```

**Race conditions:**
```bash
# Run with race detector
go test -race ./...

# Build server with race detection
go build -race -o arcaflow-mcp-race ./cmd/arcaflow-mcp
./arcaflow-mcp-race --mode server
```

**Memory leaks:**
```bash
# Run with memory profiling
go test -memprofile mem.prof ./pkg/protocol
go tool pprof mem.prof

# Commands in pprof:
(pprof) top10        # Top memory consumers
(pprof) list MyFunc  # Show memory allocation in function
```

---

## Debugging Python Analysis Engine

### Using ipdb

**Add breakpoint:**
```python
import ipdb

def analyze_result(data):
    ipdb.set_trace()  # Execution stops here
    metrics = extract_metrics(data)
    return metrics
```

**Run tests with breakpoint:**
```bash
cd analysis
poetry run pytest tests/test_result_analyzer.py

# When breakpoint hit:
ipdb> print(data)
ipdb> next  # Step to next line
ipdb> continue  # Continue execution
```

### Using pytest debugging

**Run test with verbose output:**
```bash
poetry run pytest -vv tests/test_result_analyzer.py

# Run single test
poetry run pytest tests/test_result_analyzer.py::test_metric_extraction -vv

# Stop on first failure
poetry run pytest -x

# Drop into debugger on failure
poetry run pytest --pdb
```

### Common Python Debugging Scenarios

**Import errors:**
```bash
# Verify Poetry environment
poetry env info

# Check installed packages
poetry show

# Reinstall dependencies
poetry install
```

**Type errors:**
```bash
# Run mypy type checking
cd analysis
poetry run mypy arcaflow_analysis/
```

**Test failures:**
```bash
# Run with captured output shown
poetry run pytest -s tests/test_analyzer.py

# Run with verbose assertion details
poetry run pytest -vv --tb=long
```

---

## Debugging MCP Protocol

### Enable Protocol Debugging

**Server side:**
```bash
export ARCAFLOW_MCP_LOG_LEVEL=debug
./arcaflow-mcp --mode server 2>&1 | grep '"component":"protocol"'
```

**Example debug output:**
```json
{"level":"DEBUG","component":"protocol","msg":"received request","method":"tools/call","id":1}
{"level":"DEBUG","component":"protocol","msg":"processing initialize","version":"2025-11-25"}
{"level":"DEBUG","component":"protocol","msg":"sending response","id":1}
```

### Inspecting JSON-RPC Messages

**Capture raw protocol:**
```bash
# stdio mode (local)
./arcaflow-mcp --mode local 2>&1 | tee protocol.log

# Server mode with curl
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d @- <<'EOF' | jq .
{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
EOF
```

### Common Protocol Issues

**"server not initialized":**

**Cause:** Missing `initialized` notification after `initialize` request

**Debug:**
```bash
# Check protocol state
grep "state transition" server-debug.log

# Verify handshake sequence
grep -E "initialize|initialized" server-debug.log
```

**Solution:** Ensure both steps of MCP handshake complete (see [Troubleshooting](../../docs/arcaflow-mcp/troubleshooting.md#server-not-initialized-error))

**"unsupported protocolVersion":**

**Cause:** Client using wrong MCP protocol version

**Debug:**
```bash
# Check client protocol version in logs
grep "protocolVersion" server-debug.log
```

**Solution:** Update client to use `2025-11-25`

---

## Debugging Inter-Service Communication

### Go → Python Communication

**Enable debug logging on both sides:**

**Go side:**
```bash
export ARCAFLOW_MCP_LOG_LEVEL=debug
./arcaflow-mcp --mode server 2>&1 | grep '"component":"analysis"'
```

**Python side:**
```bash
export LOG_LEVEL=DEBUG
poetry run python -m arcaflow_analysis.server.app
```

**Common issues:**

**"Analysis service unreachable":**
```bash
# Check Python service is running
curl http://localhost:8081/health

# Check Go server configuration
grep analysis_http_url config.yaml  # Should be http://localhost:8081

# Verify network connectivity
telnet localhost 8081
```

**"Analysis timeout":**
```bash
# Check Python service logs for slow operations
tail -f analysis/logs/analysis.log | grep "duration"

# Increase timeout in Go server (for large result files)
# Edit config or set environment variable
```

---

## Debugging Tests

### Verbose Test Output

**Go:**
```bash
# Run with verbose output
go test -v ./pkg/protocol

# Show test coverage
go test -v -cover ./pkg/protocol

# Run specific test
go test -v -run TestInitialize ./pkg/protocol
```

**Python:**
```bash
# Run with verbose output
poetry run pytest -vv

# Show print statements
poetry run pytest -s

# Run specific test
poetry run pytest tests/test_analyzer.py::test_metric_extraction -vv
```

### Test Failures

**Identify flaky tests:**
```bash
# Run test 100 times
go test -count=100 ./pkg/protocol

# Or use gotestsum
gotestsum --format testname --rerun-fails=3
```

**Debug test isolation:**
```bash
# Run tests sequentially (no parallelism)
go test -p=1 ./...

# Python: tests run sequentially by default
```

---

## Performance Debugging

### Go Performance Profiling

**CPU profiling:**
```bash
# Run with CPU profile
go test -cpuprofile cpu.prof ./pkg/protocol
go tool pprof cpu.prof

# In pprof:
(pprof) top10        # Top CPU consumers
(pprof) list MyFunc  # Show CPU usage per line
(pprof) web          # Visual call graph (requires graphviz)
```

**Memory profiling:**
```bash
go test -memprofile mem.prof ./pkg/protocol
go tool pprof mem.prof
```

**Benchmark:**
```bash
# Run benchmarks
go test -bench=. ./pkg/protocol

# With memory stats
go test -bench=. -benchmem ./pkg/protocol
```

### Python Performance Profiling

**cProfile:**
```bash
cd analysis

# Profile a test
poetry run python -m cProfile -o profile.stats \
  -m pytest tests/test_analyzer.py

# Analyze profile
poetry run python -m pstats profile.stats
>>> sort cumulative
>>> stats 20
```

**Line profiler:**
```bash
# Install line_profiler
poetry add --dev line-profiler

# Add @profile decorator to function
# Run with kernprof
poetry run kernprof -l -v your_script.py
```

---

## IDE Debugging

### VS Code / Cursor

**Go debugging configuration (`.vscode/launch.json`):**
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Server (local mode)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/server/cmd/arcaflow-mcp",
      "args": ["--mode", "local"],
      "env": {
        "ARCAFLOW_MCP_LOG_LEVEL": "debug"
      }
    },
    {
      "name": "Debug Server (server mode)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/server/cmd/arcaflow-mcp",
      "args": ["--mode", "server", "--address", ":8080"],
      "env": {
        "ARCAFLOW_MCP_LOG_LEVEL": "debug",
        "ARCAFLOW_MCP_ADMIN_TOKEN": "debug-token-12345",
        "DATA_DIR": "${workspaceFolder}/.debug-data"
      }
    },
    {
      "name": "Debug Current Test",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${fileDirname}"
    }
  ]
}
```

**Python debugging configuration:**
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Python Analysis Engine",
      "type": "python",
      "request": "launch",
      "module": "arcaflow_analysis.server.app",
      "cwd": "${workspaceFolder}/analysis",
      "env": {
        "LOG_LEVEL": "DEBUG"
      }
    },
    {
      "name": "Debug Python Tests",
      "type": "python",
      "request": "launch",
      "module": "pytest",
      "args": ["-vv", "${file}"],
      "cwd": "${workspaceFolder}/analysis"
    }
  ]
}
```

---

## Remote Debugging

### Debugging Server Mode Remotely

**1. Build with debug symbols:**
```bash
cd server
go build -gcflags="all=-N -l" -o arcaflow-mcp-debug ./cmd/arcaflow-mcp
```

**2. Run under delve headless:**
```bash
dlv exec ./arcaflow-mcp-debug --headless --listen=:2345 --api-version=2 -- \
  --mode server --address :8080
```

**3. Connect from local machine:**
```bash
dlv connect remote-host:2345
(dlv) break pkg/tools/workflowtools/load.go:42
(dlv) continue
```

---

## Troubleshooting Specific Components

### Protocol Layer

**Debug MCP handshake:**
```go
// In protocol.go, add logging
import "log"

func (s *Server) handleInitialize(req *jsonrpc.Request) (*jsonrpc.Response, error) {
    log.Printf("DEBUG: handleInitialize called, state=%v", s.state)
    // ...
}
```

**Trace state transitions:**
```bash
# Run with debug logging
ARCAFLOW_MCP_LOG_LEVEL=debug ./arcaflow-mcp --mode server 2>&1 | \
  grep "state"
```

### Workflow Tools

**Debug workflow loading:**
```bash
# Enable workflow tool debugging
export ARCAFLOW_MCP_LOG_LEVEL=debug

# Run and filter
./arcaflow-mcp --mode server 2>&1 | grep '"tool":"workflow_load"'
```

**Check workflow file access:**
```bash
# Verify file permissions
ls -la /path/to/workflow.yaml

# Test file reading
cat /path/to/workflow.yaml  # Should display workflow content
```

### Transport Layer

**Debug stdio transport:**
```bash
# Log all stdio communication
./arcaflow-mcp --mode local 2>debug.log

# Inspect communication
cat debug.log | grep -E "stdin|stdout"
```

**Debug HTTP/SSE transport:**
```bash
# Enable transport debugging
export ARCAFLOW_MCP_LOG_LEVEL=debug

# Run server
./arcaflow-mcp --mode server 2>&1 | grep '"component":"http"'

# Test with curl verbose
curl -v -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"jsonrpc":"2.0","id":1,"method":"ping"}'
```

---

## Common Debugging Scenarios

### Scenario 1: Tool Call Fails

**Symptom:** Tool returns error or unexpected result

**Steps:**
1. **Enable debug logging:**
   ```bash
   export ARCAFLOW_MCP_LOG_LEVEL=debug
   ```

2. **Reproduce the issue:**
   ```bash
   # Make the failing tool call
   curl ... -d '{"method":"workflow_load",...}'
   ```

3. **Check logs for errors:**
   ```bash
   grep '"tool":"workflow_load"' server.log
   grep '"level":"ERROR"' server.log
   ```

4. **Verify tool arguments:**
   ```bash
   # Check schema validation
   grep "validation error" server.log
   ```

5. **Test tool directly:**
   ```bash
   cd server
   go test -v -run TestWorkflowLoad ./pkg/tools/workflowtools
   ```

### Scenario 2: Performance Issues

**Symptom:** Server slow or unresponsive

**Steps:**
1. **Profile CPU:**
   ```bash
   # Add pprof endpoint (in development)
   import _ "net/http/pprof"
   go run cmd/arcaflow-mcp/main.go &
   
   # Profile for 30 seconds
   go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
   ```

2. **Check goroutine count:**
   ```bash
   curl http://localhost:6060/debug/pprof/goroutine?debug=1
   ```

3. **Memory analysis:**
   ```bash
   go tool pprof http://localhost:6060/debug/pprof/heap
   (pprof) top10
   ```

### Scenario 3: Test Failures

**Symptom:** Tests fail unexpectedly

**Steps:**
1. **Run failing test in isolation:**
   ```bash
   go test -v -run TestSpecificTest ./pkg/protocol
   ```

2. **Check test dependencies:**
   ```bash
   # Ensure test fixtures exist
   ls -la testdata/
   ```

3. **Debug test with delve:**
   ```bash
   dlv test ./pkg/protocol -- -test.run TestSpecificTest
   (dlv) break protocol_test.go:50
   (dlv) continue
   ```

4. **Check for race conditions:**
   ```bash
   go test -race -run TestSpecificTest ./pkg/protocol
   ```

---

## Advanced Debugging

### Memory Analysis

**Track memory allocations:**
```bash
# Run with memory statistics
GODEBUG=gctrace=1 ./arcaflow-mcp --mode server

# Output shows GC activity:
# gc 1 @0.005s 3%: 0.015+0.38+0.003 ms clock, ...
```

**Heap dump:**
```bash
# Trigger heap dump
curl http://localhost:6060/debug/pprof/heap > heap.prof

# Analyze
go tool pprof heap.prof
(pprof) top10
(pprof) list MyFunction
```

### Deadlock Detection

**Enable deadlock detection:**
```bash
# Build with race detector
go build -race -o arcaflow-mcp-race ./cmd/arcaflow-mcp

# Run and look for deadlock warnings
./arcaflow-mcp-race --mode server
```

**Manual analysis:**
```bash
# Dump all goroutine stacks
curl http://localhost:6060/debug/pprof/goroutine?debug=2

# Look for blocking operations
grep -A5 "chan receive" goroutine-dump.txt
```

---

## Debugging Integration Tests

### Enable Test Debugging

```bash
cd server/test/integration

# Run with verbose output
go test -v .

# Debug specific integration test
dlv test . -- -test.run TestHTTPTransport
```

### Mock Server Debugging

**For tests using mock HTTP servers:**
```bash
# Add logging to mock server
func mockHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Mock server received: %s %s", r.Method, r.URL.Path)
    body, _ := io.ReadAll(r.Body)
    log.Printf("Request body: %s", string(body))
    // ... handler logic
}
```

---

## Getting Help

### When Stuck

**1. Check documentation:**
- [Troubleshooting Guide](../../docs/arcaflow-mcp/troubleshooting.md) - Common issues
- [FAQ](../../docs/arcaflow-mcp/faq.md) - Frequently asked questions
- [Architecture Docs](../architecture/) - System design

**2. Search existing issues:**
```bash
# On GitHub
gh issue list --search "your error message"
```

**3. Ask for help:**
- [GitHub Discussions](https://github.com/arcalot/arcaflow-mcp/discussions)
- [Arcalot Community](https://github.com/arcalot/arcalot-round-table)
- Create detailed issue with reproduction steps

**4. Provide debugging info:**
```bash
# Gather debug information
./arcaflow-mcp --version
go version
poetry --version
uname -a

# Include in issue report along with:
# - Exact error message
# - Steps to reproduce
# - Expected vs actual behavior
# - Relevant log excerpt
```

---

## Related Documentation

- **[Testing Guide](testing.md)** - Testing standards and procedures
- **[Development Setup](setup.md)** - Environment setup
- **[Architecture Overview](../architecture/overview.md)** - System design
- **[Troubleshooting Guide](../../docs/arcaflow-mcp/troubleshooting.md)** - User-facing troubleshooting

---

[← Back to Development Documentation](README.md)
