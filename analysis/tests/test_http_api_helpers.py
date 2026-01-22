"""Tests for HTTP analysis helpers."""

from __future__ import annotations

# pylint: disable=import-error

from arcaflow_analysis.server import http_api

# pylint: disable=protected-access


def test_parse_payload_json_string() -> None:
    payload = http_api._parse_payload('{"value": 1}', "json")
    assert payload["value"] == 1


def test_parse_payload_yaml_string() -> None:
    payload = http_api._parse_payload("value: 2\n", "yaml")
    assert payload["value"] == 2


def test_parse_payload_log_fallback() -> None:
    payload = http_api._parse_payload("line", "log")
    assert payload == "line"


def test_parse_payload_infer_yaml() -> None:
    payload = http_api._parse_payload("value: 3\n", None)
    assert payload["value"] == 3


def test_parse_payload_infer_raw() -> None:
    payload = http_api._parse_payload("not-json-or-yaml", None)
    assert payload == "not-json-or-yaml"


def test_comparison_dict_serialization() -> None:
    summary = http_api._comparison_to_dict(
        type(
            "Summary",
            (),
            {
                "metric_stats": {"latency": {"mean": 1.0}},
                "rankings": {"latency": [("run", 1.0)]},
                "findings": [],
            },
        )()
    )
    assert summary["metric_stats"]["latency"]["mean"] == 1.0


def test_analysis_dict_serialization() -> None:
    summary = http_api._analysis_to_dict(
        type(
            "Summary",
            (),
            {
                "success_rate": 1.0,
                "metric_stats": {},
                "record_count": 0,
                "findings": [],
            },
        )()
    )
    assert summary["success_rate"] == 1.0
