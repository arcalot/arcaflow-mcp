## Result analysis

This guide will show how to load results, analyze outputs, and generate
suggested input changes.

### Analysis functions

The analysis service is composed of focused functions that build on each other:

- `ResultLoader` loads JSON/YAML/log result files and caches by file fingerprint.
- `ResultParser` extracts success signals, metrics, and tabular records.
- `ResultAnalyzer` summarizes success rates and metric statistics.
- `ResultComparator` ranks multiple runs and flags missing metrics.
- `SuggestionEngine` generates rule-based input suggestions.
- `HistoryRepository` persists run history for trend analysis.

### Result loading

The analysis service loads workflow result files from disk and detects JSON,
YAML, or log formats. Structured formats are parsed into Python objects, and
plain logs remain raw text. Results are cached using file size and modification
timestamp to avoid repeated parsing.

### Result parsing

Parsed results extract:

- Success indicators (e.g., `success`, `status`)
- Flat metrics from `metrics` or `results` sections
- Tabular records from `records`, `rows`, or `samples`

Parsed tables are represented as pandas DataFrames for downstream analysis.

### Result analysis

The analyzer summarizes:

- Success rate across runs (when available)
- Basic metric statistics (min/max/mean/p95)
- Record counts for tabular data

Findings highlight missing success signals, missing records, or unusual metric
variability for further investigation.

### Multi-run comparison

The comparator builds a per-run metric table and ranks runs per metric. Use the
metric directions (higher/lower) to rank throughput vs latency correctly.

### Suggestions

The suggestion engine is rule-based and surfaces:

- Failures that warrant log review
- Metrics with high variability
- Missing sample data or records
- Best-performing run pointers from comparisons

### Historical database

The analysis service stores run history in a SQL database for trend analysis.
Each run persists the workflow ID, input payload, metrics, and timestamp to
support later comparisons.

Persistence requirements:

- Use SQLite for local development and testing.
- Support PostgreSQL for production deployments.
- Persist run history deterministically with UTC timestamps.
- Store enough metadata to support filtering by workflow ID.
- Document retention and cleanup expectations when enabled later.
