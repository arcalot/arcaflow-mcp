"""Generate input suggestions from analysis summaries."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from arcaflow_analysis.analyzer.result_analyzer import AnalysisSummary
from arcaflow_analysis.analyzer.result_comparator import ComparisonSummary


@dataclass(frozen=True)
class Suggestion:
    """A single suggested input change."""

    title: str
    rationale: str
    priority: str
    suggested_change: dict[str, Any] = field(default_factory=dict)


@dataclass(frozen=True)
class SuggestionSummary:
    """Collection of suggestions generated from analysis data."""

    suggestions: list[Suggestion]


class SuggestionEngine:
    """Rule-based suggestion engine for workflow inputs."""

    def suggest(
        self,
        analysis: AnalysisSummary,
        comparison: ComparisonSummary | None = None,
    ) -> SuggestionSummary:
        """Generate suggestions from analysis and comparison summaries."""
        suggestions: list[Suggestion] = []

        suggestions.extend(self._suggest_from_success_rate(analysis))
        suggestions.extend(self._suggest_from_metric_stats(analysis))
        suggestions.extend(self._suggest_from_record_count(analysis))

        if comparison is not None:
            suggestions.extend(self._suggest_from_comparison(comparison))

        if not suggestions:
            suggestions.append(
                Suggestion(
                    title="No changes recommended",
                    rationale="Analysis did not detect actionable issues.",
                    priority="low",
                )
            )

        return SuggestionSummary(suggestions=suggestions)

    def _suggest_from_success_rate(self, analysis: AnalysisSummary) -> list[Suggestion]:
        """Generate suggestions based on success rate."""
        if analysis.success_rate is None:
            return []
        if analysis.success_rate >= 1.0:
            return []
        return [
            Suggestion(
                title="Investigate failing runs",
                rationale="Some runs reported failures. Review inputs and logs.",
                priority="high",
                suggested_change={"action": "review_failures"},
            )
        ]

    def _suggest_from_metric_stats(self, analysis: AnalysisSummary) -> list[Suggestion]:
        """Generate suggestions based on metric statistics."""
        suggestions: list[Suggestion] = []
        for metric, stats in analysis.metric_stats.items():
            if stats["p95"] > stats["mean"] * 1.5:
                suggestions.append(
                    Suggestion(
                        title=f"Reduce variability for {metric}",
                        rationale="High p95 relative to mean suggests variability.",
                        priority="medium",
                        suggested_change={
                            "metric": metric,
                            "target": "stability",
                        },
                    )
                )
        return suggestions

    def _suggest_from_record_count(self, analysis: AnalysisSummary) -> list[Suggestion]:
        """Generate suggestions when record data is missing."""
        if analysis.record_count > 0:
            return []
        return [
            Suggestion(
                title="Ensure results include samples",
                rationale="No tabular records found for deeper analysis.",
                priority="low",
                suggested_change={"action": "enable_sampling"},
            )
        ]

    def _suggest_from_comparison(
        self, comparison: ComparisonSummary
    ) -> list[Suggestion]:
        """Generate suggestions based on comparison rankings."""
        suggestions: list[Suggestion] = []
        for metric, ranking in comparison.rankings.items():
            if not ranking:
                continue
            best_run, best_value = ranking[0]
            suggestions.append(
                Suggestion(
                    title=f"Align with best {metric}",
                    rationale=f"Run {best_run} performs best for {metric}.",
                    priority="medium",
                    suggested_change={
                        "metric": metric,
                        "best_run": best_run,
                        "best_value": best_value,
                    },
                )
            )
        return suggestions
