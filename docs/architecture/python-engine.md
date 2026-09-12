# Python Analysis Engine Architecture

The Python analysis engine (`analysis/`) processes workflow results, extracts metrics, performs statistical analysis, and generates AI-driven optimization suggestions.

## Package Structure

```
analysis/
├── arcaflow_analysis/
│   ├── server/              # HTTP API server
│   │   ├── app.py           # Application setup
│   │   ├── http_api.py      # API endpoint handlers
│   │   └── http_server.py   # HTTP server main
│   ├── parser/              # Result parsing
│   │   ├── result_loader.py # Load from file/URL
│   │   └── result_parser.py # Parse JSON/YAML/text
│   ├── analyzer/            # Analysis logic
│   │   ├── result_analyzer.py    # Metrics extraction
│   │   └── result_comparator.py  # Multi-run comparison
│   ├── suggester/           # Optimization suggestions
│   │   └── suggestion_engine.py
│   └── db/                  # Persistence
│       ├── models.py        # SQLAlchemy models
│       ├── repository.py    # Data access layer
│       └── session.py       # DB session management
└── tests/                   # Pytest test suite
```

## Core Components

### HTTP API Server (`server/`)

**Purpose**: REST API for workflow result analysis.

**Framework**: Python 3.12, HTTP server (currently synchronous, designed for async upgrade)

**Endpoints:**

- `POST /analyze` - Analyze single result, generate suggestions
- `POST /compare` - Compare multiple results, rank by metrics
- `POST /suggest` - Generate optimization suggestions from results
- `POST /metrics` - Extract metrics only (no analysis)
- `POST /parse` - Parse result without analysis
- `GET /healthz` - Health check endpoint

**Request/Response Format:**

All endpoints accept JSON payloads and return JSON responses.

**Example Request (POST /analyze):**

```json
{
  "results": [
    {
      "payload": {...},
      "format": "json"
    }
  ],
  "metric_directions": {
    "throughput": "higher",
    "latency": "lower"
  },
  "compare": true
}
```

**Example Response:**

```json
{
  "metrics": {
    "throughput": {
      "min": 100,
      "max": 500,
      "mean": 300,
      "p95": 450,
      "p99": 480,
      "std_dev": 50,
      "cv": 0.16
    }
  },
  "suggestions": [
    {
      "metric": "throughput",
      "current": 300,
      "recommendation": "Increase worker_count to 8",
      "expected_impact": "20-30% improvement"
    }
  ],
  "comparison": {...}
}
```

**Application Lifecycle:**

```python
# app.py - Application setup
def create_app() -> Any:
    app = HTTPApp()
    
    # Register endpoints
    app.add_route("/analyze", handle_analyze, methods=["POST"])
    app.add_route("/compare", handle_compare, methods=["POST"])
    # ...
    
    # Configure database
    configure_database(app)
    
    return app
```

**Error Handling:**

```python
# Structured error responses
{
  "error": "Invalid input",
  "detail": "Missing required field: payload",
  "code": "VALIDATION_ERROR"
}
```

**HTTP Status Codes:**

- `200 OK` - Successful analysis
- `400 Bad Request` - Invalid input
- `500 Internal Server Error` - Server error

### Result Parser (`parser/`)

**Purpose**: Load and parse workflow results from various sources and formats.

#### Result Loader (`result_loader.py`)

**Purpose**: Load result files from filesystem or URL.

**Features:**

- File loading (JSON, YAML, text)
- URL loading (HTTP/HTTPS)
- Format detection
- Error handling for missing/invalid files

**API:**

```python
def load_result(
    source: dict,
    format: Optional[str] = None
) -> dict:
    """
    Load a result from file or URL.
    
    Args:
        source: Source specification (kind, location)
        format: Optional format hint (json, yaml, txt)
    
    Returns:
        Loaded result dictionary
    
    Raises:
        ValueError: Invalid source or format
        IOError: File/URL access error
    """
```

**Source Types:**

- `filesystem`: Load from local file path
- `url`: Load from HTTP/HTTPS URL

#### Result Parser (`result_parser.py`)

**Purpose**: Parse loaded result data into structured format.

**Features:**

- JSON parsing
- YAML parsing
- Plain text parsing
- Nested metric extraction
- Error recovery (partial parse on failure)

**API:**

