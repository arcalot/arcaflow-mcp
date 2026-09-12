"""Integration tests for analysis pipeline."""

from __future__ import annotations

# pylint: disable=import-error

from pathlib import Path

from arcaflow_analysis.analyzer.result_analyzer import ResultAnalyzer
from arcaflow_analysis.analyzer.result_comparator import ResultComparator
from arcaflow_analysis.parser.result_loader import ResultLoader
from arcaflow_analysis.parser.result_parser import ResultParser
from arcaflow_analysis.suggester.suggestion_engine import SuggestionEngine


def test_analysis_pipeline_with_fixture() -> None:
    repo_root = Path(__file__).resolve().parents[2]
    result_path = repo_root / "test" / "fixtures" / "result-basic.json"

    loader = ResultLoader()
    parser = ResultParser()
    analyzer = ResultAnalyzer()
    comparator = ResultComparator()
    suggester = SuggestionEngine()

    entry = loader.load_file(result_path)
    parsed = parser.parse(entry)

    analysis_summary = analyzer.analyze([parsed])
    comparison_summary = comparator.compare([parsed])
    suggestions = suggester.suggest(analysis_summary, comparison_summary)

    assert analysis_summary.metric_stats
    assert comparison_summary.metric_stats
    assert suggestions.suggestions
