"""Tests for the result parser."""

from __future__ import annotations

# pylint: disable=import-error

from pathlib import Path

from arcaflow_analysis.parser.result_loader import ResultEntry
from arcaflow_analysis.parser.result_parser import ResultParser


def _entry(payload: object) -> ResultEntry:
    return ResultEntry(
        path=Path("result.json"),
        format="json",
        payload=payload,
        raw_text="",
    )


def test_parse_metrics_from_dict() -> None:
    parser = ResultParser()
    entry = _entry(
        {
            "status": "ok",
            "metrics": {"latency_ms": 12, "nested": {"p95": 30}},
            "duration_ms": 50,
        }
    )

    parsed = parser.parse(entry)

    assert parsed.success is True
    assert parsed.metrics["latency_ms"] == 12
    assert parsed.metrics["nested.p95"] == 30
    assert parsed.metrics["duration_ms"] == 50


def test_parse_records_from_list() -> None:
    parser = ResultParser()
    entry = _entry([{"value": 1}, {"value": 2}])

    parsed = parser.parse(entry)

    assert parsed.records is not None
    assert parsed.records.shape[0] == 2


def test_parse_records_from_payload_key() -> None:
    parser = ResultParser()
    entry = _entry({"records": [{"value": 5}]})

    parsed = parser.parse(entry)

    assert parsed.records is not None
    assert parsed.records.iloc[0]["value"] == 5


def test_parse_raw_text_metric() -> None:
    parser = ResultParser()
    entry = _entry("raw log text")

    parsed = parser.parse(entry)

    assert parsed.metrics["raw_text"] == "raw log text"