```python
def parse_result(
    payload: Union[str, dict],
    format: str = "json"
) -> ParsedResult:
    """
    Parse a result payload.
    
    Args:
        payload: Raw result data (string or dict)
        format: Format (json, yaml, yml, txt)
    
    Returns:
        ParsedResult with metrics and metadata
    
    Raises:
        ValueError: Unsupported format or parse error
    """
```

**ParsedResult Structure:**

```python
@dataclass
class ParsedResult:
    metrics: Dict[str, float]      # Extracted numeric metrics
    metadata: Dict[str, Any]       # Non-metric data
    raw: Any                       # Original payload
    format: str                    # Detected format
    errors: List[str]              # Parse errors/warnings
```

**Metric Extraction:**

- Recursively traverse JSON/YAML structure
- Extract all numeric values
- Preserve nested keys as dot-notation (e.g., `system.cpu.usage`)
- Handle arrays (extract statistics: min, max, mean)

### Analyzer (`analyzer/`)

**Purpose**: Extract metrics, perform statistical analysis, and compare results.

#### Result Analyzer (`result_analyzer.py`)

**Purpose**: Extract and analyze metrics from parsed results.

**Features:**

- Statistical analysis (min, max, mean, median, p95, p99, std dev, CV)
- Time series handling
- Histogram generation
- Metric trend detection

**API:**

```python
def analyze_result(parsed_result: ParsedResult) -> AnalysisResult:
    """
    Analyze a parsed result.
    
    Returns:
        AnalysisResult with statistical metrics
    """

@dataclass
class AnalysisResult:
    metrics: Dict[str, MetricStats]
    trends: Dict[str, str]  # "increasing", "decreasing", "stable"
    anomalies: List[str]    # Detected anomalies
```

**MetricStats:**

```python
@dataclass
class MetricStats:
    min: float
    max: float
    mean: float
    median: float
    p95: float
    p99: float
    std_dev: float
    cv: float  # Coefficient of variation
    count: int
```

**Statistical Methods:**

- **Percentiles**: NumPy percentile calculation
- **Standard Deviation**: Population std dev
- **Coefficient of Variation**: `std_dev / mean` (for comparing variability)
- **Trend Detection**: Linear regression over time series

#### Result Comparator (`result_comparator.py`)

**Purpose**: Compare multiple workflow results and rank by metrics.

**Features:**

- Side-by-side metric comparison
- Ranking by optimization direction (higher/lower)
- Percentage change calculation
- Multi-criteria scoring

**API:**

```python
def compare_results(
    results: List[AnalysisResult],
    metric_directions: Dict[str, str]
) -> ComparisonResult:
    """
    Compare multiple analysis results.
    
    Args:
        results: List of analyzed results
        metric_directions: Optimization direction per metric
            ("higher" or "lower")
    
    Returns:
        ComparisonResult with rankings and deltas
    """

@dataclass
class ComparisonResult:
    rankings: Dict[str, List[int]]  # Metric -> ranked indices
    deltas: Dict[str, List[float]]  # Metric -> percentage changes
    best_overall: int               # Best result index (multi-criteria)
    summary: str                    # Human-readable summary
```

**Ranking Algorithm:**

1. For each metric, rank results by optimization direction
2. Assign points based on rank (1st = N points, 2nd = N-1 points, ...)
3. Sum points across all metrics for overall ranking

### Suggestion Engine (`suggester/`)

**Purpose**: Generate AI-driven optimization suggestions.

**Features:**

- Pattern-based suggestions (rules engine)
- Heuristic analysis
- Configuration recommendations
- Impact estimation

**API:**

```python
def generate_suggestions(
    analysis: AnalysisResult,
    comparison: Optional[ComparisonResult] = None,
    metric_directions: Optional[Dict[str, str]] = None
) -> List[Suggestion]:
    """
    Generate optimization suggestions.
    
    Args:
        analysis: Analysis result
        comparison: Optional comparison result (for multi-run)
        metric_directions: Desired optimization directions
    
    Returns:
        List of Suggestion objects
    """

@dataclass
class Suggestion:
    category: str         # "performance", "configuration", "resource"
    priority: str         # "high", "medium", "low"
    metric: str           # Affected metric
    current: float        # Current value
    target: float         # Suggested target
    recommendation: str   # Human-readable recommendation
    rationale: str        # Why this suggestion
    expected_impact: str  # Expected improvement
    confidence: float     # Confidence score (0.0-1.0)
```

