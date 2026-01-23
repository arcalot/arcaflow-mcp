"""HTTP-friendly analysis handlers."""

from __future__ import annotations

from dataclasses import asdict
from typing import Any

import yaml

import json

from arcaflow_analysis.analyzer.result_analyzer import ResultAnalyzer
from arcaflow_analysis.analyzer.result_comparator import ResultComparator
from arcaflow_analysis.db.repository import HistoryRepository
from arcaflow_analysis.parser.result_parser import ResultParser
from arcaflow_analysis.suggester.suggestion_engine import SuggestionEngine


def analyze_results(request: dict[str, Any]) -> dict[str, Any]:
    """Analyze results provided in a request payload."""
    parser = ResultParser()
    analyzer = ResultAnalyzer()
    comparator = ResultComparator()
    suggester = SuggestionEngine()

    results_payload = request.get("results", [])
    parsed_results = [
        parser.parse(_result_entry(payload, idx))
        for idx, payload in enumerate(results_payload)
    ]

    analysis_summary = analyzer.analyze(parsed_results)
    comparison_summary = None
    if request.get("compare"):
        comparison_summary = comparator.compare(
            parsed_results,
            request.get("metric_directions"),
        )

    suggestion_summary = suggester.suggest(analysis_summary, comparison_summary)

    response: dict[str, Any] = {
        "analysis": _analysis_to_dict(analysis_summary),
        "suggestions": [asdict(s) for s in suggestion_summary.suggestions],
    }
    if comparison_summary is not None:
        response["comparison"] = _comparison_to_dict(comparison_summary)
    return response


def list_history(
    repo: HistoryRepository, workflow_id: str | None = None
) -> dict[str, Any]:
    """List history summaries for stored runs."""
    runs = repo.list_runs(workflow_id)
    return {"runs": [_summary_to_dict(run) for run in runs]}


def get_history(repo: HistoryRepository, run_id: str) -> dict[str, Any]:
    """Fetch a stored run record by ID."""
    record = repo.get_run(run_id)
    if record is None:
        return {"run": None}
    return {
        "run": {
            "run_id": record.id,
            "workflow_id": record.workflow_id,
            "created_at": record.created_at.isoformat(),
            "input_payload": record.input_payload,
            "metrics": record.metrics,
        }
    }


def add_history(
    repo: HistoryRepository,
    workflow_id: str,
    input_payload: dict[str, Any],
    metrics: dict[str, Any],
) -> dict[str, Any]:
    """Persist a workflow run in history."""
    summary = repo.add_run(workflow_id, input_payload, metrics)
    return {"run": _summary_to_dict(summary)}


def _summary_to_dict(summary) -> dict[str, Any]:
    """Serialize a history summary to a JSON-friendly dict."""
    return {
        "run_id": summary.run_id,
        "workflow_id": summary.workflow_id,
        "created_at": summary.created_at.isoformat(),
        "metrics": summary.metrics,
    }


def _analysis_to_dict(summary) -> dict[str, Any]:
    """Serialize an analysis summary to a JSON-friendly dict."""
    return {
        "success_rate": summary.success_rate,
        "metric_stats": summary.metric_stats,
        "record_count": summary.record_count,
        "findings": [asdict(f) for f in summary.findings],
    }


def _comparison_to_dict(summary) -> dict[str, Any]:
    """Serialize a comparison summary to a JSON-friendly dict."""
    return {
        "metric_stats": summary.metric_stats,
        "rankings": summary.rankings,
        "findings": [asdict(f) for f in summary.findings],
    }


def _result_entry(payload: dict[str, Any], idx: int) -> Any:
    """Build a lightweight result entry compatible with the parser API."""
    raw_payload = payload.get("payload")
    format_hint = payload.get("format")
    parsed = _parse_payload(raw_payload, format_hint)
    return type(
        "ResultEntry",
        (),
        {
            "path": f"inline-{idx}",
            "format": format_hint or "unknown",
            "payload": parsed,
        },
    )()


def _parse_payload(raw: Any, format_hint: str | None) -> Any:
    """Parse a raw payload value using the provided format hint."""
    if format_hint == "json":
        if isinstance(raw, str):
            return json.loads(raw)
        return raw
    if format_hint in {"yaml", "yml"}:
        if isinstance(raw, str):
            return yaml.safe_load(raw)
        return raw
    if format_hint == "log":
        return raw if isinstance(raw, str) else ""
    if isinstance(raw, str):
        try:
            return json.loads(raw)
        except json.JSONDecodeError:
            try:
                return yaml.safe_load(raw)
            except yaml.YAMLError:
                return raw
    return raw
