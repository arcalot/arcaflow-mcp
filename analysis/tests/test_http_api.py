"""Tests for HTTP analysis handlers."""

from __future__ import annotations

# pylint: disable=import-error

from arcaflow_analysis.server.http_api import analyze_results


def test_analyze_results_response_shape() -> None:
    response = analyze_results(
        {
            "results": [
                {
                    "format": "json",
                    "payload": {"success": True, "metrics": {"latency": 5}},
                }
            ]
        }
    )

    assert "analysis" in response
    assert "suggestions" in response
