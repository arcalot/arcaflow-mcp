# Python Analysis Engine API Reference

API documentation for the Python analysis engine modules.

## Overview

The Python analysis engine processes workflow results, extracts metrics, performs statistical analysis, and generates optimization suggestions. All modules are documented using NumPy-style docstrings.

## Viewing Documentation

### Online (Read the Docs)

Once published, the API documentation will be available at:

```
https://arcaflow-mcp.readthedocs.io/
```

### Local (Sphinx)

Generate and view documentation locally:

```bash
cd analysis

# Install Sphinx
poetry add --group dev sphinx sphinx-rtd-theme

# Generate docs
sphinx-apidoc -o docs/source arcaflow_analysis
cd docs
make html

# Open in browser
open build/html/index.html
```

### IDE Integration

**VS Code:**

- Install `Python` extension
- Hover over symbols for inline documentation
- Use `Go to Definition` (F12) to view source

**PyCharm:**

- Quick Documentation: Ctrl+Q (Cmd+J on macOS)
- Parameter Info: Ctrl+P (Cmd+P on macOS)
- External Documentation: Shift+F1

## Module Structure

```
analysis/arcaflow_analysis/
├── server/              # HTTP API server
│   ├── app.py           # Application setup
│   ├── http_api.py      # API endpoint handlers
│   └── http_server.py   # HTTP server main
├── parser/              # Result parsing
│   ├── result_loader.py # Load from file/URL
│   └── result_parser.py # Parse JSON/YAML/text
├── analyzer/            # Analysis logic
│   ├── result_analyzer.py    # Metrics extraction
│   └── result_comparator.py  # Multi-run comparison
├── suggester/           # Optimization suggestions
│   └── suggestion_engine.py
└── db/                  # Persistence
    ├── models.py        # SQLAlchemy models
    ├── repository.py    # Data access layer
    └── session.py       # DB session management
```

## Core Modules

### server.http_api

**Module:** `arcaflow_analysis.server.http_api`

**Purpose:** REST API endpoint handlers

**Key Functions:**

- `handle_analyze()` - Analyze single result
- `handle_compare()` - Compare multiple results
- `handle_suggest()` - Generate suggestions
- `handle_metrics()` - Extract metrics only
- `handle_parse()` - Parse result

**Example:**

```python
from arcaflow_analysis.server.http_api import handle_analyze

# Handle analysis request
response = handle_analyze(request_data)
# Returns: {"metrics": {...}, "suggestions": [...]}
```

### parser.result_parser

**Module:** `arcaflow_analysis.parser.result_parser`

**Purpose:** Parse workflow results from various formats

**Key Classes:**

- `ParsedResult` - Parsed result dataclass
- `ResultParser` - Main parser class

**Example:**

```python
from arcaflow_analysis.parser.result_parser import parse_result

# Parse JSON result
result = parse_result(
    payload={"throughput": 300, "latency": 50},
    format="json"
)

# Access metrics
print(result.metrics)  # {"throughput": 300.0, "latency": 50.0}
```

### analyzer.result_analyzer

**Module:** `arcaflow_analysis.analyzer.result_analyzer`

**Purpose:** Extract and analyze metrics

**Key Classes:**

- `AnalysisResult` - Analysis result dataclass
- `MetricStats` - Statistical metrics dataclass
- `ResultAnalyzer` - Main analyzer class

**Example:**

```python
from arcaflow_analysis.analyzer.result_analyzer import analyze_result

# Analyze parsed result
analysis = analyze_result(parsed_result)

# Access statistics
stats = analysis.metrics["throughput"]
print(f"Mean: {stats.mean}, P95: {stats.p95}")
```

### analyzer.result_comparator

**Module:** `arcaflow_analysis.analyzer.result_comparator`

**Purpose:** Compare multiple workflow results

**Key Classes:**

- `ComparisonResult` - Comparison result dataclass
- `ResultComparator` - Main comparator class

**Example:**

```python
from arcaflow_analysis.analyzer.result_comparator import compare_results

# Compare multiple results
comparison = compare_results(
    results=[analysis1, analysis2, analysis3],
    metric_directions={"throughput": "higher", "latency": "lower"}
)

# Get rankings
print(comparison.rankings)  # {"throughput": [1, 0, 2], ...}
print(f"Best overall: {comparison.best_overall}")
```

### suggester.suggestion_engine

**Module:** `arcaflow_analysis.suggester.suggestion_engine`

**Purpose:** Generate AI-driven optimization suggestions

**Key Classes:**

- `Suggestion` - Suggestion dataclass
- `SuggestionEngine` - Main engine class

**Example:**

```python
from arcaflow_analysis.suggester.suggestion_engine import generate_suggestions

# Generate suggestions
suggestions = generate_suggestions(
    analysis=analysis_result,
    metric_directions={"throughput": "higher"}
)

# Print suggestions
for suggestion in suggestions:
    print(f"{suggestion.priority}: {suggestion.recommendation}")
```

### db.repository

**Module:** `arcaflow_analysis.db.repository`

**Purpose:** Data access layer for historical data

**Key Classes:**

- `HistoryRepository` - Repository class

**Example:**

```python
from arcaflow_analysis.db.repository import HistoryRepository
from arcaflow_analysis.db.session import get_db

# Create repository
db = next(get_db())
repo = HistoryRepository(db)

# Save analysis
repo.save_analysis(analysis_result)

# Query history
runs = repo.get_runs_by_workflow("workflow-id", limit=10)
```

## API Endpoints

For HTTP API endpoint documentation, see the [Inter-Service Communication](../architecture/inter-service.md) document.

## Data Models

### ParsedResult

