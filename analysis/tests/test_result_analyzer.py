"""Tests for the result analyzer."""

from __future__ import annotations

# pylint: disable=import-error
# mypy: disable-error-code=arg-type

import pandas as pd

from arcaflow_analysis.analyzer.result_analyzer import ResultAnalyzer
from arcaflow_analysis.parser.result_parser import ParsedResult


def _parsed(
    metrics: dict[str, object],
    success: bool | None = None,
    records: pd.DataFrame | None = None,
) -> ParsedResult:
    return ParsedResult(
        source=None,  # type: ignore[arg-type]
        success=success,
        metrics=metrics,
        records=records,
    )


def test_analyze_empty_results() -> None:
    analyzer = ResultAnalyzer()
    summary = analyzer.analyze([])

    assert summary.success_rate is None
    assert summary.record_count == 0
    assert summary.findings


def test_analyze_success_rate() -> None:
    analyzer = ResultAnalyzer()
    results = [
        _parsed(metrics={}, success=True),
        _parsed(metrics={}, success=False),
        _parsed(metrics={}, success=True),
    ]

    summary = analyzer.analyze(results)

    assert summary.success_rate == 2 / 3
    assert any(f.metric == "success_rate" for f in summary.findings)


def test_analyze_metric_stats() -> None:
    analyzer = ResultAnalyzer()
    results = [
        _parsed(metrics={"latency": 10}),
        _parsed(metrics={"latency": 20}),
    ]

    summary = analyzer.analyze(results)

    assert summary.metric_stats["latency"]["mean"] == 15.0
    assert summary.metric_stats["latency"]["max"] == 20.0


def test_analyze_records_count() -> None:
    analyzer = ResultAnalyzer()
    records = pd.DataFrame([{"value": 1}, {"value": 2}])
    results = [_parsed(metrics={}, records=records)]

    summary = analyzer.analyze(results)

    assert summary.record_count == 2
