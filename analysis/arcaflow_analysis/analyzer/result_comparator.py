"""Compare multiple parsed workflow results."""

from __future__ import annotations

from dataclasses import dataclass, field

import pandas as pd

from arcaflow_analysis.analyzer.result_analyzer import AnalysisFinding
from arcaflow_analysis.parser.result_parser import ParsedResult


@dataclass(frozen=True)
class ComparisonSummary:
    """Summary of multi-run comparison results."""

    metric_table: pd.DataFrame | None
    metric_stats: dict[str, dict[str, float]]
    rankings: dict[str, list[tuple[str, float]]]
    findings: list[AnalysisFinding] = field(default_factory=list)


class ResultComparator:
    """Compare multiple parsed results and rank metrics."""

    def compare(
        self,
        results: list[ParsedResult],
        metric_directions: dict[str, str] | None = None,
    ) -> ComparisonSummary:
        """Compare multiple runs and summarize metric rankings."""
        if not results:
            return ComparisonSummary(
                metric_table=None,
                metric_stats={},
                rankings={},
                findings=[
                    AnalysisFinding(
                        severity="info",
                        message="No results provided for comparison.",
                    )
                ],
            )

        metric_table = self._build_metric_table(results)
        if metric_table.empty:
            return ComparisonSummary(
                metric_table=None,
                metric_stats={},
                rankings={},
                findings=[
                    AnalysisFinding(
                        severity="info",
                        message="No numeric metrics available for comparison.",
                    )
                ],
            )

        metric_stats = self._metric_stats(metric_table)
        rankings = self._rank_metrics(metric_table, metric_directions or {})
        findings = self._find_missing_metrics(metric_table)

        return ComparisonSummary(
            metric_table=metric_table,
            metric_stats=metric_stats,
            rankings=rankings,
            findings=findings,
        )

    def _build_metric_table(self, results: list[ParsedResult]) -> pd.DataFrame:
        """Build a dataframe of numeric metrics across results."""
        rows: list[dict[str, float]] = []
        index: list[str] = []
        for result in results:
            run_id = str(result.source.path) if result.source else "unknown"
            numeric_metrics = {
                key: float(value)
                for key, value in result.metrics.items()
                if isinstance(value, (int, float))
            }
            rows.append(numeric_metrics)
            index.append(run_id)
        return pd.DataFrame(rows, index=index)

    def _metric_stats(self, table: pd.DataFrame) -> dict[str, dict[str, float]]:
        """Compute metric statistics from a comparison table."""
        stats: dict[str, dict[str, float]] = {}
        for column in table.columns:
            series = table[column].dropna()
            if series.empty:
                continue
            stats[column] = {
                "min": float(series.min()),
                "max": float(series.max()),
                "mean": float(series.mean()),
                "p95": float(series.quantile(0.95)),
            }
        return stats

    def _rank_metrics(
        self, table: pd.DataFrame, metric_directions: dict[str, str]
    ) -> dict[str, list[tuple[str, float]]]:
        """Rank metrics according to configured directions."""
        rankings: dict[str, list[tuple[str, float]]] = {}
        for column in table.columns:
            series = table[column].dropna()
            if series.empty:
                continue
            direction = metric_directions.get(column, "lower")
            ascending = direction != "higher"
            ordered = series.sort_values(ascending=ascending)
            rankings[column] = list(ordered.items())
        return rankings

    def _find_missing_metrics(self, table: pd.DataFrame) -> list[AnalysisFinding]:
        """Identify metrics missing in some runs."""
        findings: list[AnalysisFinding] = []
        for column in table.columns:
            missing = int(table[column].isna().sum())
            if missing > 0:
                findings.append(
                    AnalysisFinding(
                        severity="warning",
                        message="Metric missing for some runs.",
                        metric=column,
                        value=float(missing),
                    )
                )
        if not findings:
            findings.append(
                AnalysisFinding(
                    severity="info",
                    message="All metrics present across runs.",
                )
            )
        return findings
