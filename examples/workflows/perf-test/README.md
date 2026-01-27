# Performance Testing Workflow Example

An advanced Arcaflow workflow for performance testing and multi-run comparison.

[← Back to Examples](../../README.md)

## Purpose

This workflow demonstrates:
- Performance testing scenarios
- Multi-run comparison and analysis
- Optimization based on results
- Advanced result analysis features

## Workflow Description

The perf-test workflow:
1. Accepts performance test configuration
2. Simulates load testing with configurable parameters
3. Returns detailed performance metrics
4. Enables multi-run comparison for optimization

**Complexity:** Advanced  
**Estimated Time:** 25-30 minutes

## Files

- `workflow.yaml` - Workflow definition
- `inputs/low-load.yaml` - Low load configuration
- `inputs/medium-load.yaml` - Medium load configuration
- `inputs/high-load.yaml` - High load configuration
- `outputs/low-load-output.yaml` - Expected low load results
- `outputs/medium-load-output.yaml` - Expected medium load results
- `outputs/high-load-output.yaml` - Expected high load results

## Prerequisites

- Arcaflow MCP server installed
- MCP client or curl
- Arcaflow Engine 0.20.0+ (for execution)
- Completion of hello-world and data-processing examples recommended

## Use Cases

### 1. Single Run Analysis

Test with specific parameters and analyze results.

### 2. Multi-Run Comparison

Run with different configurations (low, medium, high load) and compare:
- Which configuration meets performance goals?
- What's the optimal parameter combination?
- How do metrics change with load?

### 3. Iterative Optimization

1. Run initial test
2. Analyze results against goals
3. Get AI optimization suggestions
4. Adjust parameters
5. Re-run and compare
6. Repeat until goals achieved

## Input Structure

```yaml
test_config:
  name: "Load Test"
  users: 50
  duration: 60
  ramp_up: 10
  timeout: 5
  success_threshold: 0.95
```

## Expected Output Structure

```yaml
output:
  test_name: "Load Test"
  duration: 60
  total_requests: 3000
  successful_requests: 2850
  failed_requests: 150
  success_rate: 0.95
  avg_response_time: 250
  p95_response_time: 450
  p99_response_time: 650
```

## Learning Objectives

1. **Multi-Run Workflow:**
   - Execute same workflow with different inputs
   - Collect results from multiple runs
   - Load results into MCP for comparison

2. **Result Comparison:**
   - Use `workflow_results_compare` tool
   - Identify best performing configuration
   - Understand metric trends

3. **Optimization:**
   - Define performance goals
   - Get AI-driven suggestions
   - Iteratively improve results

## Multi-Run Example Workflow

### Run 1: Low Load

```bash
# Build inputs with MCP
# Export to low-load-inputs.yaml
# Execute
arcaflow -input low-load-inputs.yaml workflow.yaml
# Save output as low-load-output.yaml
```

### Run 2: Medium Load

```bash
# Build inputs with MCP
# Export to medium-load-inputs.yaml
# Execute
arcaflow -input medium-load-inputs.yaml workflow.yaml
# Save output as medium-load-output.yaml
```

### Run 3: High Load

```bash
# Build inputs with MCP
# Export to high-load-inputs.yaml
# Execute
arcaflow -input high-load-inputs.yaml workflow.yaml
# Save output as high-load-output.yaml
```

### Compare Results

Using Claude Desktop:
```
"Compare these three workflow results:
- low-load-output.yaml
- medium-load-output.yaml
- high-load-output.yaml

My goal is to achieve 95% success rate with maximum throughput."
```

MCP server will:
- Load all three result files
- Extract and compare metrics
- Identify which configuration best meets goals
- Suggest optimizations

## Next Steps

- See [Multi-Run Comparison Tutorial](../../../docs/arcaflow-mcp/examples/multi-run-comparison.md)
- Try [Iterative Optimization Tutorial](../../../docs/arcaflow-mcp/examples/iterative-optimization.md)
- Explore [Result Analysis Guide](../../../docs/arcaflow-mcp/usage/result-analysis.md)

---

[← Back to Examples](../../README.md)
