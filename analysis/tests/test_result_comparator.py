"""Tests for the result comparator."""

from __future__ import annotations

# pylint: disable=import-error
# mypy: disable-error-code=arg-type

from pathlib import Path

from arcaflow_analysis.analyzer.result_comparator import ResultComparator
from arcaflow_analysis.parser.result_parser import ParsedResult


def _parsed(metrics: dict[str, object], path: str) -> ParsedResult:
    return ParsedResult(
        source=type("Source", (), {"path": Path(path)})(),
        success=None,
        metrics=metrics,
        records=None,
    )


def test_compare_metrics() -> None:
    comparator = ResultComparator()
    results = [
        _parsed({"latency": 10, "throughput": 100}, "run-a"),
        _parsed({"latency": 20, "throughput": 150}, "run-b"),
    ]

    summary = comparator.compare(results, {"throughput": "higher"})

    assert summary.metric_table is not None
    assert summary.metric_stats["latency"]["mean"] == 15.0
    assert summary.rankings["latency"][0][0].endswith("run-a")
    assert summary.rankings["throughput"][0][0].endswith("run-b")


def test_compare_missing_metrics() -> None:
    comparator = ResultComparator()
    results = [
        _parsed({"latency": 10}, "run-a"),
        _parsed({}, "run-b"),
    ]

    summary = comparator.compare(results)

    assert any(f.metric == "latency" for f in summary.findings)
