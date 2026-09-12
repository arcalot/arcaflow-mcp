"""Repository for analysis history."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any
from uuid import uuid4

from sqlalchemy import select
from sqlalchemy.orm import Session, sessionmaker

from arcaflow_analysis.db.models import RunRecord


@dataclass(frozen=True)
class RunSummary:
    """Summary view of a stored run."""

    run_id: str
    workflow_id: str
    created_at: datetime
    metrics: dict[str, Any]


class HistoryRepository:
    """Persist and query workflow run history."""

    def __init__(self, session_factory: sessionmaker[Session]) -> None:
        self._session_factory = session_factory

    def add_run(
        self,
        workflow_id: str,
        input_payload: dict[str, Any],
        metrics: dict[str, Any],
    ) -> RunSummary:
        """Persist a workflow run and return a summary view."""
        run_id = str(uuid4())
        created_at = datetime.now(tz=UTC)
        record = RunRecord(
            id=run_id,
            created_at=created_at,
            workflow_id=workflow_id,
            input_payload=input_payload,
            metrics=metrics,
        )
        with self._session_factory() as session:
            session.add(record)
            session.commit()
        return RunSummary(
            run_id=run_id,
            workflow_id=workflow_id,
            created_at=created_at,
            metrics=metrics,
        )

    def list_runs(self, workflow_id: str | None = None) -> list[RunSummary]:
        """List stored runs, optionally filtered by workflow ID."""
        stmt = select(RunRecord).order_by(RunRecord.created_at.desc())
        if workflow_id:
            stmt = stmt.where(RunRecord.workflow_id == workflow_id)
        with self._session_factory() as session:
            records = session.scalars(stmt).all()
        return [
            RunSummary(
                run_id=record.id,
                workflow_id=record.workflow_id,
                created_at=record.created_at,
                metrics=record.metrics,
            )
            for record in records
        ]

    def get_run(self, run_id: str) -> RunRecord | None:
        """Fetch a stored run by its ID."""
        with self._session_factory() as session:
            return session.get(RunRecord, run_id)
