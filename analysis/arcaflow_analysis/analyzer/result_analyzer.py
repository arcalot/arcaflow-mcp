"""Analyze parsed workflow results."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

import pandas as pd

from arcaflow_analysis.parser.result_parser import ParsedResult


@dataclass(frozen=True)
class AnalysisFinding:
    """A single analysis finding derived from parsed results."""

    severity: str
    message: str
    metric: str | None = None
    value: float | None = None
    details: dict[str, Any] = field(default_factory=dict)


@dataclass(frozen=True)
class AnalysisSummary:
    """High-level summary of analysis results."""

    success_rate: float | None
    metric_stats: dict[str, dict[str, float]]
    record_count: int
    findings: list[AnalysisFinding]


class ResultAnalyzer:
    """Analyze parsed results to produce summary insights."""

    def analyze(self, results: list[ParsedResult]) -> AnalysisSummary:
        """Analyze results and return summary statistics."""
        if not results:
            return AnalysisSummary(
                success_rate=None,
                metric_stats={},
                record_count=0,
                findings=[
                    AnalysisFinding(
                        severity="info",
                        message="No results provided for analysis.",
                    )
                ],
            )

        success_rate = self._success_rate(results)
        metric_stats = self._metric_stats(results)
        record_count = sum(self._record_count(result) for result in results)
        findings = self._build_findings(success_rate, metric_stats, record_count)

        return AnalysisSummary(
            success_rate=success_rate,
            metric_stats=metric_stats,
            record_count=record_count,
            findings=findings,
        )

    def _success_rate(self, results: list[ParsedResult]) -> float | None:
        """Compute success rate for parsed results."""
        values = [result.success for result in results if result.success is not None]
        if not values:
            return None
        return sum(1 for value in values if value) / len(values)

    def _metric_stats(self, results: list[ParsedResult]) -> dict[str, dict[str, float]]:
        """Aggregate numeric metrics across results."""
        metrics: dict[str, list[float]] = {}
        for result in results:
            for key, value in result.metrics.items():
                if isinstance(value, (int, float)):
                    metrics.setdefault(key, []).append(float(value))

        stats: dict[str, dict[str, float]] = {}
        for key, values in metrics.items():
            series = pd.Series(values)
            stats[key] = {
                "min": float(series.min()),
                "max": float(series.max()),
                "mean": float(series.mean()),
                "p95": float(series.quantile(0.95)),
            }
        return stats

    def _record_count(self, result: ParsedResult) -> int:
        """Return the record count for a parsed result."""
        if result.records is None:
            return 0
        return int(result.records.shape[0])

    def _build_findings(
        self,
        success_rate: float | None,
        metric_stats: dict[str, dict[str, float]],
        record_count: int,
    ) -> list[AnalysisFinding]:
        """Build analysis findings from derived statistics."""
        findings: list[AnalysisFinding] = []
        if success_rate is None:
            findings.append(
                AnalysisFinding(
                    severity="info",
                    message="No success indicators found in results.",
                )
            )
        elif success_rate < 1.0:
            findings.append(
                AnalysisFinding(
                    severity="warning",
                    message="Some runs reported failures.",
                    metric="success_rate",
                    value=success_rate,
                )
            )
        if record_count == 0:
            findings.append(
                AnalysisFinding(
                    severity="info",
                    message="No tabular records found in results.",
                )
            )
        for metric, stats in metric_stats.items():
            if stats["p95"] > stats["mean"] * 1.5:
                findings.append(
                    AnalysisFinding(
                        severity="info",
                        message="High variability detected in metric.",
                        metric=metric,
                        value=stats["p95"],
                        details=stats,
                    )
                )
        if not findings:
            findings.append(
                AnalysisFinding(
                    severity="info",
                    message="No issues detected in analyzed results.",
                )
            )
        return findings
