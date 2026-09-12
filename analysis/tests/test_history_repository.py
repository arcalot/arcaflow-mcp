"""Tests for analysis history repository."""

from __future__ import annotations

# pylint: disable=import-error
# mypy: disable-error-code=arg-type

from arcaflow_analysis.db.repository import HistoryRepository
from arcaflow_analysis.db.session import DatabaseConfig, init_db


def test_add_and_get_run(tmp_path) -> None:
    db_url = f"sqlite:///{tmp_path}/history.db"
    session_factory = init_db(DatabaseConfig(url=db_url))
    repo = HistoryRepository(session_factory)

    summary = repo.add_run(
        workflow_id="workflow-a",
        input_payload={"input": "value"},
        metrics={"latency": 5},
    )

    stored = repo.get_run(summary.run_id)

    assert stored is not None
    assert stored.workflow_id == "workflow-a"
    assert stored.metrics["latency"] == 5


def test_list_runs_filtered(tmp_path) -> None:
    db_url = f"sqlite:///{tmp_path}/history.db"
    session_factory = init_db(DatabaseConfig(url=db_url))
    repo = HistoryRepository(session_factory)

    repo.add_run(
        workflow_id="workflow-a",
        input_payload={"input": "value"},
        metrics={"latency": 5},
    )
    repo.add_run(
        workflow_id="workflow-b",
        input_payload={"input": "value"},
        metrics={"latency": 7},
    )

    runs = repo.list_runs("workflow-a")

    assert len(runs) == 1
    assert runs[0].workflow_id == "workflow-a"
