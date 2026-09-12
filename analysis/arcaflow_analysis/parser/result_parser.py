"""Parse workflow results into structured metrics."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import pandas as pd

from arcaflow_analysis.parser.result_loader import ResultEntry, ResultPayload

MetricValue = int | float | str | bool | None


@dataclass(frozen=True)
class ParsedResult:
    """Normalized result payload with extracted metrics."""

    source: ResultEntry
    success: bool | None
    metrics: dict[str, MetricValue]
    records: pd.DataFrame | None


class ResultParser:
    """Extract metrics and structured records from workflow results."""

    def parse(self, entry: ResultEntry) -> ParsedResult:
        """Parse a loaded result entry into structured metrics."""
        payload = entry.payload
        metrics: dict[str, MetricValue] = {}
        records: pd.DataFrame | None = None
        success = self._extract_success(payload)

        if isinstance(payload, dict):
            metrics.update(self._extract_metrics(payload))
            records = self._extract_records(payload)
        elif isinstance(payload, list):
            records = self._records_from_list(payload)
        else:
            metrics["raw_text"] = payload if isinstance(payload, str) else None

        return ParsedResult(
            source=entry,
            success=success,
            metrics=metrics,
            records=records,
        )

    def _extract_success(self, payload: ResultPayload) -> bool | None:
        """Extract a boolean success signal from a result payload."""
        if isinstance(payload, dict):
            for key in ("success", "ok", "passed"):
                value = payload.get(key)
                if isinstance(value, bool):
                    return value
            status = payload.get("status")
            if isinstance(status, str):
                lowered = status.lower()
                if lowered in {"ok", "success", "passed"}:
                    return True
                if lowered in {"failed", "error", "failure"}:
                    return False
        return None

    def _extract_metrics(self, payload: dict[str, Any]) -> dict[str, MetricValue]:
        """Extract numeric metrics from a structured payload."""
        metrics: dict[str, MetricValue] = {}
        if isinstance(payload.get("metrics"), dict):
            metrics.update(self._flatten_metrics(payload["metrics"]))
        if isinstance(payload.get("results"), dict):
            metrics.update(self._flatten_metrics(payload["results"]))
        if "duration_ms" in payload:
            metrics["duration_ms"] = payload.get("duration_ms")
        return metrics

    def _flatten_metrics(self, raw: dict[str, Any]) -> dict[str, MetricValue]:
        """Flatten nested metrics to a single-level map."""
        flattened: dict[str, MetricValue] = {}
        for key, value in raw.items():
            if isinstance(value, (int, float, str, bool)) or value is None:
                flattened[key] = value
            elif isinstance(value, dict):
                for child_key, child_value in value.items():
                    name = f"{key}.{child_key}"
                    if (
                        isinstance(child_value, (int, float, str, bool))
                        or child_value is None
                    ):
                        flattened[name] = child_value
        return flattened

    def _extract_records(self, payload: dict[str, Any]) -> pd.DataFrame | None:
        """Extract tabular records from known keys."""
        for key in ("records", "rows", "samples"):
            value = payload.get(key)
            records = self._records_from_list(value)
            if records is not None:
                return records
        return None

    def _records_from_list(self, value: Any) -> pd.DataFrame | None:
        """Build a DataFrame from a list of dict records."""
        if isinstance(value, list) and value:
            if all(isinstance(item, dict) for item in value):
                return pd.DataFrame(value)
        return None
