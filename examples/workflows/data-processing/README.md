# Data Processing Workflow Example

An intermediate Arcaflow workflow demonstrating data transformation and metrics extraction.

[← Back to Examples](../../README.md)

## Purpose

This workflow demonstrates:
- Complex input structures with nested objects
- Data transformation and processing
- Metrics extraction from outputs
- Result analysis fundamentals

## Workflow Description

The data-processing workflow:
1. Accepts input data (list of numbers)
2. Performs statistical calculations (sum, mean, median)
3. Returns metrics and transformed data
4. Suitable for result analysis demonstrations

**Complexity:** Intermediate  
**Estimated Time:** 15-20 minutes

## Files

- `workflow.yaml` - Workflow definition
- `inputs/small-dataset.yaml` - Small data example
- `inputs/large-dataset.yaml` - Larger data example
- `outputs/small-dataset-output.yaml` - Expected output for small dataset
- `outputs/large-dataset-output.yaml` - Expected output for large dataset

## Prerequisites

- Arcaflow MCP server installed
- MCP client or curl
- Arcaflow Engine 0.20.0+ (for execution)
- Understanding of hello-world example recommended

## Usage Example

### Input Structure

```yaml
dataset:
  name: "My Dataset"
  values:
    - 10
    - 20
    - 30
    - 40
    - 50
  threshold: 25
```

### Expected Output Structure

```yaml
output:
  statistics:
    count: 5
    sum: 150
    mean: 30.0
    median: 30.0
    min: 10
    max: 50
  analysis:
    above_threshold: 3
    below_threshold: 2
  dataset_name: "My Dataset"
```

## Learning Objectives

1. **Complex Input Construction:**
   - Build nested objects conversationally
   - Handle lists and arrays
   - Work with multiple field types

2. **Result Analysis:**
   - Extract metrics from outputs
   - Compare multiple runs with different inputs
   - Understand optimization opportunities

3. **Schema Validation:**
   - Validate complex structures
   - Handle required nested fields
   - Troubleshoot validation errors

## Next Steps

- Try [Performance Testing Example](../perf-test/) for multi-run comparison
- See [Result Analysis Tutorial](../../../docs/arcaflow-mcp/examples/iterative-optimization.md)

---

[← Back to Examples](../../README.md)