**Suggestion Categories:**

1. **Performance**: CPU, memory, I/O optimization
2. **Configuration**: Parameter tuning suggestions
3. **Resource**: Resource allocation recommendations
4. **Architecture**: Design/architecture improvements

**Suggestion Rules:**

- High variability (CV > 0.3) → Suggest stabilization
- Low resource utilization (<50%) → Suggest resource reduction
- High resource utilization (>90%) → Suggest resource increase
- Outliers detected → Suggest investigation
- Performance regression → Suggest rollback or investigation

**Example Suggestions:**

```python
Suggestion(
    category="performance",
    priority="high",
    metric="throughput",
    current=300.0,
    target=450.0,
    recommendation="Increase worker_count from 4 to 8",
    rationale="Current throughput is 40% below optimal",
    expected_impact="30-50% improvement",
    confidence=0.85
)
```

### Database Layer (`db/`)

**Purpose**: Persist historical analysis results for trend analysis.

**Framework**: SQLAlchemy ORM with SQLite (default) or PostgreSQL (production)

#### Models (`models.py`)

**Tables:**

1. **workflow_runs**: Workflow execution records
2. **analysis_results**: Historical analysis data
3. **suggestions**: Historical suggestions

**Schema:**

```python
class WorkflowRun(Base):
    __tablename__ = "workflow_runs"
    
    id = Column(String, primary_key=True)
    workflow_id = Column(String, nullable=False)
    tenant_id = Column(String, nullable=True)
    timestamp = Column(DateTime, default=datetime.utcnow)
    input = Column(JSON)
    result = Column(JSON)
    
class AnalysisResult(Base):
    __tablename__ = "analysis_results"
    
    id = Column(String, primary_key=True)
    run_id = Column(String, ForeignKey("workflow_runs.id"))
    metrics = Column(JSON)
    trends = Column(JSON)
    timestamp = Column(DateTime, default=datetime.utcnow)
    
class Suggestion(Base):
    __tablename__ = "suggestions"
    
    id = Column(String, primary_key=True)
    analysis_id = Column(String, ForeignKey("analysis_results.id"))
    category = Column(String)
    priority = Column(String)
    metric = Column(String)
    recommendation = Column(Text)
    confidence = Column(Float)
```

#### Repository (`repository.py`)

**Purpose**: Data access layer for historical data.

**API:**

```python
class HistoryRepository:
    def save_run(self, run: WorkflowRun) -> None
    def save_analysis(self, analysis: AnalysisResult) -> None
    def save_suggestion(self, suggestion: Suggestion) -> None
    
    def get_run(self, run_id: str) -> Optional[WorkflowRun]
    def get_runs_by_workflow(
        self, workflow_id: str,
        limit: int = 100
    ) -> List[WorkflowRun]
    
    def get_trend_data(
        self, workflow_id: str,
        metric: str,
        days: int = 30
    ) -> List[Tuple[datetime, float]]
```

#### Session Management (`session.py`)

**Purpose**: Database connection and transaction management.

**Configuration:**

```python
# SQLite (default, development)
DATABASE_URL = "sqlite:///./arcaflow_analysis.db"

# PostgreSQL (production)
DATABASE_URL = "postgresql://user:pass@host:5432/dbname"
```

**Session Lifecycle:**

```python
# Dependency injection pattern
def get_db() -> Generator[Session, None, None]:
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
```

## Data Flow

### Single Result Analysis

```mermaid
sequenceDiagram
    Go Server->>+HTTP API: POST /analyze (result)
    HTTP API->>+Loader: load_result(source)
    Loader-->>-HTTP API: raw data
    HTTP API->>+Parser: parse_result(data)
    Parser-->>-HTTP API: ParsedResult
    HTTP API->>+Analyzer: analyze_result(parsed)
    Analyzer-->>-HTTP API: AnalysisResult
    HTTP API->>+Suggester: generate_suggestions(analysis)
    Suggester-->>-HTTP API: suggestions
    HTTP API->>+Repository: save_analysis(result)
    Repository-->>-HTTP API: saved
    HTTP API-->>-Go Server: analysis + suggestions
```

### Multi-Result Comparison

