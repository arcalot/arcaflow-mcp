"""Tests for analysis server scaffolding."""

import logging

import pytest

from arcaflow_analysis.server.app import GrpcServer, ServerConfig, configure_logging


def test_server_config_defaults() -> None:
    """Ensure default server config values are set."""
    config = ServerConfig()
    assert config.host == "0.0.0.0"
    assert config.port == 50051


def test_grpc_server_start_raises() -> None:
    """Validate the placeholder server raises until implemented."""
    server = GrpcServer(ServerConfig())
    with pytest.raises(RuntimeError, match="not implemented yet"):
        server.start()


def test_configure_logging_sets_level(monkeypatch: pytest.MonkeyPatch) -> None:
    """Verify configure_logging respects ARCAFLOW_MCP_LOG_LEVEL."""
    monkeypatch.setenv("ARCAFLOW_MCP_LOG_LEVEL", "DEBUG")
    configure_logging()
    root_logger = logging.getLogger()
    assert root_logger.level == logging.DEBUG
