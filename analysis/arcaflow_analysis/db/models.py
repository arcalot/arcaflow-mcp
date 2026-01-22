"""SQLAlchemy models for analysis history.

SQLAlchemy provides a lightweight ORM with SQLite/PostgreSQL parity, giving us
portable schema definitions, migrations later, and typed sessions for the
analysis history store.
"""

from __future__ import annotations

from datetime import datetime
from typing import Any

from sqlalchemy import DateTime, String
from sqlalchemy.dialects.sqlite import JSON as SQLiteJSON
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    """Base class for analysis history models."""


class RunRecord(Base):
    """Persisted workflow run record for analysis history."""

    __tablename__ = "analysis_runs"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        nullable=False,
    )
    workflow_id: Mapped[str] = mapped_column(String(255), nullable=False)
    input_payload: Mapped[dict[str, Any]] = mapped_column(SQLiteJSON, nullable=False)
    metrics: Mapped[dict[str, Any]] = mapped_column(SQLiteJSON, nullable=False)
