# Data Flow Architecture

This document describes the end-to-end data flows for the two primary use cases: **Input Construction** and **Result Analysis**.

## Table of Contents

1. [Input Construction Flow](#input-construction-flow)
2. [Result Analysis Flow](#result-analysis-flow)
3. [Combined Optimization Loop](#combined-optimization-loop)
4. [Error Handling Flows](#error-handling-flows)

---

## Input Construction Flow

### Overview

Input construction enables conversational, AI-driven building of workflow inputs through iterative dialog with the MCP client.

### Complete Flow Diagram

```mermaid
sequenceDiagram
    participant Client as AI Client<br/>(Claude/Cursor)
    participant Transport as Transport<br/>(stdio or HTTP/SSE)
    participant Protocol as MCP Protocol<br/>Server
    participant Tools as Workflow Tools
    participant Loader as Workflow<br/>Loader
    participant Cache as Workflow<br/>Cache
    participant Parser as Workflow<br/>Parser
    participant Validator as Input<br/>Validator
    participant State as State<br/>Manager
    
    Note over Client,State: 1. Session Initialization
    Client->>+Transport: initialize request
    Transport->>+Protocol: HandleRequest(initialize)
    Protocol->>Protocol: Create session
    Protocol-->>-Transport: Capabilities (tools, resources)
    Transport-->>-Client: Server info
    Client->>Transport: initialized notification
    Transport->>Protocol: HandleNotification(initialized)
    Protocol->>Protocol: Mark session ready
    
    Note over Client,State: 2. Workflow Discovery
    Client->>+Transport: tools/call(workflow_list)
    Transport->>+Protocol: HandleToolCall(workflow_list)
    Protocol->>+Tools: workflow_list(source)
    Tools->>+Loader: ListWorkflows(source)
    Loader->>Loader: Scan source (filesystem/git/URL)
    Loader-->>-Tools: Available workflows
    Tools-->>-Protocol: Tool result
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Workflow list
    
    Note over Client,State: 3. Schema Loading
    Client->>+Transport: tools/call(workflow_schema_get)
    Transport->>+Protocol: HandleToolCall(workflow_schema_get)
    Protocol->>+Tools: workflow_schema_get(source, selector)
    Tools->>+Loader: LoadWorkflow(source, selector)
    Loader->>Cache: Get(source, selector)
    alt Cache hit
        Cache-->>Loader: Cached workflow
    else Cache miss
        Loader->>Loader: Fetch from source
        Loader->>+Parser: Parse(YAML content)
        Parser->>Parser: Extract input schema
        Parser->>Parser: Extract output schema
        Parser-->>-Loader: Parsed workflow
        Loader->>Cache: Put(workflow)
    end
    Loader-->>-Tools: Workflow with schemas
    Tools-->>-Protocol: Input/output schemas
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Schema definitions
    
    Note over Client,State: 4. Conversational Input Building (Iterative)
    Client->>+Transport: tools/call(workflow_input_build)
    Transport->>+Protocol: HandleToolCall(workflow_input_build)
    Protocol->>+Tools: workflow_input_build(source, selector, input, merge, validate)
    Tools->>Loader: LoadWorkflow(source, selector)
    Loader-->>Tools: Workflow
    Tools->>State: GetDraft(session_id)
    State-->>Tools: Existing draft (or empty)
    Tools->>Tools: Merge or replace input
    
    opt Validation enabled
        Tools->>+Validator: ValidateInput(workflow, draft)
        Validator->>Validator: Unserialize against schema
        Validator->>Validator: Check required fields
        alt Validation success
            Validator-->>-Tools: Valid
        else Validation failure
            Validator-->>Tools: Validation errors
            Tools-->>Protocol: Error response
            Protocol-->>Transport: Error
            Transport-->>Client: Validation errors
        end
    end
    
    Tools->>State: SaveDraft(session_id, draft)
    State-->>Tools: Saved
    Tools-->>-Protocol: Draft status, missing fields
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Build result
    
    Note over Client,State: 5. Input Validation (Final Check)
    Client->>+Transport: tools/call(workflow_input_validate)
    Transport->>+Protocol: HandleToolCall(workflow_input_validate)
    Protocol->>+Tools: workflow_input_validate(source, selector, input)
    Tools->>Loader: LoadWorkflow(source, selector)
    Loader-->>Tools: Workflow
    Tools->>+Validator: ValidateInput(workflow, input)
    Validator->>Validator: Full schema validation
    alt Valid
        Validator-->>-Tools: Valid
        Tools-->>Protocol: Validation success
    else Invalid
        Validator-->>Tools: Errors
        Tools-->>Protocol: Validation errors
    end
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Validation result
    
    Note over Client,State: 6. Input Export
    Client->>+Transport: tools/call(workflow_input_export)
    Transport->>+Protocol: HandleToolCall(workflow_input_export)
    Protocol->>+Tools: workflow_input_export(source, selector, input, format)
    Tools->>+Validator: ValidateInput(workflow, input)
    Validator-->>-Tools: Valid
    Tools->>Tools: Serialize to format (JSON/YAML)
    Tools-->>-Protocol: Serialized input
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Exported input (ready for execution)
```

### Flow Steps Explained

#### 1. Session Initialization

- Client sends `initialize` request with protocol version
- Server validates protocol version (`2025-11-25`)
- Server returns capabilities (tools, resources, prompts)
- Client confirms with `initialized` notification
- Session marked ready for tool calls

#### 2. Workflow Discovery

- Client requests available workflows from a source (filesystem, git, URL)
- Server scans source and returns workflow list
- Client displays workflows to user for selection

#### 3. Schema Loading

- Client requests workflow input/output schemas
- Server loads workflow (from cache if available)
- Server parses workflow and extracts schemas
- Client receives schema definitions for validation

#### 4. Conversational Input Building

**Iterative Process:**

1. AI prompts user for input values conversationally
2. Client sends partial input to `workflow_input_build`
3. Server merges new input with existing draft
4. Optionally validates against schema (partial validation allowed)
5. Server returns updated draft and missing fields
6. Repeat until input complete

**Features:**

- **Merge Mode**: Add/update fields without replacing entire input
- **Replace Mode**: Replace entire input
- **Validation Optional**: Can build incrementally without validation
- **Session Persistence**: Draft stored in session state

#### 5. Input Validation

- Final validation with all required fields present
- Server performs complete schema validation
- Returns success or detailed error messages
- Ensures input ready for workflow execution

#### 6. Input Export

- Client requests final input in desired format (JSON/YAML)
- Server validates and serializes input
- Returns formatted input ready for Arcaflow Engine execution

---

## Result Analysis Flow

### Overview

Result analysis processes workflow execution results, extracts metrics, performs statistical analysis, and generates AI-driven optimization suggestions.

### Complete Flow Diagram

```mermaid
sequenceDiagram
    participant Client as AI Client<br/>(Claude/Cursor)
    participant Transport as Transport<br/>(stdio or HTTP/SSE)
    participant Protocol as MCP Protocol<br/>Server
    participant Tools as Workflow Tools
    participant AnalysisClient as Analysis<br/>HTTP Client
    participant AnalysisAPI as Python<br/>HTTP API
    participant Loader as Result<br/>Loader
    participant Parser as Result<br/>Parser
    participant Analyzer as Result<br/>Analyzer
    participant Suggester as Suggestion<br/>Engine
    participant History as History<br/>Repository
    
    Note over Client,History: 1. Result Loading
    Client->>+Transport: tools/call(workflow_results_load)
    Transport->>+Protocol: HandleToolCall(workflow_results_load)
    Protocol->>+Tools: workflow_results_load(source)
    Tools->>Tools: Load from file or URL
    Tools-->>-Protocol: Loaded result
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Result data
    
    Note over Client,History: 2. Result Analysis (Single Result)
    Client->>+Transport: tools/call(workflow_results_analyze)
    Transport->>+Protocol: HandleToolCall(workflow_results_analyze)
    Protocol->>+Tools: workflow_results_analyze(results, metric_directions)
    Tools->>+AnalysisClient: POST /analyze
    AnalysisClient->>+AnalysisAPI: HTTP Request
    
    AnalysisAPI->>+Loader: load_result(source)
    Loader->>Loader: Read file or URL
    Loader-->>-AnalysisAPI: Raw data
    
    AnalysisAPI->>+Parser: parse_result(data, format)
    Parser->>Parser: Detect format (JSON/YAML/text)
    Parser->>Parser: Extract metrics recursively
    Parser->>Parser: Handle nested structures
    Parser-->>-AnalysisAPI: ParsedResult
    
    AnalysisAPI->>+Analyzer: analyze_result(parsed)
    Analyzer->>Analyzer: Calculate statistics<br/>(min, max, mean, p95, p99, std_dev, CV)
    Analyzer->>Analyzer: Detect trends
    Analyzer->>Analyzer: Identify anomalies
    Analyzer-->>-AnalysisAPI: AnalysisResult
    
    AnalysisAPI->>+Suggester: generate_suggestions(analysis, metric_directions)
    Suggester->>Suggester: Apply heuristic rules
    Suggester->>Suggester: Analyze metric patterns
    Suggester->>Suggester: Generate recommendations
    Suggester->>Suggester: Estimate impact
    Suggester-->>-AnalysisAPI: List of suggestions
    
    AnalysisAPI->>+History: save_analysis(result, suggestions)
    History->>History: Store in database
    History-->>-AnalysisAPI: Saved
    
    AnalysisAPI-->>-AnalysisClient: Analysis + Suggestions
    AnalysisClient-->>-Tools: Result
    Tools-->>-Protocol: Analysis result
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Analysis + Suggestions
    
    Note over Client,History: 3. Multi-Result Comparison
    Client->>+Transport: tools/call(workflow_results_compare)
    Transport->>+Protocol: HandleToolCall(workflow_results_compare)
    Protocol->>+Tools: workflow_results_compare(results[], metric_directions)
    Tools->>+AnalysisClient: POST /compare
    AnalysisClient->>+AnalysisAPI: HTTP Request
    
    loop For each result
        AnalysisAPI->>Loader: load_result
        Loader-->>AnalysisAPI: Raw data
        AnalysisAPI->>Parser: parse_result
        Parser-->>AnalysisAPI: ParsedResult
        AnalysisAPI->>Analyzer: analyze_result
        Analyzer-->>AnalysisAPI: AnalysisResult
    end
    
    AnalysisAPI->>AnalysisAPI: Compare all results
    AnalysisAPI->>AnalysisAPI: Rank by metrics
    AnalysisAPI->>AnalysisAPI: Calculate percentage deltas
    AnalysisAPI->>AnalysisAPI: Multi-criteria scoring
    
    AnalysisAPI->>+Suggester: generate_suggestions(comparison)
    Suggester->>Suggester: Identify best configurations
    Suggester->>Suggester: Suggest improvements
    Suggester-->>-AnalysisAPI: Comparative suggestions
    
    AnalysisAPI-->>-AnalysisClient: Comparison + Rankings + Suggestions
    AnalysisClient-->>-Tools: Result
    Tools-->>-Protocol: Comparison result
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Comparison + Rankings + Suggestions
    
    Note over Client,History: 4. Historical Trend Analysis
    Client->>+Transport: tools/call(workflow_history_load)
    Transport->>+Protocol: HandleToolCall(workflow_history_load)
    Protocol->>+Tools: workflow_history_load(workflow_id, run_id)
    Tools->>+AnalysisClient: GET /history
    AnalysisClient->>+AnalysisAPI: HTTP Request
    AnalysisAPI->>+History: get_trend_data(workflow_id, metric, days)
    History->>History: Query database
    History-->>-AnalysisAPI: Historical data
    AnalysisAPI-->>-AnalysisClient: Trend data
    AnalysisClient-->>-Tools: Result
    Tools-->>-Protocol: Historical data
    Protocol-->>-Transport: JSON-RPC response
    Transport-->>-Client: Trend data
```

### Flow Steps Explained

#### 1. Result Loading

- Client loads workflow execution result from file or URL
- Server reads result data
- Returns raw result to client

#### 2. Result Analysis

**Single Result Analysis:**

1. Client sends result to `workflow_results_analyze`
2. Go server forwards to Python analysis engine via HTTP
3. Python engine loads and parses result
4. Extracts metrics (all numeric values)
5. Performs statistical analysis
6. Generates optimization suggestions based on heuristics
7. Stores analysis in history database
8. Returns analysis and suggestions to client

**Metrics Extracted:**

- All numeric values in result (nested)
- Time series data (if present)
- Resource utilization (CPU, memory, disk, network)
- Application metrics (throughput, latency, errors)

**Statistical Analysis:**

- Min, max, mean, median
- Percentiles (p95, p99)
- Standard deviation
- Coefficient of variation (variability measure)

**Suggestion Generation:**

- High variability → Stabilization recommendations
- Low utilization → Resource reduction
- High utilization → Resource increase
- Performance patterns → Configuration tuning

#### 3. Multi-Result Comparison

**Comparative Analysis:**

1. Client submits multiple results (different configurations)
2. Python engine analyzes each result independently
3. Compares all results across metrics
4. Ranks results by optimization direction (higher/lower)
5. Calculates percentage improvements/regressions
6. Performs multi-criteria scoring for overall ranking
7. Generates comparative suggestions

**Use Cases:**

- A/B testing different configurations
- Performance regression detection
- Configuration optimization experiments
- Trend analysis over time

#### 4. Historical Trend Analysis

- Query historical execution data from database
- Visualize metric trends over time
- Detect performance regressions
- Track improvement progress

---

## Combined Optimization Loop

### Overview

The complete optimization workflow combines input construction and result analysis in an iterative loop.

### Optimization Loop Diagram

```mermaid
graph TB
    Start([User Goal:<br/>Optimize Performance])
    
    Discover[1. Discover Workflow<br/>workflow_list]
    LoadSchema[2. Load Schema<br/>workflow_schema_get]
    BuildInput[3. Build Input<br/>workflow_input_build<br/><i>Conversational</i>]
    Validate[4. Validate Input<br/>workflow_input_validate]
    Export[5. Export Input<br/>workflow_input_export]
    
    Execute[6. Execute Workflow<br/><i>External: Arcaflow Engine</i>]
    
    LoadResult[7. Load Result<br/>workflow_results_load]
    Analyze[8. Analyze Result<br/>workflow_results_analyze]
    
    Satisfied{Performance<br/>Acceptable?}
    
    Suggest[9. Get Suggestions<br/>workflow_results_analyze<br/><i>AI suggestions</i>]
    UpdateInput[10. Update Input<br/>workflow_input_build<br/><i>Apply suggestions</i>]
    
    Compare[11. Compare Runs<br/>workflow_results_compare]
    History[12. Store History<br/>workflow_history_load]
    
    End([Optimized<br/>Configuration])
    
    Start --> Discover
    Discover --> LoadSchema
    LoadSchema --> BuildInput
    BuildInput --> Validate
    Validate --> Export
    Export --> Execute
    Execute --> LoadResult
    LoadResult --> Analyze
    Analyze --> Satisfied
    
    Satisfied -->|No| Suggest
    Suggest --> UpdateInput
    UpdateInput --> Validate
    
    Satisfied -->|Yes| Compare
    Compare --> History
    History --> End
    
    style Start fill:#e1f5e1
    style End fill:#e1f5e1
    style Execute fill:#fff3cd
    style Suggest fill:#cce5ff
    style Compare fill:#cce5ff
```

### Optimization Loop Steps

1. **Discovery**: Find workflow to optimize
2. **Schema Loading**: Understand input requirements
3. **Input Building**: Create baseline configuration (conversationally)
4. **Validation**: Ensure input is valid
5. **Export**: Generate execution-ready input
6. **Execution**: Run workflow with Arcaflow Engine (external)
7. **Load Result**: Import execution result
8. **Analysis**: Extract metrics, generate suggestions
9. **Satisfaction Check**: Are performance goals met?
   - **Yes**: Proceed to comparison and archival
   - **No**: Apply suggestions, update input, repeat
10. **Suggestion Application**: Update input based on AI recommendations
11. **Comparison**: Compare all iterations
12. **History**: Store for future reference

---

## Error Handling Flows

### Input Validation Errors

```mermaid
sequenceDiagram
    Client->>+Server: workflow_input_validate(invalid_input)
    Server->>+Validator: ValidateInput
    Validator->>Validator: Unserialize fails
    Validator-->>-Server: ValidationError
    Server-->>-Client: Error response with details
    
    Note over Client,Server: Error response includes:<br/>- Missing required fields<br/>- Type mismatches<br/>- Constraint violations<br/>- Suggested fixes
    
    Client->>Client: Display errors to user
    Client->>Server: workflow_input_build (corrections)
    Server-->>Client: Updated draft
```

### Analysis Errors

```mermaid
sequenceDiagram
    Client->>+Server: workflow_results_analyze(malformed_result)
    Server->>+AnalysisEngine: POST /analyze
    AnalysisEngine->>+Parser: parse_result
    Parser->>Parser: Parse fails
    Parser-->>-AnalysisEngine: ParseError
    AnalysisEngine-->>-Server: Error response
    Server-->>-Client: Error with details
    
    Note over Client,Server: Error response includes:<br/>- Parse error location<br/>- Expected format<br/>- Partial results (if any)<br/>- Recovery suggestions
    
    Client->>Client: Display error to user
    Client->>Server: workflow_results_load (corrected source)
    Server-->>Client: Loaded result
```

### Workflow Loading Errors

```mermaid
sequenceDiagram
    Client->>+Server: workflow_load(invalid_source)
    Server->>+Loader: LoadWorkflow
    Loader->>Loader: Source not found
    Loader-->>-Server: LoadError
    Server-->>-Client: Error response
    
    Note over Client,Server: Error types:<br/>- File not found<br/>- Git clone failed<br/>- URL fetch failed<br/>- Parse error<br/>- Schema invalid
    
    Client->>Client: Display error to user
    Client->>Server: workflow_list (discover sources)
    Server-->>Client: Available workflows
```

---

## Performance Characteristics

### Input Construction

- **Latency**: Low (< 100ms for cached workflows)
- **Throughput**: High (stateless, concurrent)
- **Bottlenecks**: Git clones (mitigated by caching)

### Result Analysis

- **Latency**: Medium (100-500ms depending on result size)
- **Throughput**: Medium (limited by Python HTTP server)
- **Bottlenecks**: Large result parsing, database writes

### Optimization Loop

- **Total Time**: Dominated by workflow execution (external)
- **Iterations**: Typically 2-5 iterations to optimize
- **Efficiency**: Each iteration informed by previous results

---

## Related Documentation

- [Architecture Overview](overview.md) - High-level system architecture
- [Go Server Architecture](go-server.md) - Go component details
- [Python Analysis Engine](python-engine.md) - Python component details
- [Inter-Service Communication](inter-service.md) - Go ↔ Python protocol details

---

*For usage examples, see the [User Documentation](../arcaflow-mcp/examples/).*