```mermaid
sequenceDiagram
    Go Server->>+HTTP API: POST /compare (results[])
    loop For each result
        HTTP API->>Parser: parse_result
        Parser-->>HTTP API: ParsedResult
        HTTP API->>Analyzer: analyze_result
        Analyzer-->>HTTP API: AnalysisResult
    end
    HTTP API->>+Comparator: compare_results(analyses)
    Comparator-->>-HTTP API: ComparisonResult
    HTTP API->>+Suggester: generate_suggestions (with comparison)
    Suggester-->>-HTTP API: suggestions
    HTTP API-->>-Go Server: comparison + suggestions
```

## Configuration

**Environment Variables:**

- `DATABASE_URL`: Database connection string
- `ANALYSIS_PORT`: HTTP server port (default: 8081)
- `LOG_LEVEL`: Logging level (DEBUG, INFO, WARNING, ERROR)

**Configuration File:**

```yaml
# analysis/config.yaml
database:
  url: "sqlite:///./arcaflow_analysis.db"
  echo: false  # Log SQL queries

server:
  host: "0.0.0.0"
  port: 8081
  
logging:
  level: "INFO"
  format: "json"  # or "text"
```

## Testing Strategy

### Unit Tests

**Framework**: pytest

**Coverage**: >85% required

**Test Structure:**

```python
# tests/test_result_parser.py
def test_parse_json_result():
    payload = {"metric1": 100, "metric2": 200}
    result = parse_result(payload, format="json")
    assert result.metrics == {"metric1": 100.0, "metric2": 200.0}

# Parameterized tests
@pytest.mark.parametrize("format,payload", [
    ("json", '{"metric": 100}'),
    ("yaml", 'metric: 100'),
])
def test_parse_formats(format, payload):
    result = parse_result(payload, format=format)
    assert result.metrics["metric"] == 100.0
```

### Integration Tests

**Purpose**: Test HTTP API end-to-end.

**Framework**: pytest with HTTP test client

```python
# tests/test_http_api.py
def test_analyze_endpoint(client):
    response = client.post("/analyze", json={
        "results": [{"payload": {"metric": 100}, "format": "json"}]
    })
    assert response.status_code == 200
    data = response.json()
    assert "metrics" in data
    assert "suggestions" in data
```

### Fixtures

**Purpose**: Reusable test data and mocks.

```python
# tests/conftest.py
@pytest.fixture
def sample_result():
    return ParsedResult(
        metrics={"throughput": 300.0, "latency": 50.0},
        metadata={},
        raw={},
        format="json",
        errors=[]
    )

@pytest.fixture
def db_session():
    # Create in-memory SQLite for tests
    engine = create_engine("sqlite:///:memory:")
    Base.metadata.create_all(engine)
    Session = sessionmaker(bind=engine)
    session = Session()
    yield session
    session.close()
```

## Performance Considerations

### Bottlenecks

- **Large Result Parsing**: Memory-intensive for large JSON/YAML
- **Database Queries**: Can be slow for historical trend analysis
- **Concurrent Requests**: Limited by synchronous HTTP server

### Optimization Strategies

- **Streaming Parsing**: For very large result files (future)
- **Database Indexing**: Indexed queries on workflow_id, timestamp
- **Connection Pooling**: Reuse database connections
- **Async HTTP Server**: Upgrade to asyncio-based server (future)
- **Caching**: Cache frequently accessed analyses

## Security Considerations

### Input Validation

- Validate all API inputs
- Limit result size to prevent DoS
- Sanitize file paths (prevent traversal)

### Database

- Use parameterized queries (SQLAlchemy ORM)
- No raw SQL execution
- Encrypt sensitive data at rest (future)

### API Security

- No authentication currently (trust Go server)
- Future: Mutual TLS or API keys

## Future Enhancements

1. **Async HTTP Server**: Upgrade to aiohttp/FastAPI for better concurrency
2. **Advanced ML**: Machine learning models for suggestions
3. **Real-Time Analysis**: WebSocket streaming for live results
4. **Multi-Database**: Redis for caching, PostgreSQL for persistence
5. **Distributed Analysis**: Horizontal scaling with task queue (Celery)

## Related Documentation

- [Architecture Overview](overview.md) - High-level system architecture
- [Go Server Architecture](go-server.md) - Go server details
- [Data Flow](data-flow.md) - End-to-end data flow diagrams
- [Inter-Service Communication](inter-service.md) - Go ↔ Python HTTP protocol

---

*For API reference and generated docs, see the [API Documentation](../api/python-engine.md).*
