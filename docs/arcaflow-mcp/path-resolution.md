# Automatic Path Resolution

Arcaflow MCP automatically resolves relative filesystem paths to absolute paths,
eliminating the need for clients to determine the current working directory before
calling tools.

## Problem Solved

**Before path resolution:**
```
User: "Analyze results at @mcp-test-out-1.yaml"
Client: workflow_results_analyze({"source": {"kind": "filesystem", "location": "mcp-test-out-1.yaml"}})
Server: Error - "stat mcp-test-out-1.yaml: no such file or directory"
Client: Run pwd to get "/home/user/project"
Client: workflow_results_analyze({"source": {"kind": "filesystem", "location": "/home/user/project/mcp-test-out-1.yaml"}})
Server: Success
```

**After path resolution:**
```
User: "Analyze results at @mcp-test-out-1.yaml"
Client: workflow_results_analyze({"source": {"kind": "filesystem", "location": "mcp-test-out-1.yaml"}})
Server: Automatically resolves to "/home/user/project/mcp-test-out-1.yaml"
Server: Success
```

## Supported Paths

All workflow and result tools accept filesystem paths in these formats:

### Relative Paths (Resolved Automatically)

- **Single file:** `workflow.yaml`, `results.json`
- **Subdirectory:** `subdir/workflow.yaml`, `results/output.yaml`
- **Current directory:** `.`, `./workflow.yaml`
- **Parent directory:** `../workflow.yaml`, `../../results.yaml`

### Absolute Paths (Used As-Is)

- **Unix/Linux:** `/home/user/workflows/workflow.yaml`
- **Windows:** `C:\Users\user\workflows\workflow.yaml`

## Implementation

Path resolution happens at the tool handler level before any file operations:

1. **Check if path is already absolute** - If yes, use as-is
2. **Get current working directory** - Via `os.Getwd()`
3. **Join relative path with CWD** - Using `filepath.Join()`
4. **Return resolved absolute path** - Guaranteed to be absolute

The resolution is **idempotent** - passing an already-resolved path returns the
same result.

## Affected Tools

Path resolution is applied to all tools that accept filesystem sources:

### Workflow Tools

- `workflow_list` - source.location
- `workflow_load` - source.location
- `workflow_input_recommend` - source.location
- `workflow_discover` - source.location (hidden)
- `workflow_describe` - source.location (hidden)
- `workflow_schema_get` - source.location (hidden)
- `workflow_input_build` - source.location (hidden)
- `workflow_input_validate` - source.location (hidden)
- `workflow_input_export` - source.location (hidden)

### Result Tools

- `workflow_results_load` - source.location
- `workflow_results_describe` - source.location
- `workflow_results_analyze` - source.location

## Benefits for AI Clients

1. **Natural `@file` references** - Clients can pass filenames directly from user input
2. **No need for `pwd`** - Eliminates extra Shell calls to determine working directory
3. **Fewer retries** - No trial-and-error with relative vs absolute paths
4. **Consistent behavior** - Works the same regardless of how the client specifies paths

## Error Handling

If the current working directory cannot be determined (rare), the path resolution
function returns an error:

```
Error: "resolve filesystem path: get current directory: <error>"
```

This is surfaced to the client as an MCP error with the underlying system error
message.

## Testing

Comprehensive tests verify path resolution:

- ✅ Absolute paths unchanged
- ✅ Relative files resolved (e.g., `workflow.yaml`)
- ✅ Relative directories resolved (e.g., `subdir/file`)
- ✅ Dot paths resolved (e.g., `.`, `./file`)
- ✅ Parent paths resolved (e.g., `../file`)
- ✅ Empty paths rejected
- ✅ Idempotent (resolving twice returns same result)

## Example: Real-World Usage

From actual client transcript showing path resolution in action:

**First attempt (failed before path resolution):**
```json
{
  "name": "workflow_input_recommend",
  "args": {
    "source": {
      "kind": "filesystem",
      "location": "workflow.yaml"
    }
  }
}
```
**Error:** "stat filesystem root: stat workflow.yaml: no such file or directory"

**Second attempt (worked after implementing path resolution):**
Same call now succeeds automatically - the server resolves `workflow.yaml` to
`/home/dblack/git/gitlab/perfscale/arcaflow-workflow-auto-perf/workflow.yaml`
based on the current working directory.

## Code Location

- **Path resolution function:** `server/pkg/arcaflow/workflow/pathutil.go`
- **Tests:** `server/pkg/arcaflow/workflow/pathutil_test.go`
- **Workflow loader integration:** `server/pkg/arcaflow/workflow/loader.go`
- **Result loader integration:** `server/pkg/tools/workflowtools/results_load.go`
