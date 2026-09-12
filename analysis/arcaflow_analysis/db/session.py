"""Database session setup for analysis history."""

from __future__ import annotations

from dataclasses import dataclass

from sqlalchemy import Engine, create_engine
from sqlalchemy.orm import Session, sessionmaker

from arcaflow_analysis.db.models import Base


@dataclass(frozen=True)
class DatabaseConfig:
    """Database configuration for the analysis history store."""

    url: str


def init_db(config: DatabaseConfig) -> sessionmaker[Session]:
    """Initialize the database and return a session factory."""
    engine = create_engine(config.url, future=True)
    Base.metadata.create_all(engine)
    return sessionmaker(bind=engine, future=True, expire_on_commit=False)


def create_engine_from_config(config: DatabaseConfig) -> Engine:
    """Create a SQLAlchemy engine for the provided config."""
    return create_engine(config.url, future=True)
