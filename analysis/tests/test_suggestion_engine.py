"""Tests for suggestion engine."""

from __future__ import annotations

# pylint: disable=import-error
# mypy: disable-error-code=arg-type

from arcaflow_analysis.analyzer.result_analyzer import AnalysisSummary
from arcaflow_analysis.analyzer.result_comparator import ComparisonSummary
from arcaflow_analysis.suggester.suggestion_engine import SuggestionEngine


def test_suggest_from_failure() -> None:
    engine = SuggestionEngine()
    analysis = AnalysisSummary(
        success_rate=0.5,
        metric_stats={},
        record_count=1,
        findings=[],
    )

    summary = engine.suggest(analysis)

    assert any(s.priority == "high" for s in summary.suggestions)


def test_suggest_from_metrics() -> None:
    engine = SuggestionEngine()
    analysis = AnalysisSummary(
        success_rate=1.0,
        metric_stats={"latency": {"min": 1, "max": 10, "mean": 2, "p95": 5}},
        record_count=1,
        findings=[],
    )

    summary = engine.suggest(analysis)

    assert any("latency" in s.title for s in summary.suggestions)


def test_suggest_from_comparison() -> None:
    engine = SuggestionEngine()
    analysis = AnalysisSummary(
        success_rate=None,
        metric_stats={},
        record_count=0,
        findings=[],
    )
    comparison = ComparisonSummary(
        metric_table=None,
        metric_stats={},
        rankings={"throughput": [("run-a", 123.0)]},
        findings=[],
    )

    summary = engine.suggest(analysis, comparison)

    assert any("throughput" in s.title for s in summary.suggestions)