```python
@dataclass
class ParsedResult:
    """Parsed workflow result.
    
    Attributes:
        metrics: Extracted numeric metrics (flat dict)
        metadata: Non-metric data (arbitrary structure)
        raw: Original payload (unmodified)
        format: Detected format ("json", "yaml", "txt")
        errors: Parse errors or warnings
    """
    metrics: Dict[str, float]
    metadata: Dict[str, Any]
    raw: Any
    format: str
    errors: List[str]
```

### AnalysisResult

```python
@dataclass
class AnalysisResult:
    """Result of metric analysis.
    
    Attributes:
        metrics: Statistical metrics per metric name
        trends: Trend detection ("increasing", "decreasing", "stable")
        anomalies: Detected anomalies (outliers, spikes)
    """
    metrics: Dict[str, MetricStats]
    trends: Dict[str, str]
    anomalies: List[str]
```

### MetricStats

```python
@dataclass
class MetricStats:
    """Statistical metrics for a single metric.
    
    Attributes:
        min: Minimum value
        max: Maximum value
        mean: Mean (average)
        median: Median (50th percentile)
        p95: 95th percentile
        p99: 99th percentile
        std_dev: Standard deviation
        cv: Coefficient of variation (std_dev / mean)
        count: Sample count
    """
    min: float
    max: float
    mean: float
    median: float
    p95: float
    p99: float
    std_dev: float
    cv: float
    count: int
```

### Suggestion

```python
@dataclass
class Suggestion:
    """Optimization suggestion.
    
    Attributes:
        category: Suggestion category ("performance", "configuration", "resource")
        priority: Priority level ("high", "medium", "low")
        metric: Affected metric name
        current: Current metric value
        target: Suggested target value
        recommendation: Human-readable recommendation
        rationale: Explanation of why this suggestion
        expected_impact: Expected improvement description
        confidence: Confidence score (0.0-1.0)
    """
    category: str
    priority: str
    metric: str
    current: float
    target: float
    recommendation: str
    rationale: str
    expected_impact: str
    confidence: float
```

## Generating Documentation

### Sphinx Documentation

```bash
cd analysis

# Install Sphinx and dependencies
poetry add --group dev sphinx sphinx-rtd-theme sphinx-autodoc-typehints

# Create docs structure
mkdir -p docs/source
sphinx-quickstart docs

# Configure conf.py
cat > docs/source/conf.py <<EOF
import os
import sys
sys.path.insert(0, os.path.abspath('../..'))

project = 'Arcaflow Analysis Engine'
extensions = [
    'sphinx.ext.autodoc',
    'sphinx.ext.napoleon',
    'sphinx.ext.viewcode',
    'sphinx_autodoc_typehints',
]

html_theme = 'sphinx_rtd_theme'
EOF

# Generate API docs
sphinx-apidoc -o docs/source arcaflow_analysis

# Build HTML
cd docs
make html
```

### MkDocs (Alternative)

```bash
cd analysis

# Install MkDocs
poetry add --group dev mkdocs mkdocs-material mkdocstrings[python]

# Create mkdocs.yml
cat > mkdocs.yml <<EOF
site_name: Arcaflow Analysis Engine
theme:
  name: material

plugins:
  - mkdocstrings:
      handlers:
        python:
          paths: [.]
          options:
            show_source: true

nav:
  - Home: index.md
  - API Reference:
    - Server: api/server.md
    - Parser: api/parser.md
    - Analyzer: api/analyzer.md
    - Suggester: api/suggester.md
EOF

# Build docs
mkdocs build

# Serve locally
mkdocs serve
```

### API Docs from Docstrings

```bash
# Install pydoc
# (included with Python)

# Generate HTML for a module
python -m pydoc -w arcaflow_analysis.analyzer.result_analyzer

# Start HTTP server
python -m pydoc -p 8000
# Open http://localhost:8000/arcaflow_analysis.html
```

## Documentation Standards

All modules, classes, functions, and methods must have NumPy-style docstrings:

```python
def analyze_result(parsed_result: ParsedResult) -> AnalysisResult:
    """Analyze a parsed workflow result.
    
    Extracts statistical metrics (min, max, mean, percentiles, std dev, CV)
    for all numeric metrics in the result. Detects trends and anomalies.
    
    Parameters
    ----------
    parsed_result : ParsedResult
        The parsed result to analyze
    
    Returns
    -------
    AnalysisResult
        Analysis result with statistics, trends, and anomalies
    
    Raises
    ------
    ValueError
        If the parsed result contains no numeric metrics
    
    Examples
    --------
    >>> parsed = parse_result({"throughput": 300}, format="json")
    >>> analysis = analyze_result(parsed)
    >>> print(analysis.metrics["throughput"].mean)
    300.0
    
    Notes
    -----
    The coefficient of variation (CV) is calculated as std_dev / mean.
    High CV (> 0.3) indicates high variability in the metric.
    
    See Also
    --------
    parse_result : Parse a workflow result
    compare_results : Compare multiple analysis results
    """
    # Implementation...
```

## Type Hints

All functions and methods must include complete type hints:

```python
from typing import Dict, List, Optional, Union, Any

def parse_result(
    payload: Union[str, dict],
    format: str = "json"
) -> ParsedResult:
    """Parse a result payload."""
    # ...
```

## Testing

For testing guidelines and examples, see the [Testing Guide](../development/testing.md).

```bash
# Run tests
cd analysis
poetry run pytest

# Generate coverage report
poetry run pytest --cov=arcaflow_analysis --cov-report=html
open htmlcov/index.html
```

## Related Documentation

- [Python Analysis Engine Architecture](../architecture/python-engine.md) - Architectural overview
- [Development Setup](../development/setup.md) - Setting up development environment
- [Testing Guide](../development/testing.md) - Writing and running tests

---

*For Go server API, see [go-server.md](go-server.md).*
